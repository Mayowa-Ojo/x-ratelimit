package xratelimit

import (
	"fmt"
	"net/http"
)

type MiddlewareStd struct {
	*RateLimit
	OnError         ErrMiddlewareHandler
	OnLimitExceeded RateLimitExceededHandler
	IpAddress       string
}

type Option func(*MiddlewareStd)

func NewMiddlewareStd(rl *RateLimit, options ...Option) *MiddlewareStd {
	ms := &MiddlewareStd{
		RateLimit:       rl,
		OnError:         DefaultErrMiddlewareHandler,
		OnLimitExceeded: DefaultRateLimitExceededHandler,
	}

	// functional options pattern
	for _, opt := range options {
		opt(ms)
	}

	return ms
}

func WithOnError(onError ErrMiddlewareHandler) Option {
	return func(ms *MiddlewareStd) {
		ms.OnError = onError
	}
}

func WithOnLimitExceeded(onLimitExceeded RateLimitExceededHandler) Option {
	return func(ms *MiddlewareStd) {
		ms.OnLimitExceeded = onLimitExceeded
	}
}

func WithIpAddress(ip string) Option {
	return func(ms *MiddlewareStd) {
		ms.IpAddress = ip
	}
}

func (m *MiddlewareStd) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var key string

		if m.IpAddress == "" {
			k, err := m.RateLimit.GetIp(r)
			if err != nil {
				m.OnError(rw, r, err)
				return
			}

			key = k
		} else {
			key = m.IpAddress
		}

		_, err := m.RateLimit.Consume(r.Context(), key)
		if err != nil {
			if err == ErrRateLimitExceeded {
				m.OnLimitExceeded(rw, r)
				return
			}

			m.OnError(rw, r, err)
			return
		}

		remaining, err := m.RateLimit.Remaining(r.Context(), key)
		if err != nil {
			m.OnError(rw, r, err)
			return
		}

		rw.Header().Add("X-Ratelimit-Limit", fmt.Sprint(m.RateLimit.RateLimitConfig.Limit))
		rw.Header().Add("X-Ratelimit-Remaining", fmt.Sprint(*remaining))

		h.ServeHTTP(rw, r)
	})
}
