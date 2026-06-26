package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	_ "github.com/PlakarKorp/integration-fs/importer"
	_ "github.com/PlakarKorp/integration-fs/storage"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
)

// newTestCtx builds an appcontext backed by a temporary config dir, mirroring
// the setup used by the existing tests in this package.
func newTestCtx(t *testing.T) (*appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "config.yaml")
	cfg, err := utils.LoadOldConfigIfExists(configPath)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()
	ctx.ConfigDir = tmpDir
	ctx.Config = cfg
	ctx.Stdout = bufOut
	ctx.Stderr = bufErr
	return ctx, bufOut, bufErr
}

// ---------------------------------------------------------------------------
// Parse() coverage for every command type, good and bad args.
// ---------------------------------------------------------------------------

func TestParseStore(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	// no action -> error
	cmd := &ConfigStoreCmd{}
	require.EqualError(t, cmd.Parse(ctx, []string{}), "no action specified")

	// good args -> stored
	cmd = &ConfigStoreCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"show", "foo"}))
	require.Equal(t, []string{"show", "foo"}, cmd.args)
}

func TestParseSource(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	cmd := &ConfigSourceCmd{}
	require.EqualError(t, cmd.Parse(ctx, []string{}), "no action specified")

	cmd = &ConfigSourceCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "n", "fs://x"}))
	require.Equal(t, []string{"add", "n", "fs://x"}, cmd.args)
}

func TestParseDestination(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	cmd := &ConfigDestinationCmd{}
	require.EqualError(t, cmd.Parse(ctx, []string{}), "no action specified")

	cmd = &ConfigDestinationCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"rm", "n"}))
	require.Equal(t, []string{"rm", "n"}, cmd.args)
}

func TestParsePolicy(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	cmd := &ConfigPolicyCmd{}
	require.EqualError(t, cmd.Parse(ctx, []string{}), "no action specified")

	cmd = &ConfigPolicyCmd{}
	require.NoError(t, cmd.Parse(ctx, []string{"add", "daily"}))
	require.Equal(t, []string{"add", "daily"}, cmd.args)
}

// ---------------------------------------------------------------------------
// Execute path: end-to-end through Parse + Execute for store/source/dest.
// ---------------------------------------------------------------------------

func TestExecuteStoreAddAndShow(t *testing.T) {
	ctx, bufOut, _ := newTestCtx(t)
	repo := &repository.Repository{}

	add := &ConfigStoreCmd{}
	require.NoError(t, add.Parse(ctx, []string{"add", "r1", "fs:///tmp/r1"}))
	status, err := add.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	show := &ConfigStoreCmd{}
	require.NoError(t, show.Parse(ctx, []string{"show", "r1"}))
	status, err = show.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "r1")
	require.Contains(t, bufOut.String(), "location")
}

