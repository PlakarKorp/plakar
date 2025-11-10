package plugins

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLoadedPluginsReturnsSortedCopy(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, tmpDir)

	pkgA := Package{Name: "alpha", Version: "v1.0.0", Os: "linux", Arch: "amd64"}
	pkgB := Package{Name: "beta", Version: "v1.0.0", Os: "linux", Arch: "amd64"}

	mgr.pluginsMtx.Lock()
	mgr.plugins[pkgB] = &Plugin{}
	mgr.plugins[pkgA] = &Plugin{}
	mgr.pluginsMtx.Unlock()

	loaded := mgr.GetLoadedPlugins()
	require.Equal(t, []Package{pkgA, pkgB}, loaded)

	loaded[0].Name = "mutated"
	loadedAgain := mgr.GetLoadedPlugins()
	require.Equal(t, []Package{pkgA, pkgB}, loadedAgain)
}

func TestManagerConcurrentAccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, tmpDir)
	pkg := Package{Name: "testpkg", Version: "v1.0.0", Os: "linux", Arch: "amd64"}

	const iterations = 1000

	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			mgr.IsPluginLoaded(pkg)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			mgr.GetLoadedPlugins()
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			mgr.pluginsMtx.Lock()
			if i%2 == 0 {
				mgr.plugins[pkg] = &Plugin{}
			} else {
				delete(mgr.plugins, pkg)
			}
			mgr.pluginsMtx.Unlock()
		}
	}()

	close(start)
	wg.Wait()
}
