package caching

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/google/uuid"
)

type CheckCache struct {
	*PebbleCache

	id      string
	manager *Manager
}

func newCheckCache(cacheManager *Manager) (*CheckCache, error) {
	id := uuid.NewString()
	cacheDir := filepath.Join(cacheManager.cacheDir, "check", id)

	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &CheckCache{
		PebbleCache: db,
		manager:     cacheManager,
	}, nil
}

func (c *CheckCache) Close() error {
	c.PebbleCache.Close()
	return os.RemoveAll(filepath.Join(c.manager.cacheDir, "scan", c.id))
}

func (c *CheckCache) PutPackfileStatus(mac objects.MAC, err []byte) error {
	return c.put("__packfile__", fmt.Sprintf("%x", mac), err)
}

func (c *CheckCache) GetPackfileStatus(mac objects.MAC) ([]byte, error) {
	return c.get("__packfile__", fmt.Sprintf("%x", mac))
}

func (c *CheckCache) PutVFSStatus(mac objects.MAC, err []byte) error {
	return c.put("__vfs__", fmt.Sprintf("%x", mac), err)
}

func (c *CheckCache) GetVFSStatus(mac objects.MAC) ([]byte, error) {
	return c.get("__vfs__", fmt.Sprintf("%x", mac))
}

func (c *CheckCache) PutVFSEntryStatus(mac objects.MAC, err []byte) error {
	return c.put("__vfs_entry__", fmt.Sprintf("%x", mac), err)
}

func (c *CheckCache) GetVFSEntryStatus(mac objects.MAC) ([]byte, error) {
	return c.get("__vfs_entry__", fmt.Sprintf("%x", mac))
}

func (c *CheckCache) PutObjectStatus(mac objects.MAC, err []byte) error {
	return c.put("__object__", fmt.Sprintf("%x", mac), err)
}

func (c *CheckCache) GetObjectStatus(mac objects.MAC) ([]byte, error) {
	return c.get("__object__", fmt.Sprintf("%x", mac))
}

func (c *CheckCache) PutChunkStatus(mac objects.MAC, err []byte) error {
	return c.put("__chunk__", fmt.Sprintf("%x", mac), err)
}

func (c *CheckCache) GetChunkStatus(mac objects.MAC) ([]byte, error) {
	return c.get("__chunk__", fmt.Sprintf("%x", mac))
}
