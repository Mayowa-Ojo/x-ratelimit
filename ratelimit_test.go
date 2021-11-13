package xratelimit

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGetItem(t *testing.T) {
	redis := NewRedisStore()

	ret, err := redis.GetItem(context.Background(), "test-key")
	if err != nil {
		t.Errorf("Expected to return value at key, got: %s", err.Error())
	}

	if ret == nil {
		t.Logf("Expected return value to be nil, got: %+v", ret)
	}
}

func TestSetItem(t *testing.T) {
	redis := NewRedisStore()

	payload := &RequestLog{
		Timestamp: time.Now(),
		Counter:   1,
	}

	key := "127.0.0.1"

	if err := redis.SetItem(context.Background(), key, payload); err != nil {
		t.Errorf("Expected err to be nil, instead got: %s", err.Error())
	}

	data, err := redis.GetItem(context.Background(), key)
	if err != nil {
		t.Errorf("Expected <GetItem> to return data, instead got: %s", err.Error())
	}

	if data.Counter != 1 {
		t.Errorf("Expected <data.Remaining> to be '1', instead got: %d", data.Counter)
	}

	t.Cleanup(func() {
		if err := redis.DeleteItem(context.Background(), key); err != nil {
			t.Errorf("Error occured in clean-up function")
		}
	})
}

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
}
