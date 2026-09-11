package config

import (
	"bytes"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	_ "github.com/PlakarKorp/integrations/fs/importer"
	_ "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/stretchr/testify/require"
)

// covCtx builds an AppContext backed by an empty on-disk config in a temp dir.
func covCtx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
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
	return ctx, bufOut, bufErr
}

// fsLoc returns an absolute fs:/// location pointing at a fresh temp dir.
func fsLoc(t *testing.T) string {
	t.Helper()
	return "fs://" + t.TempDir()
}

// ---------- check success (store / source / destination) ----------

func TestCovCheckStoreSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", loc}))
	// fs storage backend opens cleanly even when uninitialized.
	require.NoError(t, dispatchSubcommand(ctx, "store", "check", []string{"r"}))
}

func TestCovCheckSourceSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", loc}))
	require.NoError(t, dispatchSubcommand(ctx, "source", "check", []string{"s"}))
}

func TestCovCheckDestinationSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", loc}))
	require.NoError(t, dispatchSubcommand(ctx, "destination", "check", []string{"d"}))
}

// ---------- ping success (store / source / destination) ----------

func TestCovPingStoreSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", loc}))
	require.NoError(t, dispatchSubcommand(ctx, "store", "ping", []string{"r"}))
}

func TestCovPingSourceSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", loc}))
	require.NoError(t, dispatchSubcommand(ctx, "source", "ping", []string{"s"}))
}

func TestCovPingDestinationSuccess(t *testing.T) {
	ctx, _, _ := covCtx(t)
	loc := fsLoc(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", loc}))
	require.NoError(t, dispatchSubcommand(ctx, "destination", "ping", []string{"d"}))
}

// ---------- source add/set/unset/rm/show lifecycle ----------

func TestCovSourceLifecycle(t *testing.T) {
	ctx, bufOut, _ := covCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", "fs:/tmp/s", "k=v"}))
	require.True(t, ctx.Config.HasSource("s"))
	require.Equal(t, "v", ctx.Config.Sources["s"]["k"])

	require.NoError(t, dispatchSubcommand(ctx, "source", "set", []string{"s", "k2=v2"}))
	require.Equal(t, "v2", ctx.Config.Sources["s"]["k2"])

	require.NoError(t, dispatchSubcommand(ctx, "source", "unset", []string{"s", "k2"}))
	_, ok := ctx.Config.Sources["s"]["k2"]
	require.False(t, ok)

	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "source", "show", []string{"s"}))
	require.Contains(t, bufOut.String(), "s")

	require.NoError(t, dispatchSubcommand(ctx, "source", "rm", []string{"s"}))
	require.False(t, ctx.Config.HasSource("s"))
}

// ---------- destination add/set/unset/rm/show lifecycle ----------

func TestCovDestinationLifecycle(t *testing.T) {
	ctx, bufOut, _ := covCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", "fs:/tmp/d", "k=v"}))
	require.True(t, ctx.Config.HasDestination("d"))

	require.NoError(t, dispatchSubcommand(ctx, "destination", "set", []string{"d", "k2=v2"}))
	require.Equal(t, "v2", ctx.Config.Destinations["d"]["k2"])

	require.NoError(t, dispatchSubcommand(ctx, "destination", "unset", []string{"d", "k2"}))
	_, ok := ctx.Config.Destinations["d"]["k2"]
	require.False(t, ok)

	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "destination", "show", []string{"-yaml", "d"}))
	require.Contains(t, bufOut.String(), "d")

	require.NoError(t, dispatchSubcommand(ctx, "destination", "rm", []string{"d"}))
	require.False(t, ctx.Config.HasDestination("d"))
}

// ---------- check/ping failure for unconfigured source & destination ----------

func TestCovCheckSourceUnknownGetFails(t *testing.T) {
	ctx, _, _ := covCtx(t)
	// name exists in the map but with a bogus protocol -> NewImporter fails.
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", "bogus://x"}))
	require.Error(t, dispatchSubcommand(ctx, "source", "check", []string{"s"}))
}

func TestCovCheckDestinationUnknownGetFails(t *testing.T) {
	ctx, _, _ := covCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", "bogus://x"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "check", []string{"d"}))
}

func TestCovPingSourceBadProto(t *testing.T) {
	ctx, _, _ := covCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", "bogus://x"}))
	require.Error(t, dispatchSubcommand(ctx, "source", "ping", []string{"s"}))
}

func TestCovPingDestinationBadProto(t *testing.T) {
	ctx, _, _ := covCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", "bogus://x"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "ping", []string{"d"}))
}

// ---------- source/destination check/ping on unknown name ----------

func TestCovCheckPingMissingName(t *testing.T) {
	ctx, _, _ := covCtx(t)
	require.Error(t, dispatchSubcommand(ctx, "source", "check", []string{"ghost"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "check", []string{"ghost"}))
	require.Error(t, dispatchSubcommand(ctx, "source", "ping", []string{"ghost"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "ping", []string{"ghost"}))
}

// ---------- policy lifecycle via dispatchPolicy (set/unset/show formats) ----------

func TestCovPolicyShowFormatsAndUnset(t *testing.T) {
	ctx, bufOut, _ := covCtx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"daily", "days=7"}))

	// set a value, then show default (yaml), then json.
	require.NoError(t, dispatchPolicy(ctx, "policy", "set", []string{"daily", "tags=auto,nightly"}))

	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"daily"}))
	require.Contains(t, bufOut.String(), "daily")

	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "daily"}))
	require.Contains(t, bufOut.String(), "{")

	// unset a real key.
	require.NoError(t, dispatchPolicy(ctx, "policy", "unset", []string{"daily", "tags"}))

	// rm it.
	require.NoError(t, dispatchPolicy(ctx, "policy", "rm", []string{"daily"}))
}
