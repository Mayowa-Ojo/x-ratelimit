package xratelimit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	MemoryTTL        = time.Second * 3600
	FNVOffsetBasis   = uint64(14695981039346656037)
	FNVPrime         = 1099511628211
	InitialArraySize = 1024
)

var ErrHashKeyNotFound = errors.New("key not found in hash table")

type HashTable struct {
	entries []*HashTableEntry
	length  int // number of items in array
}

type HashTableEntry struct {
	key   []byte
	value []byte
}

type MemoryStore struct {
	namespace string
	ttl       time.Duration
	logs      *HashTable
	sync.Mutex
}

type OptionMemory func(*MemoryStore)

func NewHashTable() *HashTable {
	h := &HashTable{
		entries: make([]*HashTableEntry, InitialArraySize),
	}

	for i := range h.entries {
		h.entries[i] = &HashTableEntry{}
	}

	return h
}

func NewMemoryStore(options ...OptionMemory) *MemoryStore {
	logs := NewHashTable()

	ms := &MemoryStore{
		namespace: "x-ratelimit",
		ttl:       MemoryTTL,
		logs:      logs,
	}

	for _, opt := range options {
		opt(ms)
	}

	return ms
}

func WithTTL(ttl time.Duration) OptionMemory {
	return func(ms *MemoryStore) {
		ms.ttl = ttl
	}
}

func (ms *MemoryStore) GetItem(ctx context.Context, key string) (*RequestLog, error) {
	key = fmt.Sprintf("%s:%s", ms.namespace, key)
	entries := ms.logs.entries
	capacity := cap(ms.logs.entries)
	index := ms.hashKey(key, capacity)
	var log *RequestLog

	for entries[index].key != nil {
		if strings.EqualFold(string(entries[index].key), key) {
			if err := json.Unmarshal(entries[index].value, &log); err != nil {
				return nil, err
			}

			return log, nil
		}

		index++ // start linear probbing

		if index >= len(entries) {
			index = 0
		}
	}

	return nil, ErrHashKeyNotFound
}

func (ms *MemoryStore) SetItem(ctx context.Context, key string, payload *RequestLog) error {
	ms.Lock()
	defer ms.Unlock()

	key = fmt.Sprintf("%s:%s", ms.namespace, key)
	capacity := cap(ms.logs.entries)
	index := ms.hashKey(key, capacity)

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if ms.logs.length >= (capacity / 2) {
		if err := ms.expand(ctx); err != nil {
			return err
		}

		capacity = cap(ms.logs.entries)
		index = ms.hashKey(key, capacity)
	}

	entry := new(HashTableEntry)
	entry.key = []byte(key)
	entry.value = b

	for ms.logs.entries[index].key != nil {
		if strings.EqualFold(string(ms.logs.entries[index].key), key) {
			// key already exists
			ms.logs.entries[index] = entry

			ms.logs.length++
			return nil
		}

		index++ // start linear probbing

		if index >= len(ms.logs.entries) {
			index = 0
		}
	}

	ms.logs.entries[index] = entry
	ms.logs.length++

	return nil
}

func (ms *MemoryStore) DeleteItem(ctx context.Context, key string) error {
	key = fmt.Sprintf("%s:%s", ms.namespace, key)
	capacity := cap(ms.logs.entries)
	index := ms.hashKey(key, capacity)
	entries := ms.logs.entries

	for entries[index].key != nil {
		if strings.EqualFold(string(entries[index].key), key) {
			entries[index].key = nil
			entries[index].value = nil

			return nil
		}

		index++ // start linear probbing

		if index >= len(entries) {
			index = 0
		}
	}

	return ErrHashKeyNotFound
}

func (ms *MemoryStore) hashKey(key string, capacity int) int {
	b := []byte(key)
	h := FNVOffsetBasis

	for _, v := range b {
		h ^= uint64(v)
		h *= uint64(FNVPrime)
	}

	return int(h % uint64(capacity))
}

func (ms *MemoryStore) expand(ctx context.Context) error {
	entries := ms.logs.entries
	capacity := cap(entries)

	nEntries := make([]*HashTableEntry, capacity*2)

	ms.logs.entries = nEntries

	for _, v := range entries {
		if v.key != nil {
			var log *RequestLog

			if err := json.Unmarshal(v.value, &log); err != nil {
				return err
			}

			if err := ms.SetItem(ctx, string(v.key), log); err != nil {
				return err
			}
		}
	}

	return nil
}
