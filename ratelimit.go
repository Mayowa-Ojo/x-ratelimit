package xratelimit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimitConfig struct {
	Duration  time.Duration
	Limit     int
	Skip      func(rw http.Request, r *http.Request) bool // cond for a request to be skipped
	Whitelist []string                                    // whitelisted ips
}

type RateLimit struct {
	RateLimitConfig
	Store
	m sync.Mutex
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
