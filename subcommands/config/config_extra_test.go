package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

func TestConfigRegisteredFactories(t *testing.T) {
	// Look each command up through the registry to invoke the factory closures
	// registered in init().
	cases := []struct {
		name string
		typ  interface{}
	}{
		{"store", &ConfigStoreCmd{}},
		{"source", &ConfigSourceCmd{}},
		{"destination", &ConfigDestinationCmd{}},
		{"policy", &ConfigPolicyCmd{}},
	}
	for _, c := range cases {
		cmd, _, _ := subcommands.Lookup([]string{c.name})
		require.NotNil(t, cmd, "command %q not registered", c.name)
		require.IsType(t, c.typ, cmd)
	}
}

// newConfigCtx returns a context with an empty on-disk config rooted in a temp
// dir, plus buffered stdout/stderr.
func newConfigCtx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
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

// ---------- helpers ----------

func TestNormalizeHelpers(t *testing.T) {
	require.Equal(t, "name", normalizeName("@name"))
	require.Equal(t, "name", normalizeName("name"))
	require.Equal(t, "fs:/x", normalizeLocation("location=fs:/x"))
	require.Equal(t, "fs:/x", normalizeLocation("fs:/x"))
}

func TestMarshalINISections(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, MarshalINISections("mysection", map[string]string{"key": "val"}, &buf))
	out := buf.String()
	require.Contains(t, out, "[mysection]")
	require.Contains(t, out, "key")
	require.Contains(t, out, "val")
}

func TestDispatchUnknownCmd(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	err := dispatchSubcommand(ctx, "bogus", "show", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown cmd")
}

// ---------- store / source / destination entity wrappers ----------

func TestEntityParseNoAction(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)

	require.Error(t, (&ConfigStoreCmd{}).Parse(ctx, []string{}))
	require.Error(t, (&ConfigSourceCmd{}).Parse(ctx, []string{}))
	require.Error(t, (&ConfigDestinationCmd{}).Parse(ctx, []string{}))
	require.Error(t, (&ConfigPolicyCmd{}).Parse(ctx, []string{}))
}

func TestDestinationParseExecute(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	repo := &repository.Repository{}

	cmd := &ConfigDestinationCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "mydest", "fs:/tmp/dst"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.True(t, ctx.Config.HasDestination("mydest"))

	// A failing dispatch (rm of unknown) propagates status 1.
	cmd = &ConfigDestinationCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "ghost"}))
	status, err = cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestSourceParseExecute(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	repo := &repository.Repository{}

	cmd := &ConfigSourceCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "mysrc", "fs:/tmp/src"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.True(t, ctx.Config.HasSource("mysrc"))
}

// ---------- dispatchSubcommand actions ----------

func TestDispatchAddDuplicateAndMalformed(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)

	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r"}))

	// duplicate name
	err := dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r2"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")

	// too few args
	err = dispatchSubcommand(ctx, "store", "add", []string{"only-name"})
	require.Error(t, err)

	// malformed key=value
	err = dispatchSubcommand(ctx, "store", "add", []string{"r2", "fs:/tmp/r2", "noequalsign"})
	require.Error(t, err)
}

func TestDispatchAddWithOptions(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r", "key=val", "k2=v2"}))
	require.Equal(t, "val", ctx.Config.Repositories["r"]["key"])
	require.Equal(t, "v2", ctx.Config.Repositories["r"]["k2"])
}

func TestDispatchSetUnset(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r"}))

	// set on unknown name
	require.Error(t, dispatchSubcommand(ctx, "store", "set", []string{"ghost", "k=v"}))
	// set malformed
	require.Error(t, dispatchSubcommand(ctx, "store", "set", []string{"r", "bad"}))
	// set ok
	require.NoError(t, dispatchSubcommand(ctx, "store", "set", []string{"r", "k=v"}))
	require.Equal(t, "v", ctx.Config.Repositories["r"]["k"])

	// unset too few args
	require.Error(t, dispatchSubcommand(ctx, "store", "unset", []string{"r"}))
	// unset unknown name
	require.Error(t, dispatchSubcommand(ctx, "store", "unset", []string{"ghost", "k"}))
	// unset location is forbidden
	require.Error(t, dispatchSubcommand(ctx, "store", "unset", []string{"r", "location"}))
	// unset ok
	require.NoError(t, dispatchSubcommand(ctx, "store", "unset", []string{"r", "k"}))
	_, ok := ctx.Config.Repositories["r"]["k"]
	require.False(t, ok)
}

func TestDispatchRm(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r"}))

	// rm unknown
	require.Error(t, dispatchSubcommand(ctx, "store", "rm", []string{"ghost"}))
	// rm too few args
	require.Error(t, dispatchSubcommand(ctx, "store", "rm", []string{}))
	// rm ok
	require.NoError(t, dispatchSubcommand(ctx, "store", "rm", []string{"r"}))
	require.False(t, ctx.Config.HasRepository("r"))
}

func TestDispatchShowSecretsRevealed(t *testing.T) {
	// -secrets must be checked first: non-secret show mutates the in-memory
	// config map (overwriting the value with "********"), so it has to run on a
	// fresh config.
	ctx, bufOut, _ := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "fs:/tmp/r", "passphrase=topsecret"}))

	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-secrets", "r"}))
	require.Contains(t, bufOut.String(), "topsecret")
}

