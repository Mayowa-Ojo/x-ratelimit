package xratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMiddlewareStd(t *testing.T) {
	is := require.New(t)
	path := "/"
	limit := 10
	duration := time.Second * 60
	numRequests := 12

	request, err := http.NewRequest("GET", path, nil)
	is.NoError(err)
	is.NotNil(request)

	handler := http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("testing middleware_std..."))
	})

	redis := NewRedisStore()
	is.NotZero(redis)

	rl := New(redis, RateLimitConfig{
		Duration: duration,
		Limit:    limit,
	})
	is.NotZero(rl)

	ms := NewMiddlewareStd(rl, WithIpAddress("middleware-std-test-ip")).Handler(handler)
	is.NotZero(ms)

	for i := 0; i < numRequests; i++ {
		resp := httptest.NewRecorder()

		ms.ServeHTTP(resp, request)

		if i <= (limit - 1) {
			is.Equal(http.StatusOK, resp.Code)
		} else {
			is.Equal(http.StatusTooManyRequests, resp.Code)
		}
	}
}
