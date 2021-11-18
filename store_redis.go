package xratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
)

const RedisAddr = "localhost:6379"

// Redis store
type RedisStore struct {
	client    *redis.Client
	namespace string
	ttl       time.Duration
}

func NewRedisStore() *RedisStore {
	redis := redis.NewClient(&redis.Options{
		Addr:        RedisAddr,
		Password:    "",
		DB:          0,
		MaxRetries:  10,
		DialTimeout: 15 * time.Second,
	})

	return &RedisStore{
		client:    redis,
		namespace: "x-ratelimit",
		ttl:       0,
	}
}

func (s *RedisStore) GetItem(ctx context.Context, key string) (*RequestLog, error) {
	var log *RequestLog
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	val, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(val), &log); err != nil {
		return nil, err
	}

	return log, nil
}

func (s *RedisStore) SetItem(ctx context.Context, key string, payload *RequestLog) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := s.client.Set(ctx, key, b, s.ttl).Err(); err != nil {
		return err
	}

	return nil
}

func (s *RedisStore) DeleteItem(ctx context.Context, key string) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return err
	}

	return nil
}
