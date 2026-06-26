package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	"github.com/stretchr/testify/require"
)

func pkgTestContext(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	ctx.Stdout = &bytes.Buffer{}
	ctx.Stderr = &bytes.Buffer{}
	t.Cleanup(func() { ctx.Close() })
	return ctx
}

func TestSetupPkgManagerHermetic(t *testing.T) {
	ctx := pkgTestContext(t)
	dataDir := t.TempDir()
	cacheDir := t.TempDir()

	// Empty plugin directories: NewFlatBackend + LoadAll succeed with nothing
	// to load. No network is touched (the install/api URLs are configured but
	// never reached without an install command).
	err := setupPkgManager(ctx, dataDir, cacheDir)
	require.NoError(t, err)

	mgr := ctx.GetPkgManager()
	require.NotNil(t, mgr)

	// the API-versioned plugin/cache subdirectories should now exist
	require.DirExists(t, filepath.Join(dataDir, "plugins", pkg.PLUGIN_API_VERSION))
}

func TestPkgPreloadHookUnknownTypeSkipped(t *testing.T) {
	// An unknown connector type is skipped (warning only), not an error.
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: pkg.ConnectorType("totallybogus"), Protocols: []string{"whatever"}},
		},
	}
	require.NoError(t, pkgpreloadhook(m))
}

func TestPkgPreloadHookNoConnectors(t *testing.T) {
	require.NoError(t, pkgpreloadhook(&pkg.Manifest{}))
}

func TestPkgPreloadHookConflictingProtocol(t *testing.T) {
	// "fs" is registered as an importer/exporter/storage backend by the blank
	// imports in main.go, so a package claiming it must be rejected.
	for _, typ := range []pkg.ConnectorType{
		pkg.ConnectorTypeImporter,
		pkg.ConnectorTypeExporter,
		pkg.ConnectorTypeStorage,
	} {
		t.Run(string(typ), func(t *testing.T) {
			m := &pkg.Manifest{
				Connectors: []pkg.ManifestConnector{
					{Type: typ, Protocols: []string{"fs"}},
				},
			}
			err := pkgpreloadhook(m)
			require.Error(t, err)
			require.Contains(t, err.Error(), "already provided")
		})
	}
}

func TestPkgPreloadHookFreshProtocolOK(t *testing.T) {
	// A protocol no backend provides passes the preload check.
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: pkg.ConnectorTypeStorage, Protocols: []string{"this-protocol-does-not-exist-xyz"}},
		},
	}
	require.NoError(t, pkgpreloadhook(m))
}

func TestPkgLoadAndUnloadHookErrorPaths(t *testing.T) {
	// A manifest pointing at a non-existent plugin directory makes plugins.Load
	// and plugins.Unload fail; the hooks swallow the error and only print to
	// stderr, so they must not panic.
	m := &pkg.Manifest{Name: "bogus"}
	p := &pkg.Package{Version: "0.0.0"}

	require.NotPanics(t, func() {
		pkgloadhook(m, p, filepath.Join(t.TempDir(), "missing-plugin-dir"))
	})
	require.NotPanics(t, func() {
		pkgunloadhook(m, p)
	})
}
