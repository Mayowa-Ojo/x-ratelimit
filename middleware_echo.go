package xratelimit

import (
	"context"
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
)

type MiddlewareEcho struct {
	*RateLimit
	OnError         ErrMiddlewareHandler
	OnLimitExceeded RateLimitExceededHandler
	IpAddress       string
}

type OptionEcho func(*MiddlewareEcho)

func NewMiddlewareEcho(rl *RateLimit, options ...OptionEcho) *MiddlewareEcho {
	mw := &MiddlewareEcho{
		RateLimit:       rl,
		OnError:         DefaultErrMiddlewareHandler,
		OnLimitExceeded: DefaultRateLimitExceededHandler,
	}

	for _, opt := range options {
		opt(mw)
	}

	return mw
}

func WithOnErrorEcho(onError ErrMiddlewareHandler) OptionEcho {
	return func(mw *MiddlewareEcho) {
		mw.OnError = onError
	}
}

func WithOnLimitExceededEcho(onLimitExceeded RateLimitExceededHandler) OptionEcho {
	return func(mw *MiddlewareEcho) {
		mw.OnLimitExceeded = onLimitExceeded
	}
}

func WithIpAddressEcho(ip string) OptionEcho {
	return func(mw *MiddlewareEcho) {
		mw.IpAddress = ip
	}
}

func (mw *MiddlewareEcho) Handler(h echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var key string
		// ctx := c.(context.Context)

		if mw.IpAddress == "" {
			k, err := mw.GetIp(c)
			if err != nil {
				mw.OnError(c.Response(), c.Request(), err)
				return nil
			}

			key = k
		} else {
			key = mw.IpAddress
		}

		_, err := mw.RateLimit.Consume(context.Background(), key)
		if err != nil {
			if err == ErrRateLimitExceeded {
				mw.OnLimitExceeded(c.Response(), c.Request())
				return nil
			}

			mw.OnError(c.Response(), c.Request(), err)
			return nil
		}

		remaining, err := mw.RateLimit.Remaining(context.Background(), key)
		if err != nil {
			mw.OnError(c.Response(), c.Request(), err)
			return nil
		}

		res := c.Response()

		res.Header().Add("X-Ratelimit-Limit", fmt.Sprint(mw.RateLimit.RateLimitConfig.Limit))
		res.Header().Add("X-Ratelimit-Remaining", fmt.Sprint(*remaining))

		return h(c)
	}
}

func (mw *MiddlewareEcho) GetIp(ctx echo.Context) (string, error) {
	// r := ctx.Request
	extractor := echo.ExtractIPFromXFFHeader()
	ip := extractor(ctx.Request())

	if ip != "" {
		return ip, nil
	}

	extractor = echo.ExtractIPFromRealIPHeader()
	ip = extractor(ctx.Request())

	if ip != "" {
		return ip, nil
	}

	return "", errors.New("ip not found")
}
