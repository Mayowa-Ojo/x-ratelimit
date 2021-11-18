package xratelimit

import "context"

type Store interface {
	GetItem(ctx context.Context, key string) (*RequestLog, error)
	SetItem(ctx context.Context, key string, payload *RequestLog) error
	DeleteItem(ctx context.Context, key string) error
}
