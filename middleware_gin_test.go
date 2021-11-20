package xratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMiddlewareGin(t *testing.T) {
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

	mg := NewMiddlewareGin(rl, WithIpAddressGin("middleware-gin-test-ip"))

	router := gin.New()
	router.Use(mg.Handler())
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "testing middleware_gin...")
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
