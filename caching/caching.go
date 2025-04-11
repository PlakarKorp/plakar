package caching

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/google/uuid"
)

const CACHE_VERSION = "2.0.0"

var ErrInUse = fmt.Errorf("cache in use")

type Manager struct {
	cacheDir string

	repositoryCache      map[uuid.UUID]*_RepositoryCache
	repositoryCacheMutex sync.Mutex

	vfsCache      map[string]*VFSCache
	vfsCacheMutex sync.Mutex

	maintenanceCache      map[uuid.UUID]*MaintenanceCache
	maintenanceCacheMutex sync.Mutex
}

func NewManager(cacheDir string) *Manager {
	return &Manager{
		cacheDir: filepath.Join(cacheDir, CACHE_VERSION),

		repositoryCache:  make(map[uuid.UUID]*_RepositoryCache),
		vfsCache:         make(map[string]*VFSCache),
		maintenanceCache: make(map[uuid.UUID]*MaintenanceCache),
	}
}

func (m *Manager) Close() error {
	m.vfsCacheMutex.Lock()
	defer m.vfsCacheMutex.Unlock()

	for _, cache := range m.repositoryCache {
		cache.Close()
	}

	for _, cache := range m.vfsCache {
		cache.Close()
	}

	for _, cache := range m.maintenanceCache {
		cache.Close()
	}

	// we may rework the interface later to allow for error handling
	// at this point closing is best effort
	return nil
}

func (m *Manager) VFS(repositoryID uuid.UUID, scheme string, origin string) (*VFSCache, error) {
	m.vfsCacheMutex.Lock()
	defer m.vfsCacheMutex.Unlock()

	key := fmt.Sprintf("%s://%s", scheme, origin)

	if cache, ok := m.vfsCache[key]; ok {
		return cache, nil
	}

	if cache, err := newVFSCache(m, repositoryID, scheme, origin); err != nil {
		return nil, err
	} else {
		m.vfsCache[key] = cache
		return cache, nil
	}
}

func (m *Manager) Repository(repositoryID uuid.UUID) (*_RepositoryCache, error) {
	m.repositoryCacheMutex.Lock()
	defer m.repositoryCacheMutex.Unlock()

	if cache, ok := m.repositoryCache[repositoryID]; ok {
		return cache, nil
	}

	if cache, err := newRepositoryCache(m, repositoryID); err != nil {
		return nil, err
	} else {
		m.repositoryCache[repositoryID] = cache
		return cache, nil
	}
}

func (m *Manager) Maintenance(repositoryID uuid.UUID) (*MaintenanceCache, error) {
	m.maintenanceCacheMutex.Lock()
	defer m.maintenanceCacheMutex.Unlock()

	if cache, ok := m.maintenanceCache[repositoryID]; ok {
		return cache, nil
	}

	if cache, err := newMaintenanceCache(m, repositoryID); err != nil {
		return nil, err
	} else {
		m.maintenanceCache[repositoryID] = cache
		return cache, nil
	}
}

// XXX - beware that caller has responsibility to call Close() on the returned cache
func (m *Manager) Scan(snapshotID objects.MAC) (*ScanCache, error) {
	return newScanCache(m, snapshotID)
}

// XXX - beware that caller has responsibility to call Close() on the returned cache
func (m *Manager) Check() (*CheckCache, error) {
	return newCheckCache(m)
}
