package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	_ "github.com/PlakarKorp/integrations/fs/importer"
	_ "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/stretchr/testify/require"
)

// ctxOpt seeds an AppContext before a test uses it.
type ctxOpt func(t *testing.T, ctx *appcontext.AppContext)

// withPolicy adds a policy entry to the context.
func withPolicy(name string, kv ...string) ctxOpt {
	return func(t *testing.T, ctx *appcontext.AppContext) {
		t.Helper()
		require.NoError(t, dispatchPolicy(ctx, "policy", "add", append([]string{name}, kv...)))
	}
}

// withStore adds a store entry to the context.
func withStore(name, location string, kv ...string) ctxOpt {
	return func(t *testing.T, ctx *appcontext.AppContext) {
		t.Helper()
		require.NoError(t, dispatchSubcommand(ctx, "store", "add", append([]string{name, location}, kv...)))
	}
}

// newCtx builds an AppContext backed by an empty on-disk config in a temp dir,
// with stdout and stderr captured.
func newCtx(t *testing.T, opts ...ctxOpt) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	tmpDir := t.TempDir()
	cfg, err := config.LoadOldConfigIfExists(filepath.Join(tmpDir, "config.yaml"))
	require.NoError(t, err)

	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	ctx := appcontext.NewAppContext()
	ctx.Config = cfg
	ctx.ConfigDir = tmpDir
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr

	for _, opt := range opts {
		opt(t, ctx)
	}

	return ctx, bufOut, bufErr
}

// fsLoc returns an fs:// location backed by a fresh temp dir.
func fsLoc(t *testing.T) string {
	t.Helper()
	return "fs://" + t.TempDir()
}

// writeConf writes body to a file in a fresh temp dir and returns its path.
func writeConf(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}
