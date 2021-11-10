package xratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	redis "github.com/go-redis/redis/v8"
)

const RedisAddr = "localhost:6379"

var ErrRateLimitExceeded = errors.New("client has exceeded rate limit for given period")

type RateLimit struct {
	Config
	Store
}

type Store interface {
	GetItem(key string) string
	SetItem()
	DeleteItem()
}

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

type RequestLog struct {
	Timestamp time.Time
	Remaining int
}

type Config struct {
	Duration  time.Duration
	Limit     int
	Store     interface{}
	Skip      func() bool
	Whitelist []string
}

func (c *RateLimit) New(store Store, config Config) {

}

func (c *RateLimit) Incr(key string) (*RequestLog, error) {
	// entry := c.store.GetItem(key)
	return nil, nil
}

func (c *RateLimit) Remaining() {

}

func (c *RateLimit) Reset() {

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

	return "", errors.New("pp not found")
}

// func Test() {
// 	c := &RateLimit{}
// 	redis := &RedisStore{}
// 	config := Config{}

// 	c.New(redis, config)
// }
