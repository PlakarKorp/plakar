//go:build linux || darwin

package plakarfs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInodeCacheEvictsLRU(t *testing.T) {
	c := newInodeCacheWithCapacity(2)

	a := &Dir{cacheKey: "a"}
	b := &Dir{cacheKey: "b"}
	d := &Dir{cacheKey: "d"}

	c.setDir("a", a)
	c.setDir("b", b)

	// Touch a so b becomes the LRU.
	_, ok := c.getDir("a")
	require.True(t, ok)

	c.setDir("d", d)

	_, ok = c.getDir("b")
	require.False(t, ok, "b should have been evicted as LRU")
	_, ok = c.getDir("a")
	require.True(t, ok, "a should still be cached")
	_, ok = c.getDir("d")
	require.True(t, ok, "d should be cached")
}

func TestInodeCacheRemove(t *testing.T) {
	c := newInodeCacheWithCapacity(8)
	d := &Dir{cacheKey: "x"}
	c.setDir("x", d)
	c.removeDir("x")
	_, ok := c.getDir("x")
	require.False(t, ok)
}

func TestInodeCacheCapacityHonored(t *testing.T) {
	c := newInodeCacheWithCapacity(4)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("k%d", i)
		c.setDir(key, &Dir{cacheKey: key})
	}
	c.mu.Lock()
	size := c.evictList.Len()
	c.mu.Unlock()
	require.LessOrEqual(t, size, 4)
}

func TestInodeCacheFilesAndDirsSeparated(t *testing.T) {
	c := newInodeCacheWithCapacity(8)
	c.setDir("same", &Dir{cacheKey: "same"})
	c.setFile("same", &File{cacheKey: "same"})
	_, dok := c.getDir("same")
	_, fok := c.getFile("same")
	require.True(t, dok)
	require.True(t, fok)
}
