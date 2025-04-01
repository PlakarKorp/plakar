package caching

import (
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/google/uuid"
)

type _RepositoryCache struct {
	*PebbleCache
	manager    *Manager
	cookiesDir string
}

func newRepositoryCache(cacheManager *Manager, repositoryID uuid.UUID) (*_RepositoryCache, error) {
	cookiesDir := filepath.Join(cacheManager.cacheDir, "cookies", repositoryID.String())
	if err := os.MkdirAll(cookiesDir, 0700); err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(cacheManager.cacheDir, "repository", repositoryID.String())
	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &_RepositoryCache{
		manager:     cacheManager,
		cookiesDir:  cookiesDir,
		PebbleCache: db,
	}, nil
}

func (c *_RepositoryCache) HasCookie(name string) bool {
	name = strings.ReplaceAll(name, "/", "_")
	_, err := os.Stat(filepath.Join(c.cookiesDir, name))
	return err == nil
}

func (c *_RepositoryCache) PutCookie(name string) error {
	name = strings.ReplaceAll(name, "/", "_")
	_, err := os.Create(filepath.Join(c.cookiesDir, name))
	return err
}

func (c *_RepositoryCache) PutState(stateID objects.MAC, data []byte) error {
	return c.put("__state__", fmt.Sprintf("%x", stateID), data)
}

func (c *_RepositoryCache) HasState(stateID objects.MAC) (bool, error) {
	return c.has("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetState(stateID objects.MAC) ([]byte, error) {
	return c.get("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) DelState(stateID objects.MAC) error {
	return c.delete("__state__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetStates() (map[objects.MAC][]byte, error) {
	ret := make(map[objects.MAC][]byte, 0)

	for key, val := range c.getObjectsWithMAC("__state__:") {
		value := make([]byte, len(val))
		copy(value, val)

		ret[key] = value
	}

	return ret, nil
}

func (c *_RepositoryCache) GetDelta(blobType resources.Type, blobCsum objects.MAC) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__delta__:%d:%x:", blobType, blobCsum))
}

func (c *_RepositoryCache) PutDelta(blobType resources.Type, blobCsum, packfile objects.MAC, data []byte) error {
	return c.put("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfile), data)
}

func (c *_RepositoryCache) GetDeltasByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__delta__:%d:", blobType))
}

func (c *_RepositoryCache) GetDeltas() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC("__delta__:")
}

func (c *_RepositoryCache) DelDelta(blobType resources.Type, blobCsum, packfileMAC objects.MAC) error {
	return c.delete("__delta__", fmt.Sprintf("%d:%x:%x", blobType, blobCsum, packfileMAC))
}

func (c *_RepositoryCache) PutDeleted(blobType resources.Type, blobCsum objects.MAC, data []byte) error {
	return c.put("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum), data)
}

func (c *_RepositoryCache) HasDeleted(blobType resources.Type, blobCsum objects.MAC) (bool, error) {
	return c.has("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *_RepositoryCache) GetDeleteds() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__deleted__:"))
}

func (c *_RepositoryCache) GetDeletedsByType(blobType resources.Type) iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC(fmt.Sprintf("__deleted__:%d:", blobType))
}

func (c *_RepositoryCache) DelDeleted(blobType resources.Type, blobCsum objects.MAC) error {
	return c.delete("__deleted__", fmt.Sprintf("%d:%x", blobType, blobCsum))
}

func (c *_RepositoryCache) PutPackfile(packfile objects.MAC, data []byte) error {
	return c.put("__packfile__", fmt.Sprintf("%x", packfile), data)
}

func (c *_RepositoryCache) HasPackfile(packfile objects.MAC) (bool, error) {
	return c.has("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *_RepositoryCache) DelPackfile(packfile objects.MAC) error {
	return c.delete("__packfile__", fmt.Sprintf("%x", packfile))
}

func (c *_RepositoryCache) GetPackfiles() iter.Seq2[objects.MAC, []byte] {
	return c.getObjectsWithMAC("__packfile__:")
}

func (c *_RepositoryCache) PutConfiguration(key string, data []byte) error {
	return c.put("__configuration__", key, data)
}

func (c *_RepositoryCache) GetConfiguration(key string) ([]byte, error) {
	return c.get("__configuration__", key)
}

func (c *_RepositoryCache) GetConfigurations() iter.Seq[[]byte] {
	return c.getObjects("__configuration__:")
}

func (c *_RepositoryCache) PutSnapshot(stateID objects.MAC, data []byte) error {
	return c.put("__snapshot__", fmt.Sprintf("%x", stateID), data)
}

func (c *_RepositoryCache) HasSnapshot(stateID objects.MAC) (bool, error) {
	return c.has("__snapshot__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) GetSnapshot(stateID objects.MAC) ([]byte, error) {
	return c.get("__snapshot__", fmt.Sprintf("%x", stateID))
}

func (c *_RepositoryCache) DelSnapshot(stateID objects.MAC) error {
	return c.delete("__snapshot__", fmt.Sprintf("%x", stateID))
}
