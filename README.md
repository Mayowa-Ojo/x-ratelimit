## X-Ratelimit

Exploring rate-limiting techniques with a simple implementation of the fixed-window algorithm. This algorithm uses a window size (n seconds) to track the fixed-window algorithm rate. There is a counter that is incremented on each request, and a request can be discarded if it exceeds a set threshold (max requests / n seconds) within the timeframe.

> One drawback of this algorithm is that the system can be overloaded at the boundary of the window.

Storage options include:
- [x] Redis
- [x] BadgerDB
- [x] In-Memory

Middleware implementations include:
- [x] Standard Lib
- [x] Gin
- [x] Echo
- [x] Fasthttp
- [ ] Fibre

#### Example
```go
package main

import (
   "log"
   "time"
   "net/http"

   limiter "github.com/Mayowa-Ojo/x-ratelimit"
)

var (
   duration = time.Second * 60
   limit = 10
)

func main() {
   mux = http.NewServeMux()

	handler := func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Hello World..."))
	}

	redis := limiter.NewRedisStore()

	rl := limiter.New(redis, RateLimitConfig{
		Duration: duration,
		Limit:    limit,
	})

	ms := limiter.NewMiddlewareStd(rl, limiter.WithIpAddressStd("ip-address")).Handler(handler)

   mux.Handle("/", ms.Handler(http.HandlerFunc(handler)))

   log.Fatal(http.ListenAndServe(":8080", mux))
}
```