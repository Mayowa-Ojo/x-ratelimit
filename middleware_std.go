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

type OptionStd func(*MiddlewareStd)

func NewMiddlewareStd(rl *RateLimit, options ...OptionStd) *MiddlewareStd {
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

func WithOnErrorStd(onError ErrMiddlewareHandler) OptionStd {
	return func(ms *MiddlewareStd) {
		ms.OnError = onError
	}
}

func WithOnLimitExceededStd(onLimitExceeded RateLimitExceededHandler) OptionStd {
	return func(ms *MiddlewareStd) {
		ms.OnLimitExceeded = onLimitExceeded
	}
}

func WithIpAddressStd(ip string) OptionStd {
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

		if m.RateLimit.Skip != nil && m.RateLimit.Skip(rw, r) {
			h.ServeHTTP(rw, r)
			return
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
