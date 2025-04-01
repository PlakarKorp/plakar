package caching

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
)

type ScanCache struct {
	*PebbleCache

	snapshotID [32]byte
	manager    *Manager
}

func newScanCache(cacheManager *Manager, snapshotID [32]byte) (*ScanCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "scan", fmt.Sprintf("%x", snapshotID))

	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &ScanCache{
		PebbleCache: db,
		snapshotID:  snapshotID,
		manager:     cacheManager,
	}, nil
}

func (c *ScanCache) Close() error {
	c.PebbleCache.Close()
	return os.RemoveAll(filepath.Join(c.manager.cacheDir, "scan", fmt.Sprintf("%x", c.snapshotID)))
}

func (c *ScanCache) PutFile(file string, data []byte) error {
	return c.put("__file__", file, data)
}

func (c *ScanCache) GetFile(file string) ([]byte, error) {
	return c.get("__file__", file)
}

func (c *ScanCache) PutXattr(xattr string, data []byte) error {
	return c.put("__xattr__", xattr, data)
}

func (c *ScanCache) GetXattr(xattr string) ([]byte, error) {
	return c.get("__xattr__", xattr)
}

func (c *ScanCache) PutDirectory(directory string, data []byte) error {
	return c.put("__directory__", directory, data)
}

func (c *ScanCache) GetDirectory(directory string) ([]byte, error) {
	return c.get("__directory__", directory)
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

func (c *ScanCache) PutState(stateID objects.MAC, data []byte) error {
	return c.put("__state__", fmt.Sprintf("%x", stateID), data)
}

func (c *ScanCache) HasState(stateID objects.MAC) (bool, error) {
	panic("HasState should never be used on the ScanCache backend")
}

func (c *ScanCache) GetState(stateID objects.MAC) ([]byte, error) {
	panic("GetState should never be used on the ScanCache backend")
}

func (c *ScanCache) GetStates() (map[objects.MAC][]byte, error) {
	panic("GetStates should never be used on the ScanCache backend")
}

func (c *ScanCache) DelState(stateID objects.MAC) error {
	panic("DelStates should never be used on the ScanCache backend")
}

func (c *ScanCache) GetDelta(blobType resources.Type, blobCsum objects.MAC) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__delta__:%d:%x:", blobType, blobCsum))
}

func (c *ScanCache) PutDelta(blobType resources.Type, blobCsum, packfile objects.MAC, data []byte) error {
	return c.put("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfile), data)
}

func (c *ScanCache) GetDeltasByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__delta__:%d:", blobType))
}

func (c *ScanCache) GetDeltas() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC("__delta__:")
}

func (c *ScanCache) DelDelta(blobType resources.Type, blobCsum, packfileMAC objects.MAC) error {
	return c.delete("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfileMAC))
}

func (c *ScanCache) PutDeleted(blobType resources.Type, blobCsum objects.MAC, data []byte) error {
	return c.put("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum), data)
}

func (c *ScanCache) HasDeleted(blobType resources.Type, blobCsum objects.MAC) (bool, error) {
	return c.has("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *ScanCache) GetDeleteds() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__deleted__:"))
}

func (c *ScanCache) GetDeletedsByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__deleted__:%d:", blobType))
}

func (c *ScanCache) DelDeleted(blobType resources.Type, blobCsum objects.MAC) error {
	return c.delete("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *ScanCache) PutPackfile(packfile objects.MAC, data []byte) error {
	return c.put("__packfile__", fmt.Sprintf("%x", packfile), data)
}

func (c *ScanCache) HasPackfile(packfile objects.MAC) (bool, error) {
	return c.has("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *ScanCache) DelPackfile(packfile objects.MAC) error {
	return c.delete("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *ScanCache) GetPackfiles() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC("__packfile__:")
}

func (c *ScanCache) PutConfiguration(key string, data []byte) error {
	return c.put("__configuration__", key, data)
}

func (c *ScanCache) GetConfiguration(key string) ([]byte, error) {
	return c.get("__configuration__", key)
}

func (c *ScanCache) GetConfigurations() iter.Seq[[]byte] {
	return c.getObjects("__configuration__:")
}

func (c *ScanCache) EnumerateKeysWithPrefix(prefix string, reverse bool) iter.Seq2[string, []byte] {
	l := len(prefix)

	return func(yield func(string, []byte) bool) {
		iter, _ := c.db.NewIter(MakePrefixIterIterOptions([]byte(prefix)))
		defer iter.Close()

		if reverse {
			iter.Last()
		} else {
			iter.First()
		}

		for iter.Valid() {
			key := iter.Key()
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
