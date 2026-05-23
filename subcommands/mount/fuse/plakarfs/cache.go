//go:build linux || darwin

package plakarfs

import (
	"container/list"
	"strings"
	"sync"
)

// defaultInodeCacheSize bounds the number of Dir + File entries cached.
// At this size the cache footprint is on the order of a few MB.
const defaultInodeCacheSize = 65536

// inodeCache is a bounded LRU shared between Dir and File entries. Entries
// are evicted in LRU order when the cap is reached. The cache is purely a
// performance optimization: Forget removes an entry, and dropping an entry
// while the kernel still references the corresponding node is safe — a later
// Lookup will recreate it.
type inodeCache struct {
	mu       sync.Mutex
	capacity int

	files     map[string]*list.Element
	dirs      map[string]*list.Element
	evictList *list.List
}

type cacheEntry struct {
	key   string
	isDir bool
	file  *File
	dir   *Dir
}

func newInodeCache() *inodeCache {
	return newInodeCacheWithCapacity(defaultInodeCacheSize)
}

func newInodeCacheWithCapacity(capacity int) *inodeCache {
	if capacity < 1 {
		capacity = 1
	}
	return &inodeCache{
		capacity:  capacity,
		files:     make(map[string]*list.Element, capacity),
		dirs:      make(map[string]*list.Element, capacity),
		evictList: list.New(),
	}
}

func stableKey(parts ...string) string {
	return strings.Join(parts, "/")
}

func (c *inodeCache) setFile(key string, file *File) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elt, ok := c.files[key]; ok {
		elt.Value.(*cacheEntry).file = file
		c.evictList.MoveToBack(elt)
		return
	}
	elt := c.evictList.PushBack(&cacheEntry{key: key, isDir: false, file: file})
	c.files[key] = elt
	c.evictLocked()
}

func (c *inodeCache) getFile(key string) (*File, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	elt, ok := c.files[key]
	if !ok {
		return nil, false
	}
	c.evictList.MoveToBack(elt)
	return elt.Value.(*cacheEntry).file, true
}

func (c *inodeCache) removeFile(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elt, ok := c.files[key]; ok {
		c.evictList.Remove(elt)
		delete(c.files, key)
	}
}

func (c *inodeCache) setDir(key string, dir *Dir) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elt, ok := c.dirs[key]; ok {
		elt.Value.(*cacheEntry).dir = dir
		c.evictList.MoveToBack(elt)
		return
	}
	elt := c.evictList.PushBack(&cacheEntry{key: key, isDir: true, dir: dir})
	c.dirs[key] = elt
	c.evictLocked()
}

func (c *inodeCache) getDir(key string) (*Dir, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	elt, ok := c.dirs[key]
	if !ok {
		return nil, false
	}
	c.evictList.MoveToBack(elt)
	return elt.Value.(*cacheEntry).dir, true
}

func (c *inodeCache) removeDir(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if elt, ok := c.dirs[key]; ok {
		c.evictList.Remove(elt)
		delete(c.dirs, key)
	}
}

func (c *inodeCache) evictLocked() {
	for c.evictList.Len() > c.capacity {
		front := c.evictList.Front()
		if front == nil {
			return
		}
		ent := front.Value.(*cacheEntry)
		c.evictList.Remove(front)
		if ent.isDir {
			delete(c.dirs, ent.key)
		} else {
			delete(c.files, ent.key)
		}
	}
}
