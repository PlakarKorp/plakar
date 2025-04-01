package caching

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"iter"
	"syscall"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/cockroachdb/pebble/v2"
)

type PebbleCache struct {
	db *pebble.DB
}

// Lifter from internal/logger pebble.
type NoopLoggerAndTracer struct{}

func (l NoopLoggerAndTracer) Infof(format string, args ...interface{})                       {}
func (l NoopLoggerAndTracer) Errorf(format string, args ...interface{})                      {}
func (l NoopLoggerAndTracer) Fatalf(format string, args ...interface{})                      {}
func (l NoopLoggerAndTracer) Eventf(ctx context.Context, format string, args ...interface{}) {}
func (l NoopLoggerAndTracer) IsTracingEnabled(ctx context.Context) bool                      { return false }

func New(dir string) (*PebbleCache, error) {
	opts := pebble.Options{
		MemTableSize: 256 << 20,
		Logger:       NoopLoggerAndTracer{},
	}
	db, err := pebble.Open(dir, &opts)
	if err != nil {
		if errors.Is(err, syscall.EAGAIN) {
			return nil, ErrInUse
		}
		return nil, err
	}

	return &PebbleCache{db: db}, nil
}

func (p *PebbleCache) put(prefix, key string, data []byte) error {
	return p.db.Set([]byte(fmt.Sprintf("%s:%s", prefix, key)), data, pebble.NoSync)
}

func (p *PebbleCache) has(prefix, key string) (bool, error) {
	_, del, err := p.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))
	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	del.Close()

	return true, nil
}

func (p *PebbleCache) get(prefix, key string) ([]byte, error) {
	data, del, err := p.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	ret := make([]byte, len(data))
	copy(ret, data)
	del.Close()

	return ret, nil
}

// Iterate over keys sharing `keyPrefix` prefix. Extracts the MAC out of the
// key rather than having to unmarshal the value. Beware you can't hold a
// reference to the value between call to Next().
func (p *PebbleCache) getObjectsWithMAC(keyPrefix string) iter.Seq2[objects.MAC, []byte] {
	return func(yield func(objects.MAC, []byte) bool) {
		// It's safe to ignore the error here, the implementation always return nil
		iter, _ := p.db.NewIter(MakePrefixIterIterOptions([]byte(keyPrefix)))
		defer iter.Close()

		for iter.First(); iter.Valid(); iter.Next() {
			/* Extract the csum part of the key, this avoids decoding the full
			 * entry later on if that's the only thing we need */
			key := iter.Key()
			hex_csum := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
			csum, _ := hex.DecodeString(hex_csum)

			if !yield(objects.MAC(csum), iter.Value()) {
				return
			}
		}
	}
}

// Iterate over keys sharing `keyPrefix` prefix. Beware you can't hold a
// reference to the value between call to Next().
func (p *PebbleCache) getObjects(keyPrefix string) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		iter, _ := p.db.NewIter(MakePrefixIterIterOptions([]byte(keyPrefix)))
		defer iter.Close()

		for iter.First(); iter.Valid(); iter.Next() {
			if !yield(iter.Value()) {
				return
			}
		}
	}
}

func (p *PebbleCache) delete(prefix, key string) error {
	return p.db.Delete([]byte(fmt.Sprintf("%s:%s", prefix, key)), nil)
}

func (p *PebbleCache) Close() error {
	return p.db.Close()
}

func MakeKeyUpperBound(key []byte) []byte {
	end := make([]byte, len(key))
	copy(end, key)

	for i := len(end) - 1; i >= 0; i-- {
		end[i] = end[i] + 1
		if end[i] != 0 {
			return end[:i+1]
		}
	}

	return nil // no upper-bound
}

func MakePrefixIterIterOptions(prefix []byte) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: MakeKeyUpperBound(prefix),
	}
}