func TestExecuteStoreError(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	repo := &repository.Repository{}

	// rm of a non-existent store -> Execute returns status 1 + error
	rm := &ConfigStoreCmd{}
	require.NoError(t, rm.Parse(ctx, []string{"rm", "nope"}))
	status, err := rm.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// ---------------------------------------------------------------------------
// dispatchSubcommand: exhaustive over store/source/destination + subcommands.
// ---------------------------------------------------------------------------

func TestDispatchUnknownCmd(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	err := dispatchSubcommand(ctx, "bogus", "show", nil)
	require.EqualError(t, err, `unknown cmd "bogus"`)
}

func TestDispatchUnknownSubcommand(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	err := dispatchSubcommand(ctx, "store", "frobnicate", nil)
	require.EqualError(t, err, "usage: plakar store [add|check|import|ping|rm|set|show|unset]")
}

func TestDispatchAdd(t *testing.T) {
	for _, cmd := range []string{"store", "source", "destination"} {
		t.Run(cmd, func(t *testing.T) {
			ctx, _, _ := newTestCtx(t)

			// too few args
			err := dispatchSubcommand(ctx, cmd, "add", []string{"only-name"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "<name> <location>")

			// good add with @-prefixed name and location= prefix + extra kv
			err = dispatchSubcommand(ctx, cmd, "add", []string{"@thing", "location=fs:///tmp/x", "opt=val"})
			require.NoError(t, err)

			// duplicate add -> already exists
			err = dispatchSubcommand(ctx, cmd, "add", []string{"thing", "fs:///tmp/x"})
			require.EqualError(t, err, cmd+` "thing" already exists`)

			// bad kv (no '=')
			err = dispatchSubcommand(ctx, cmd, "add", []string{"other", "fs:///tmp/y", "badkv"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "<key>=<value>")
		})
	}
}

func TestDispatchSetUnset(t *testing.T) {
	for _, cmd := range []string{"store", "source", "destination"} {
		t.Run(cmd, func(t *testing.T) {
			ctx, _, _ := newTestCtx(t)
			require.NoError(t, dispatchSubcommand(ctx, cmd, "add", []string{"n", "fs:///tmp/x"}))

			// set good
			require.NoError(t, dispatchSubcommand(ctx, cmd, "set", []string{"n", "a=1", "b=2"}))

			// set too few args
			err := dispatchSubcommand(ctx, cmd, "set", []string{"n"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "<key>=<value>")

			// set on missing name
			err = dispatchSubcommand(ctx, cmd, "set", []string{"missing", "a=1"})
			require.EqualError(t, err, cmd+` "missing" does not exist`)

			// set bad kv
			err = dispatchSubcommand(ctx, cmd, "set", []string{"n", "nope"})
			require.Error(t, err)

			// unset good
			require.NoError(t, dispatchSubcommand(ctx, cmd, "unset", []string{"n", "a"}))

			// unset location forbidden
			err = dispatchSubcommand(ctx, cmd, "unset", []string{"n", "location"})
			require.EqualError(t, err, "cannot unset location")

			// unset too few args
			err = dispatchSubcommand(ctx, cmd, "unset", []string{"n"})
			require.Error(t, err)
			require.Contains(t, err.Error(), "<key>")

			// unset missing name
			err = dispatchSubcommand(ctx, cmd, "unset", []string{"missing", "a"})
			require.EqualError(t, err, cmd+` "missing" does not exist`)
		})
	}
}

func TestDispatchRm(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"n", "fs:///tmp/x"}))

	// rm too many args
	err := dispatchSubcommand(ctx, "store", "rm", []string{"a", "b"})
	require.Error(t, err)

	// rm missing
	err = dispatchSubcommand(ctx, "store", "rm", []string{"missing"})
	require.EqualError(t, err, `store "missing" does not exist`)

	// rm good
	require.NoError(t, dispatchSubcommand(ctx, "store", "rm", []string{"n"}))
	require.False(t, ctx.Config.HasRepository("n"))
}

func TestDispatchShowFormats(t *testing.T) {
	ctx, bufOut, bufErr := newTestCtx(t)
	require.NoError(t, dispatchSubcommand(ctx, "store", "add",
		[]string{"n", "fs:///tmp/x", "access_key=topsecret", "plain=visible"}))

	// secrets revealed first (show masks cfgMap in place, so test -secrets before masking)
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-secrets", "n"}))
	require.Contains(t, bufOut.String(), "topsecret")
	bufOut.Reset()

	// json
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-json", "n"}))
	require.Contains(t, bufOut.String(), `"n"`)
	bufOut.Reset()

	// ini
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"-ini", "n"}))
	require.Contains(t, bufOut.String(), "[n]")
	bufOut.Reset()

	// default (yaml) hides secrets
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", []string{"n"}))
	out := bufOut.String()
	require.Contains(t, out, "********")
	require.Contains(t, out, "visible")
	bufOut.Reset()

	// no names -> dumps all
	require.NoError(t, dispatchSubcommand(ctx, "store", "show", nil))
	require.Contains(t, bufOut.String(), "n")
	bufOut.Reset()

	// unknown name -> stderr message + aggregate error
	err := dispatchSubcommand(ctx, "store", "show", []string{"ghost"})
	require.EqualError(t, err, "one or more stores do not exist")
	require.Contains(t, bufErr.String(), `store "ghost" does not exist`)
}

// ---------------------------------------------------------------------------
// check / ping success and error branches per command type.
// ---------------------------------------------------------------------------

func TestDispatchCheckStore(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	// missing name
	err := dispatchSubcommand(ctx, "store", "check", []string{"missing"})
	require.EqualError(t, err, `store "missing" does not exist`)

	// wrong arg count
	err = dispatchSubcommand(ctx, "store", "check", []string{"a", "b"})
	require.EqualError(t, err, "usage: plakar store check <name>")

	// good: fs storage never does I/O on New/Close
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"s", "fs:///tmp/store-check"}))
	require.NoError(t, dispatchSubcommand(ctx, "store", "check", []string{"s"}))

	// unknown backend -> error from storage.New
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"bad", "invalid://x"}))
	err = dispatchSubcommand(ctx, "store", "check", []string{"bad"})
	require.Error(t, err)
}

