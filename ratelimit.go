package xratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	redis "github.com/go-redis/redis/v8"
)

const RedisAddr = "localhost:6379"

var ErrRateLimitExceeded = errors.New("client has exceeded rate limit for given period")

// Redis store
type RedisStore struct {
	client    *redis.Client
	namespace string
	ttl       time.Duration
}

func NewRedisStore() *RedisStore {
	redis := redis.NewClient(&redis.Options{
		Addr:        RedisAddr,
		Password:    "",
		DB:          0,
		MaxRetries:  10,
		DialTimeout: 15 * time.Second,
	})

	return &RedisStore{
		client:    redis,
		namespace: "x-ratelimit",
		ttl:       0,
	}
}

func (s *RedisStore) GetItem(ctx context.Context, key string) (*RequestLog, error) {
	var log *RequestLog
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(val), &log); err != nil {
		return nil, err
	}

	return log, nil
}

func (s *RedisStore) SetItem(ctx context.Context, key string, payload *RequestLog) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := s.client.Set(ctx, key, b, s.ttl).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStore) DeleteItem(ctx context.Context, key string) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}

type RateLimitConfig struct {
	Duration  time.Duration
	Limit     int
	Skip      func() bool
	Whitelist []string
}

type RateLimit struct {
	RateLimitConfig
	Store
	m sync.Mutex
}

type Store interface {
	GetItem(ctx context.Context, key string) (*RequestLog, error)
	SetItem(ctx context.Context, key string, payload *RequestLog) error
	DeleteItem(ctx context.Context, key string) error
}

type RequestLog struct {
	Timestamp time.Time
	Counter   int
}

func New(store Store, config RateLimitConfig) *RateLimit {
	return &RateLimit{
		RateLimitConfig: config,
		Store:           store,
	}
}

func (c *RateLimit) Consume(ctx context.Context, key string) (*RequestLog, error) {
	c.m.Lock()
	defer c.m.Unlock()

	var payload RequestLog

	rlog, err := c.Store.GetItem(ctx, key)
	if err != nil {
		// check nil error
		if strings.Split(err.Error(), ": ")[1] != "nil" {
			return nil, err
		}

		payload.Timestamp = time.Now()
		payload.Counter = 1

		if err := c.Store.SetItem(ctx, key, &payload); err != nil {
			return nil, err
		}

		rlog, err := c.Store.GetItem(ctx, key)
		if err != nil {
			return nil, err
		}

		return rlog, nil
	}

	if time.Since(rlog.Timestamp) >= c.RateLimitConfig.Duration {
		// Reset counter
		return c.Reset(ctx, key)
	}

	if rlog.Counter >= c.RateLimitConfig.Limit {
		return nil, ErrRateLimitExceeded
	}

	payload.Timestamp = rlog.Timestamp
	payload.Counter = rlog.Counter + 1

	if err := c.Store.SetItem(ctx, key, &payload); err != nil {
		return nil, err
	}

	rlog, err = c.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return rlog, nil
}

func (c *RateLimit) Remaining(ctx context.Context, key string) (*int, error) {
	rlog, err := c.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return &rlog.Counter, nil
}

func (c *RateLimit) Reset(ctx context.Context, key string) (*RequestLog, error) {
	var payload RequestLog

	payload.Timestamp = time.Now()
	payload.Counter = 1

	if err := c.Store.SetItem(ctx, key, &payload); err != nil {
		return nil, err
	}

	rlog, err := c.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return rlog, nil
}

func (c *RateLimit) GetIp(r *http.Request) (string, error) {
	ip := r.Header.Get("x-real-ip")
	netIp := net.ParseIP(ip)
	if netIp != nil {
		return netIp.String(), nil
	}

	ips := r.Header.Get("x-fowarded-for")
	ipSlice := strings.Split(ips, ",")

	for _, v := range ipSlice {
		netIp := net.ParseIP(v)
		if netIp != nil {
			return netIp.String(), nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	netIp = net.ParseIP(ip)
	if netIp != nil {
		return netIp.String(), nil
	}

	return "", errors.New("ip not found")
}
