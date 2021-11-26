package xratelimit

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetSet(t *testing.T) {
	is := require.New(t)

	ms := NewMemoryStore()
	is.NotNil(ms)

	payload := &RequestLog{
		Timestamp: time.Now(),
		Counter:   1,
	}

	key := "127.0.0.1"

	err := ms.SetItem(context.Background(), key, payload)
	is.NoError(err)

	for i := 0; i < 10; i++ {
		key = fmt.Sprintf("%s.%s", key, strconv.Itoa(i))
		err := ms.SetItem(context.Background(), key, payload)
		is.NoError(err)
	}

	log, err := ms.GetItem(context.Background(), key)
	is.NoError(err)
	is.NotNil(log)

	if log.Counter != 1 {
		t.Errorf("expected 'counter' to be 1, instead got: %d", log.Counter)
	}

	if math.Signbit(float64(time.Now().Sub(log.Timestamp))) {
		t.Errorf("expected log.Timestamp to be before current time")
	}

	if ms.logs.length != 11 {
		t.Errorf("expected 'logs.length' to be 10, instead got: %d", ms.logs.length)
	}
}
