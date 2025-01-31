package caching

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"iter"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/cockroachdb/pebble"
)

type ScanCache struct {
	snapshotID [32]byte
	manager    *Manager
	db         *pebble.DB
}

func newScanCache(cacheManager *Manager, snapshotID [32]byte) (*ScanCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "scan", fmt.Sprintf("%x", snapshotID))

	db, err := pebble.Open(cacheDir, nil)
	if err != nil {
		return nil, err
	}

	return &ScanCache{
		snapshotID: snapshotID,
		manager:    cacheManager,
		db:         db,
	}, nil
}

func (c *ScanCache) Close() error {
	c.db.Close()
	return os.RemoveAll(filepath.Join(c.manager.cacheDir, "scan", fmt.Sprintf("%x", c.snapshotID)))
}

func (c *ScanCache) put(prefix string, key string, data []byte) error {
	return c.db.Set([]byte(fmt.Sprintf("%s:%s", prefix, key)), data, nil)
}

func (c *ScanCache) get(prefix, key string) ([]byte, error) {
	val, closer, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	ret := make([]byte, len(val))
	copy(ret, val)
	closer.Close()
	return ret, nil
}

func (c *ScanCache) has(prefix, key string) (bool, error) {
	_, closer, err := c.db.Get([]byte(fmt.Sprintf("%s:%s", prefix, key)))
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if closer != nil {
		closer.Close()
	}
	return err == nil, err
}

func (c *ScanCache) delete(prefix, key string) error {
	return c.db.Delete([]byte(fmt.Sprintf("%s:%s", prefix, key)), nil)
}

func (c *ScanCache) PutFile(file string, data []byte) error {
	return c.put("__file__", file, data)
}

func (c *ScanCache) GetFile(file string) ([]byte, error) {
	return c.get("__file__", file)
}

func (c *ScanCache) PutDirectory(directory string, data []byte) error {
	return c.put("__directory__", directory, data)
}

func (c *ScanCache) GetDirectory(directory string) ([]byte, error) {
	return c.get("__directory__", directory)
}

func (c *ScanCache) PutChecksum(pathname string, checksum objects.Checksum) error {
	pathname = strings.TrimSuffix(pathname, "/")
	if pathname == "" {
		pathname = "/"
	}
	return c.put("__checksum__", pathname, checksum[:])
}

func (c *ScanCache) GetChecksum(pathname string) (objects.Checksum, error) {
	pathname = strings.TrimSuffix(pathname, "/")
	if pathname == "" {
		pathname = "/"
	}

	data, err := c.get("__checksum__", pathname)
	if err != nil {
		return objects.Checksum{}, err
	}

	if len(data) != 32 {
		return objects.Checksum{}, fmt.Errorf("invalid checksum length: %d", len(data))
	}

	return objects.Checksum(data), nil
}

func (c *ScanCache) PutSummary(pathname string, data []byte) error {
	pathname = strings.TrimSuffix(pathname, "/")
	if pathname == "" {
		pathname = "/"
	}

	return c.put("__summary__", pathname, data)
}

func (c *ScanCache) GetSummary(pathname string) ([]byte, error) {
	pathname = strings.TrimSuffix(pathname, "/")
	if pathname == "" {
		pathname = "/"
	}

	return c.get("__summary__", pathname)
}

func (c *ScanCache) PutState(stateID objects.Checksum, data []byte) error {
	return c.put("__state__", fmt.Sprintf("%x", stateID), data)
}

func (c *ScanCache) HasState(stateID objects.Checksum) (bool, error) {
	panic("HasState should never be used on the ScanCache backend")
}

func (c *ScanCache) GetState(stateID objects.Checksum) ([]byte, error) {
	panic("GetState should never be used on the ScanCache backend")
}

func (c *ScanCache) GetStates() ([]objects.Checksum, error) {
	panic("GetStates should never be used on the ScanCache backend")
}

func (c *ScanCache) DelState(stateID objects.Checksum) error {
	panic("DelStates should never be used on the ScanCache backend")
}

func (c *ScanCache) GetDelta(blobType packfile.Type, blobCsum objects.Checksum) ([]byte, error) {
	return c.get("__delta__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *ScanCache) HasDelta(blobType packfile.Type, blobCsum objects.Checksum) (bool, error) {
	return c.has("__delta__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *ScanCache) GetDeltaByCsum(blobCsum objects.Checksum) ([]byte, error) {
	for typ := packfile.TYPE_SNAPSHOT; typ <= packfile.TYPE_ERROR; typ++ {
		ret, err := c.GetDelta(typ, blobCsum)

		if err != nil {
			return nil, err
		}

		if ret != nil {
			return ret, nil
		}
	}

	return nil, nil
}

func (c *ScanCache) PutDelta(blobType packfile.Type, blobCsum objects.Checksum, data []byte) error {
	return c.put("__delta__", fmt.Sprintf("%d:%x", blobType, blobCsum), data)
}

func (c *ScanCache) GetDeltasByType(blobType packfile.Type) iter.Seq2[objects.Checksum, []byte] {
	return func(yield func(objects.Checksum, []byte) bool) {
		iter, err := c.db.NewIter(nil)
		if err != nil {
			panic(err)
		}
		defer iter.Close()

		keyPrefix := fmt.Sprintf("__delta__:%d", blobType)
		for iter.SeekGE([]byte(keyPrefix)); iter.Valid(); iter.Next() {
			if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
				break
			}

			/* Extract the csum part of the key, this avoids decoding the full
			 * entry later on if that's the only thing we need */
			key := iter.Key()
			hex_csum := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
			csum, _ := hex.DecodeString(hex_csum)

			if !yield(objects.Checksum(csum), iter.Value()) {
				return
			}
		}
	}
}

func (c *ScanCache) GetDeltas() iter.Seq2[objects.Checksum, []byte] {
	return func(yield func(objects.Checksum, []byte) bool) {
		iter, err := c.db.NewIter(nil)
		if err != nil {
			panic(err)
		}
		defer iter.Close()

		keyPrefix := "__delta__:"
		for iter.SeekGE([]byte(keyPrefix)); iter.Valid(); iter.Next() {
			if !strings.HasPrefix(string(iter.Key()), keyPrefix) {
				break
			}

			/* Extract the csum part of the key, this avoids decoding the full
			 * entry later on if that's the only thing we need */
			key := iter.Key()
			hex_csum := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
			csum, _ := hex.DecodeString(hex_csum)

			if !yield(objects.Checksum(csum), iter.Value()) {
				return
			}
		}
	}
}

func (c *ScanCache) EnumerateKeysWithPrefix(prefix string, reverse bool) iter.Seq2[string, []byte] {
	l := len(prefix)

	return func(yield func(string, []byte) bool) {
		// Use LevelDB's iterator
		iter, err := c.db.NewIter(nil)
		if err != nil {
			panic(err)
		}
		defer iter.Close()

		iter.SeekGE([]byte(prefix))
		if reverse {
			log.Println("seeking to the end of", prefix)
			for iter.Valid() {
				if !strings.HasPrefix(string(iter.Key()), prefix) {
					break
				}
				iter.Next()
			}
			iter.Prev()
			log.Println("done")
		}

		for iter.Valid() {
			key := iter.Key()

			// Check if the key starts with the given prefix
			if !strings.HasPrefix(string(key), prefix) {
				break
			}

			if !yield(string(key)[l:], iter.Value()) {
				return
			}

			if reverse {
				iter.Prev()
			} else {
				iter.Next()
			}
		}
	}
}