func TestDispatchPingStore(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	// wrong arg count
	err := dispatchSubcommand(ctx, "store", "ping", nil)
	require.EqualError(t, err, "usage: plakar store ping <name>")

	// missing name
	err = dispatchSubcommand(ctx, "store", "ping", []string{"missing"})
	require.EqualError(t, err, `store "missing" does not exist`)

	// good
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"s", "fs:///tmp/store-ping"}))
	require.NoError(t, dispatchSubcommand(ctx, "store", "ping", []string{"s"}))

	// unknown backend
	require.NoError(t, dispatchSubcommand(ctx, "store", "add", []string{"bad", "invalid://x"}))
	err = dispatchSubcommand(ctx, "store", "ping", []string{"bad"})
	require.Error(t, err)
}

func TestDispatchCheckPingSource(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	dir := t.TempDir() // existing dir, required by fs importer

	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"s", "fs://" + dir}))
	require.NoError(t, dispatchSubcommand(ctx, "source", "check", []string{"s"}))
	require.NoError(t, dispatchSubcommand(ctx, "source", "ping", []string{"s"}))

	// unsupported protocol -> NewImporter error
	require.NoError(t, dispatchSubcommand(ctx, "source", "add", []string{"bad", "invalid://nope"}))
	require.Error(t, dispatchSubcommand(ctx, "source", "check", []string{"bad"}))
	require.Error(t, dispatchSubcommand(ctx, "source", "ping", []string{"bad"}))
}

func TestDispatchCheckPingDestination(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	dir := t.TempDir()

	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"d", "fs://" + dir}))
	require.NoError(t, dispatchSubcommand(ctx, "destination", "check", []string{"d"}))
	require.NoError(t, dispatchSubcommand(ctx, "destination", "ping", []string{"d"}))

	// unsupported protocol -> NewExporter error
	require.NoError(t, dispatchSubcommand(ctx, "destination", "add", []string{"bad", "invalid://nope"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "check", []string{"bad"}))
	require.Error(t, dispatchSubcommand(ctx, "destination", "ping", []string{"bad"}))
}

// ---------------------------------------------------------------------------
// import subcommand.
// ---------------------------------------------------------------------------

func TestDispatchImportFromFile(t *testing.T) {
	ctx, _, bufErr := newTestCtx(t)

	// Write an INI-style config file with two sections.
	cfgFile := filepath.Join(t.TempDir(), "in.ini")
	content := "[alpha]\nlocation = fs:///tmp/alpha\n\n[beta]\nlocation = fs:///tmp/beta\n"
	require.NoError(t, os.WriteFile(cfgFile, []byte(content), 0644))

	// import all sections
	require.NoError(t, dispatchSubcommand(ctx, "store", "import", []string{"-config", cfgFile}))
	require.True(t, ctx.Config.HasRepository("alpha"))
	require.True(t, ctx.Config.HasRepository("beta"))

	// re-import without overwrite -> skipped messages on stderr
	require.NoError(t, dispatchSubcommand(ctx, "store", "import", []string{"-config", cfgFile}))
	require.Contains(t, bufErr.String(), "already exists, skipping")

	// import a specific section with rename
	require.NoError(t, dispatchSubcommand(ctx, "store", "import",
		[]string{"-config", cfgFile, "alpha:alpha-copy"}))
	require.True(t, ctx.Config.HasRepository("alpha-copy"))

	// requesting a non-existent section
	bufErr.Reset()
	require.NoError(t, dispatchSubcommand(ctx, "store", "import",
		[]string{"-config", cfgFile, "ghost"}))
	require.Contains(t, bufErr.String(), `does not exist in config`)
}