func TestDispatchShowFormats(t *testing.T) {
	ctx, bufOut, _ := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"r", "fs:/tmp/r", "passphrase=topsecret", "plain=visible"}))

	// default (yaml), secrets masked
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"r"}))
	require.Contains(t, bufOut.String(), "********")
	require.NotContains(t, bufOut.String(), "topsecret")
	require.Contains(t, bufOut.String(), "visible")

	// json
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-json", "r"}))
	require.Contains(t, bufOut.String(), "{")

	// ini
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-ini", "r"}))
	require.Contains(t, bufOut.String(), "[r]")
}

func TestDispatchShowAllAndMissing(t *testing.T) {
	ctx, bufOut, bufErr := newConfigCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"r", "fs:/tmp/r"}))

	// no name -> show all
	bufOut.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", nil))
	require.Contains(t, bufOut.String(), "r")

	// a missing name reports an error and writes to stderr
	bufErr.Reset()
	err := dispatchSubcommand(ctx, "store", "show", []string{"ghost"})
	require.Error(t, err)
	require.Contains(t, bufErr.String(), "does not exist")
}

func TestDispatchImportFromFile(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)

	// Write a YAML config with two store sections.
	tmp := filepath.Join(t.TempDir(), "stores.yaml")
	content := "alpha:\n  location: fs:/tmp/alpha\nbeta:\n  location: fs:/tmp/beta\n"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0644))

	require.NoError(t, dispatchSubcommand(ctx, "store", "import", []string{"-config", tmp}))
	require.True(t, ctx.Config.HasRepository("alpha"))
	require.True(t, ctx.Config.HasRepository("beta"))
}

func TestDispatchImportSelectedSection(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	tmp := filepath.Join(t.TempDir(), "stores.yaml")
	content := "alpha:\n  location: fs:/tmp/alpha\nbeta:\n  location: fs:/tmp/beta\n"
	require.NoError(t, os.WriteFile(tmp, []byte(content), 0644))

	// Import only alpha, renamed to gamma.
	require.NoError(t, dispatchSubcommand(ctx, "store", "import", []string{"-config", tmp, "alpha:gamma"}))
	require.True(t, ctx.Config.HasRepository("gamma"))
	require.False(t, ctx.Config.HasRepository("beta"))
}

func TestDispatchImportMissingFile(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	err := dispatchSubcommand(ctx, "store", "import", []string{"-config", "/nonexistent/x.yaml"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open file")
}

func TestDispatchCheckUnknown(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	// check unknown name
	require.Error(t, dispatchSubcommand(ctx, "store", "check", []string{"ghost"}))
	// check wrong arg count
	require.Error(t, dispatchSubcommand(ctx, "store", "check", []string{}))
}

func TestDispatchPingUnknown(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	require.Error(t, dispatchSubcommand(ctx, "store", "ping", []string{"ghost"}))
	require.Error(t, dispatchSubcommand(ctx, "store", "ping", []string{}))
}

// ---------- dispatchPolicy ----------

func TestPolicyParseExecute(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	repo := &repository.Repository{}

	cmd := &ConfigPolicyCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "nightly"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// policies.yml was written.
	_, err = os.Stat(filepath.Join(ctx.ConfigDir, "policies.yml"))
	require.NoError(t, err)
}

func TestDispatchPolicyLifecycle(t *testing.T) {
	ctx, bufOut, _ := newConfigCtx(t)

	// unknown subcommand
	require.Error(t, dispatchPolicy(ctx, "policy", "bogus", nil))

	// add (with no args -> error), then a real add
	require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{}))
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"daily", "tags=auto"}))

	// add duplicate
	require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{"daily"}))
	// add malformed kv
	require.Error(t, dispatchPolicy(ctx, "policy", "add", []string{"weekly", "bad"}))

	// set
	require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"ghost", "k=v"}))
	require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"daily"}))        // too few
	require.Error(t, dispatchPolicy(ctx, "policy", "set", []string{"daily", "bad"})) // malformed

	// show (yaml + json), all + specific
	bufOut.Reset()
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", nil))
	require.Contains(t, bufOut.String(), "daily")
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "daily"}))

	// unset
	require.Error(t, dispatchPolicy(ctx, "policy", "unset", []string{"daily"})) // too few
	require.Error(t, dispatchPolicy(ctx, "policy", "unset", []string{"ghost", "tags"}))

	// rm
	require.Error(t, dispatchPolicy(ctx, "policy", "rm", []string{"ghost"}))
	require.Error(t, dispatchPolicy(ctx, "policy", "rm", []string{}))
	require.NoError(t, dispatchPolicy(ctx, "policy", "rm", []string{"daily"}))
}

func TestDispatchPolicyLoadError(t *testing.T) {
	ctx, _, _ := newConfigCtx(t)
	// A malformed policies.yml makes the loader fail.
	require.NoError(t, os.WriteFile(filepath.Join(ctx.ConfigDir, "policies.yml"),
		[]byte("this: : : not valid\n  - broken"), 0644))
	err := dispatchPolicy(ctx, "policy", "show", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config file")
}
