package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	_ "github.com/PlakarKorp/integrations/fs/importer"
	_ "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/stretchr/testify/require"
)

func cov80Ctx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
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

// ---------- import with -rclone (third-party prefix synthesis) ----------

func TestDispatchImportRcloneCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)

	// An rclone-style INI section; with -rclone the importer synthesizes a
	// "rclone://" location and prefixes each key.
	content := "[remote1]\ntype = s3\nprovider = AWS\n"
	file := filepath.Join(t.TempDir(), "rclone.conf")
	require.NoError(t, os.WriteFile(file, []byte(content), 0600))

	err := dispatchSubcommand(ctx, "store", "import", []string{"-rclone", "-config", file})
	require.NoError(t, err)
	require.True(t, ctx.Config.HasRepository("remote1"))
	require.Equal(t, "rclone://", ctx.Config.Repositories["remote1"]["location"])
	require.Equal(t, "s3", ctx.Config.Repositories["remote1"]["rclone_type"])
}

// import of a config that contains no usable sections must error with "no
// valid ...s found".
func TestDispatchImportEmptyAfterParseCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)

	// A YAML doc whose only top-level entry is a scalar (skipped by LoadYAML)
	// plus a location so GetConf doesn't reject it, but yields no real
	// sections once empties are stripped. We use an rclone import of an empty
	// INI so the resulting map is empty.
	file := filepath.Join(t.TempDir(), "empty.conf")
	require.NoError(t, os.WriteFile(file, []byte("\n"), 0600))

	err := dispatchSubcommand(ctx, "store", "import", []string{"-rclone", "-config", file})
	// Empty file -> GetConf returns an empty map (rclone synthesizes nothing),
	// so dispatch reports "no valid stores found" OR a load error; either way
	// it must be an error.
	require.Error(t, err)
}

// ---------- ping / check error branches ----------

func TestDispatchPingStoreOpenErrorCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	// Register a store whose location uses an unknown scheme so storage.New
	// fails inside the ping handler (config.go store-ping error branch).
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "no-such-scheme://nowhere"}))
	err := dispatchSubcommand(ctx, "store", "ping", []string{"r"})
	require.Error(t, err)
}

func TestDispatchCheckStoreOpenErrorCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "no-such-scheme://nowhere"}))
	err := dispatchSubcommand(ctx, "store", "check", []string{"r"})
	require.Error(t, err)
}

func TestDispatchCheckSourceBadProtoCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "source", "add",
		[]string{"s", "no-such-scheme://nowhere"}))
	err := dispatchSubcommand(ctx, "source", "check", []string{"s"})
	require.Error(t, err)
}

func TestDispatchCheckDestinationBadProtoCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add",
		[]string{"d", "no-such-scheme://nowhere"}))
	err := dispatchSubcommand(ctx, "destination", "check", []string{"d"})
	require.Error(t, err)
}

// ---------- policy add/set with an invalid value (config.Set error) ----------

func TestDispatchPolicyAddSetErrorCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	// "minutes" expects a non-negative int; a garbage value makes config.Set
	// fail inside the policy add handler.
	err := dispatchPolicy(ctx, "policy", "add", []string{"p", "minutes=notanint"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set key")
}

func TestDispatchPolicySetErrorCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"p"}))
	err := dispatchPolicy(ctx, "policy", "set", []string{"p", "minutes=notanint"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set key")
}

// ---------- policy show in both formats over a populated policy ----------

func TestDispatchPolicyShowYAMLAndJSONCov80(t *testing.T) {
	ctx, bufOut, _ := cov80Ctx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"keep", "days=7"}))

	// YAML (default)
	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"keep"}))
	require.Contains(t, bufOut.String(), "keep")

	// JSON
	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "keep"}))
	require.True(t, strings.HasPrefix(strings.TrimSpace(bufOut.String()), "{"))
}

// ---------- policy default subcommand (unknown action) ----------

func TestDispatchPolicyUnknownActionCov80(t *testing.T) {
	ctx, _, _ := cov80Ctx(t)
	err := dispatchPolicy(ctx, "policy", "bogus", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage:")
}

// ---------- store show -ini format over a populated store ----------

func TestDispatchShowINIFormatCov80(t *testing.T) {
	ctx, bufOut, _ := cov80Ctx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "fs://" + t.TempDir(), "extra=val"}))
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-ini", "r"}))
	out := bufOut.String()
	require.Contains(t, out, "[r]")
	require.Contains(t, out, "extra")
}
