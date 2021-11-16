package xratelimit

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/valyala/fasthttp"
)

type OnError = func(rw fasthttp.Response, r fasthttp.Request, e error)

type MiddlewareFasthttp struct {
	*RateLimit
	OnError         interface{}
	OnLimitExceeded RateLimitExceededHandler
	IpAddress       string
}

type OptionFasthttp func(*MiddlewareFasthttp)

func NewMiddlewareFasthttp(rl *RateLimit, options ...OptionFasthttp) *MiddlewareFasthttp {
	mw := &MiddlewareFasthttp{
		RateLimit:       rl,
		OnError:         DefaultErrMiddlewareHandler,
		OnLimitExceeded: DefaultRateLimitExceededHandler,
	}

	for _, opt := range options {
		opt(mw)
	}

	return mw
}

func WithOnErrorFasthttp(onError ErrMiddlewareHandler) OptionFasthttp {
	return func(mw *MiddlewareFasthttp) {
		mw.OnError = onError
	}
}

func WithOnLimitExceededFasthttp(onLimitExceeded RateLimitExceededHandler) OptionFasthttp {
	return func(mw *MiddlewareFasthttp) {
		mw.OnLimitExceeded = onLimitExceeded
	}
}

func WithIpAddressFasthttp(ip string) OptionFasthttp {
	return func(mw *MiddlewareFasthttp) {
		mw.IpAddress = ip
	}
}

func (mw *MiddlewareFasthttp) Handler(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		var key string

		if mw.IpAddress == "" {
			k, err := mw.GetIp(ctx)
			if err != nil {
				mw.OnError.(OnError)(ctx.Response, ctx.Request, err)
				return
			}

			key = k
		} else {
			key = mw.IpAddress
		}

		_, err := mw.RateLimit.Consume(ctx, key)
		if err != nil {
			if err == ErrRateLimitExceeded {
				// mw.OnLimitExceeded(rw, r)
				return
			}

			// mw.OnError(rw, r, err)
			return
		}

		remaining, err := mw.RateLimit.Remaining(ctx, key)
		if err != nil {
			// mw.OnError(rw, r, err)
			return
		}

		ctx.Request.Header.Add("X-Ratelimit-Limit", fmt.Sprint(mw.RateLimit.RateLimitConfig.Limit))
		ctx.Request.Header.Add("X-Ratelimit-Remaining", fmt.Sprint(*remaining))

		h(ctx)
	}
}

func (mw *MiddlewareFasthttp) GetIp(ctx *fasthttp.RequestCtx) (string, error) {
	r := ctx.Request
	ip := fmt.Sprint(r.Header.Peek("x-real-ip"))
	netIp := net.ParseIP(ip)
	if netIp != nil {
		return netIp.String(), nil
	}

	ips := r.Header.Peek("x-fowarded-for")
	ipSlice := strings.Split(string(ips), ",")

	for _, v := range ipSlice {
		netIp := net.ParseIP(v)
		if netIp != nil {
			return netIp.String(), nil
		}
	}

	hostport := ctx.RemoteAddr().String()
	ip, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", err
	}

	netIp = net.ParseIP(ip)
	if netIp != nil {
		return netIp.String(), nil
	}

	return "", errors.New("ip not found")
}