func TestDispatchImportErrors(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	// non-existent file
	err := dispatchSubcommand(ctx, "store", "import", []string{"-config", "/no/such/file.ini"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open file")

	// malformed/empty config -> GetConf fails -> "failed to load config"
	empty := filepath.Join(t.TempDir(), "empty.ini")
	require.NoError(t, os.WriteFile(empty, []byte(""), 0644))
	err = dispatchSubcommand(ctx, "store", "import", []string{"-config", empty})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to load config")
}

// ---------------------------------------------------------------------------
// dispatchPolicy: add/rm/set/unset/show + error branches.
// ---------------------------------------------------------------------------

func TestDispatchPolicyUnknown(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	err := dispatchPolicy(ctx, "policy", "frobnicate", nil)
	require.EqualError(t, err, "usage: plakar policy [add|rm|set|show|unset]")
}

func TestDispatchPolicyAdd(t *testing.T) {
	ctx, _, _ := newTestCtx(t)

	// no name
	err := dispatchPolicy(ctx, "policy", "add", []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "<name>")

	// good add with valid key=value (name is a string filter field)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"@daily", "name=foo", "days=3"}))

	// duplicate
	err = dispatchPolicy(ctx, "policy", "add", []string{"daily"})
	require.EqualError(t, err, `policy "daily" already exists`)

	// bad kv (no '=')
	err = dispatchPolicy(ctx, "policy", "add", []string{"other", "badkv"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "<key>=<value>")

	// invalid policy key -> failed to set key
	err = dispatchPolicy(ctx, "policy", "add", []string{"another", "boguskey=1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set key")
}

func TestDispatchPolicySetUnset(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"p"}))

	// set good
	require.NoError(t, dispatchPolicy(ctx, "policy", "set", []string{"p", "days=7"}))

	// set too few args
	err := dispatchPolicy(ctx, "policy", "set", []string{"p"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "<key>=<value>")

	// set missing name
	err = dispatchPolicy(ctx, "policy", "set", []string{"missing", "days=1"})
	require.EqualError(t, err, `policy "missing" does not exist`)

	// set bad kv
	err = dispatchPolicy(ctx, "policy", "set", []string{"p", "nope"})
	require.Error(t, err)

	// set invalid key
	err = dispatchPolicy(ctx, "policy", "set", []string{"p", "bogus=1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to set key")

	// unset good
	require.NoError(t, dispatchPolicy(ctx, "policy", "unset", []string{"p", "days"}))

	// unset too few args
	err = dispatchPolicy(ctx, "policy", "unset", []string{"p"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "<key>")

	// unset missing name
	err = dispatchPolicy(ctx, "policy", "unset", []string{"missing", "days"})
	require.EqualError(t, err, `policy "missing" does not exist`)
}

func TestDispatchPolicyRm(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"p"}))

	// too many args
	err := dispatchPolicy(ctx, "policy", "rm", []string{"a", "b"})
	require.Error(t, err)

	// missing
	err = dispatchPolicy(ctx, "policy", "rm", []string{"missing"})
	require.EqualError(t, err, `policy "missing" does not exist`)

	// good
	require.NoError(t, dispatchPolicy(ctx, "policy", "rm", []string{"p"}))
}

func TestDispatchPolicyShow(t *testing.T) {
	ctx, bufOut, _ := newTestCtx(t)
	require.NoError(t, dispatchPolicy(ctx, "policy", "add", []string{"p", "days=2"}))

	// yaml (default), named
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"p"}))
	require.Contains(t, bufOut.String(), "p")
	bufOut.Reset()

	// json
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", []string{"-json", "p"}))
	require.True(t, strings.Contains(bufOut.String(), "{") || strings.Contains(bufOut.String(), "p"))
	bufOut.Reset()

	// no names -> all
	require.NoError(t, dispatchPolicy(ctx, "policy", "show", nil))
}

func TestExecutePolicyEndToEnd(t *testing.T) {
	ctx, _, _ := newTestCtx(t)
	repo := &repository.Repository{}

	add := &ConfigPolicyCmd{}
	require.NoError(t, add.Parse(ctx, []string{"add", "daily", "days=5"}))
	status, err := add.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// error path through Execute: rm missing -> status 1
	rm := &ConfigPolicyCmd{}
	require.NoError(t, rm.Parse(ctx, []string{"rm", "ghost"}))
	status, err = rm.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// ---------------------------------------------------------------------------
// Small helpers.
// ---------------------------------------------------------------------------

func TestNormalizeHelpers(t *testing.T) {
	require.Equal(t, "name", normalizeName("@name"))
	require.Equal(t, "name", normalizeName("name"))
	require.Equal(t, "fs://x", normalizeLocation("location=fs://x"))
	require.Equal(t, "fs://x", normalizeLocation("fs://x"))
}

func TestMarshalINISections(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, MarshalINISections("sect", map[string]string{"k": "v"}, &buf))
	out := buf.String()
	require.Contains(t, out, "[sect]")
	require.Contains(t, out, "k")
	require.Contains(t, out, "v")
}
