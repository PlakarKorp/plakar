package caching

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/google/uuid"
)

type PackingCache struct {
	*PebbleCache

	id      string
	manager *Manager
}

func newPackingCache(cacheManager *Manager) (*PackingCache, error) {
	id := uuid.NewString()
	cacheDir := filepath.Join(cacheManager.cacheDir, "packing", id)

	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &PackingCache{
		PebbleCache: db,
		manager:     cacheManager,
	}, nil
}

func (c *PackingCache) Close() error {
	c.PebbleCache.Close()
	return os.RemoveAll(filepath.Join(c.manager.cacheDir, "packing", c.id))
}

func (c *PackingCache) PutBlob(Type resources.Type, mac objects.MAC) error {
	return c.put("__blob__", fmt.Sprintf("%d:%x", Type, mac), nil)
}

func (c *PackingCache) HasBlob(Type resources.Type, mac objects.MAC) (bool, error) {
	return c.has("__blob__", fmt.Sprintf("%d:%x", Type, mac))
}
