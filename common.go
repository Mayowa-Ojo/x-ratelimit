package xratelimit

import (
	"errors"
	"net/http"
)

type ErrMiddlewareHandler = func(rw http.ResponseWriter, r *http.Request, e error)
type RateLimitExceededHandler = func(rw http.ResponseWriter, r *http.Request)

var ErrRateLimitExceeded = errors.New("client has exceeded rate limit for given period")

func DefaultErrMiddlewareHandler(rw http.ResponseWriter, r *http.Request, e error) {
	http.Error(rw, e.Error(), http.StatusInternalServerError)
}

func DefaultRateLimitExceededHandler(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, ErrRateLimitExceeded.Error(), http.StatusTooManyRequests)
}
