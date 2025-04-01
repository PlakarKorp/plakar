package caching

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"iter"
	"path/filepath"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/cockroachdb/pebble/v2"
	"github.com/google/uuid"
)

type MaintenanceCache struct {
	*PebbleCache
	manager *Manager
}

func newMaintenanceCache(cacheManager *Manager, repositoryID uuid.UUID) (*MaintenanceCache, error) {
	cacheDir := filepath.Join(cacheManager.cacheDir, "maintenance", repositoryID.String())

	db, err := New(cacheDir)
	if err != nil {
		return nil, err
	}

	return &MaintenanceCache{
		PebbleCache: db,
		manager:     cacheManager,
	}, nil
}

func (c *MaintenanceCache) PutSnapshot(snapshotID objects.MAC, data []byte) error {
	return c.put("__snapshot__", fmt.Sprintf("%x", snapshotID), data)
}

func (c *MaintenanceCache) HasSnapshot(snapshotID objects.MAC) (bool, error) {
	return c.has("__snapshot__", fmt.Sprintf("%x", snapshotID))
}

func (c *MaintenanceCache) DeleteSnapshot(snapshotID objects.MAC) error {
	return c.delete("__snapshot__", fmt.Sprintf("%x", snapshotID))
}

func (c *MaintenanceCache) PutPackfile(snapshotID, packfileMAC objects.MAC) error {
	return c.put("__packfile__", fmt.Sprintf("%x:%x", packfileMAC, snapshotID), packfileMAC[:])
}

func (c *MaintenanceCache) HasPackfile(packfileMAC objects.MAC) bool {
	for range c.getObjects(fmt.Sprintf("__packfile__:%x:", packfileMAC)) {
		return true
	}

	return false
}

func (c *MaintenanceCache) GetPackfiles(snapshotID objects.MAC) iter.Seq[objects.MAC] {
	return func(yield func(objects.MAC) bool) {
		for p := range c.getObjects("__packfile__:") {
			if !yield(objects.MAC(p)) {
				return
			}
		}
	}
}

func (c *MaintenanceCache) DeleletePackfiles(snapshotID objects.MAC) error {
	iter, _ := c.db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		hex_mac := string(key[bytes.LastIndexByte(key, byte(':'))+1:])
		mac, err := hex.DecodeString(hex_mac)
		if err != nil {
			return err
		}

		if objects.MAC(mac) == snapshotID {
			err := c.db.Delete(iter.Key(), nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
