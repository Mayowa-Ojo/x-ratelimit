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
	Skip      func(rw http.ResponseWriter, r *http.Request) bool // cond for a request to be skipped
	Whitelist []string                                           // whitelisted ips
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

func (rl *RateLimit) Consume(ctx context.Context, key string) (*RequestLog, error) {
	rl.m.Lock()
	defer rl.m.Unlock()

	var payload RequestLog

	if rl.isWhitelistedIp(key) {
		return nil, nil
	}

	rlog, err := rl.Store.GetItem(ctx, key)
	if err != nil {
		// check nil error
		if strings.Split(err.Error(), ": ")[1] != "nil" {
			return nil, err
		}

		payload.Timestamp = time.Now()
		payload.Counter = 1

		if err := rl.Store.SetItem(ctx, key, &payload); err != nil {
			return nil, err
		}

		rlog, err := rl.Store.GetItem(ctx, key)
		if err != nil {
			return nil, err
		}

		return rlog, nil
	}

	if time.Since(rlog.Timestamp) >= rl.RateLimitConfig.Duration {
		// Reset counter
		return rl.Reset(ctx, key)
	}

	if rlog.Counter >= rl.RateLimitConfig.Limit {
		return nil, ErrRateLimitExceeded
	}

	payload.Timestamp = rlog.Timestamp
	payload.Counter = rlog.Counter + 1

	if err := rl.Store.SetItem(ctx, key, &payload); err != nil {
		return nil, err
	}

	rlog, err = rl.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return rlog, nil
}

func (rl *RateLimit) Remaining(ctx context.Context, key string) (*int, error) {
	rlog, err := rl.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return &rlog.Counter, nil
}

func (rl *RateLimit) Reset(ctx context.Context, key string) (*RequestLog, error) {
	var payload RequestLog

	payload.Timestamp = time.Now()
	payload.Counter = 1

	if err := rl.Store.SetItem(ctx, key, &payload); err != nil {
		return nil, err
	}

	rlog, err := rl.Store.GetItem(ctx, key)
	if err != nil {
		return nil, err
	}

	return rlog, nil
}

func (rl *RateLimit) GetIp(r *http.Request) (string, error) {
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

func (rl *RateLimit) isWhitelistedIp(ip string) bool {
	for _, v := range rl.Whitelist {
		if strings.EqualFold(v, ip) {
			return true
		}
	}

	return false
}
