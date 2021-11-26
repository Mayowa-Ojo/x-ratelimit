package xratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestConsume(t *testing.T) {
	var w sync.WaitGroup

	redis := NewRedisStore()
	rl := New(redis, RateLimitConfig{
		Duration: time.Second * 60,
		Limit:    10,
	})

	key := "127.0.0.1"

	for i := 0; i < 9; i++ {
		w.Add(1)
		go func(index int) {
			defer w.Done()

			time.Sleep(time.Second * (1 + time.Duration(index)))

			_, err := rl.Consume(context.Background(), key)
			if err != nil {
				t.Errorf("Expected err to be nil, instead got: %s", err.Error())
			}
		}(i)
	}

	w.Wait()

	t.Cleanup(func() {
		if err := redis.DeleteItem(context.Background(), key); err != nil {
			t.Errorf("Error occured in clean-up function")
		}
	})
}
