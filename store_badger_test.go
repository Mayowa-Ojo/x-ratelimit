package xratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetSetItem(t *testing.T) {
	is := require.New(t)

	badger, err := NewBadgerStore()
	is.NoError(err)

	payload := &RequestLog{
		Timestamp: time.Now(),
		Counter:   1,
	}

	key := "127.0.0.1"

	err = badger.SetItem(context.Background(), key, payload)
	is.NoError(err)

	log, err := badger.GetItem(context.Background(), key)
	is.NoError(err)
	is.NotNil(log)

	if log.Counter != 1 {
		t.Errorf("Expected value of 'counter' to be %d, instead got %d", payload.Counter, log.Counter)
	}

	t.Cleanup(func() {
		err := badger.DeleteItem(context.Background(), key)
		is.NoError(err)

		badger.client.Close()
	})
}
