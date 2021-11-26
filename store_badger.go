package xratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v3"
)

const (
	BadgerPath = "/tmp/badger"
	BadgerTTL  = 18000
)

type BadgerStore struct {
	client    *badger.DB
	namespace string
	ttl       time.Duration
	path      string
}

type OptionBadger func(*BadgerStore)

func NewBadgerStore(options ...OptionBadger) (*BadgerStore, error) {
	bs := &BadgerStore{
		client:    nil,
		namespace: "x-ratelimit",
		ttl:       time.Second * BadgerTTL,
		path:      BadgerPath,
	}

	db, err := badger.Open(badger.DefaultOptions(BadgerPath))
	if err != nil {
		return nil, err
	}

	bs.client = db

	for _, opt := range options {
		opt(bs)
	}

	return bs, nil
}

func WithPath(path string) OptionBadger {
	return func(bs *BadgerStore) {
		bs.path = path
	}
}

func (s *BadgerStore) GetItem(ctx context.Context, key string) (*RequestLog, error) {
	var log *RequestLog
	var copy []byte
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	err := s.client.View(func(txn *badger.Txn) error {
		v, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = v.Value(func(val []byte) error {
			copy = append([]byte{}, val...)
			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(copy, &log); err != nil {
		return nil, err
	}

	return log, nil
}

func (s *BadgerStore) SetItem(ctx context.Context, key string, payload *RequestLog) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = s.client.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), b)

		err := txn.SetEntry(entry)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *BadgerStore) DeleteItem(ctx context.Context, key string) error {
	key = fmt.Sprintf("%s:%s", s.namespace, key)

	txn := s.client.NewTransaction(true)
	defer txn.Discard()

	if err := txn.Delete([]byte(key)); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	return nil
}
