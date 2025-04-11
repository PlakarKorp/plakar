package caching

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
)

type VFSCache struct {
	*PebbleCache
	manager *Manager
}

func newVFSCache(cacheManager *Manager, repositoryID uuid.UUID, scheme string, origin string) (*VFSCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "vfs", repositoryID.String(), scheme, origin)

	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &VFSCache{
		PebbleCache: db,
		manager:     cacheManager,
	}, nil
}

func (c *VFSCache) PutDirectory(pathname string, data []byte) error {
	return c.put("__directory__", pathname, data)
}

func (c *VFSCache) GetDirectory(pathname string) ([]byte, error) {
	return c.get("__directory__", pathname)
}

func (c *VFSCache) PutFilename(pathname string, data []byte) error {
	return c.put("__filename__", pathname, data)
}

func (c *VFSCache) GetFilename(pathname string) ([]byte, error) {
	return c.get("__filename__", pathname)
}

func (c *VFSCache) PutFileSummary(pathname string, data []byte) error {
	return c.put("__file_summary__", pathname, data)
}

func (c *VFSCache) GetFileSummary(pathname string) ([]byte, error) {
	return c.get("__file_summary__", pathname)
}

func (c *VFSCache) PutObject(mac [32]byte, data []byte) error {
	return c.put("__object__", fmt.Sprintf("%x", mac), data)
}

func (c *VFSCache) GetObject(mac [32]byte) ([]byte, error) {
	return c.get("__object__", fmt.Sprintf("%x", mac))
}
