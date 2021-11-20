package xratelimit

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type MiddlewareGin struct {
	*RateLimit
	OnError         ErrMiddlewareHandler
	OnLimitExceeded RateLimitExceededHandler
	IpAddress       string
}

type OptionGin func(*MiddlewareGin)

func NewMiddlewareGin(rl *RateLimit, options ...OptionGin) *MiddlewareGin {
	ms := &MiddlewareGin{
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

func WithOnErrorGin(onError ErrMiddlewareHandler) OptionGin {
	return func(ms *MiddlewareGin) {
		ms.OnError = onError
	}
}

func WithOnLimitExceededGin(onLimitExceeded RateLimitExceededHandler) OptionGin {
	return func(ms *MiddlewareGin) {
		ms.OnLimitExceeded = onLimitExceeded
	}
}

func WithIpAddressGin(ip string) OptionGin {
	return func(ms *MiddlewareGin) {
		ms.IpAddress = ip
	}
}

func (mg *MiddlewareGin) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var key string

		if mg.IpAddress == "" {
			k, err := mg.RateLimit.GetIp(ctx.Request)
			if err != nil {
				mg.OnError(ctx.Writer, ctx.Request, err)
				return
			}

			key = k
		} else {
			key = mg.IpAddress
		}

		if _, err := mg.RateLimit.Consume(ctx, key); err != nil {
			if err == ErrRateLimitExceeded {
				mg.OnLimitExceeded(ctx.Writer, ctx.Request)
				return
			}

			mg.OnError(ctx.Writer, ctx.Request, err)
			return
		}

		remaining, err := mg.RateLimit.Remaining(ctx, key)
		if err != nil {
			mg.OnError(ctx.Writer, ctx.Request, err)
			return
		}

		ctx.Request.Header.Add("X-Ratelimit-Limit", fmt.Sprint(mg.RateLimit.RateLimitConfig.Limit))
		ctx.Request.Header.Add("X-Ratelimit-Remaining", fmt.Sprint(remaining))

		ctx.Next()
	}
}
