package xratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareEcho(t *testing.T) {
	is := require.New(t)
	path := "/"
	limit := 10
	duration := time.Second * 60
	numRequests := 12

	request, err := http.NewRequest("GET", path, nil)
	is.NoError(err)
	is.NotNil(request)

	redis := NewRedisStore()
	is.NotZero(redis)

	rl := New(redis, RateLimitConfig{
		Duration: duration,
		Limit:    limit,
	})
	is.NotZero(rl)

	mw := NewMiddlewareEcho(rl, WithIpAddressEcho("middleware-echo-test-ip"))

	router := echo.New()
	router.Use(mw.Handler)
	router.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "testing middleware_gin...")
	})

	for i := 0; i < numRequests; i++ {
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, request)

		if i <= (limit - 1) {
			is.Equal(http.StatusOK, resp.Code)
		} else {
			is.Equal(http.StatusTooManyRequests, resp.Code)
		}
	}
}
