package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cookies"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// withEntryPointEnv prepares a fully hermetic environment for entryPoint():
//   - config/cache/data dirs are redirected to temp dirs via the XDG_* env vars
//     honored by utils.Get{Config,Cache,Data}Dir.
//   - HOME is redirected so the fallback "fs:$HOME/.plakar" repository never
//     touches the real home directory.
//   - the global flag.CommandLine FlagSet is reset, because entryPoint()
//     registers flags on it and would otherwise panic ("flag redefined") on a
//     second call within the same test binary.
//   - os.Args is saved and restored around the call.
//
// It returns nothing; callers set os.Args themselves after calling this.
func withEntryPointEnv(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	// Force the non-TUI stdio renderer so we never start the bubbletea UI.
	t.Setenv("TERM", "dumb")
	// Make sure no stray passphrase leaks in from the runner's environment.
	t.Setenv("PLAKAR_PASSPHRASE", "")
	os.Unsetenv("PLAKAR_PASSPHRASE")
	t.Setenv("PLAKAR_REPOSITORY", "")
	os.Unsetenv("PLAKAR_REPOSITORY")

	// Reset the global flag set: entryPoint defines flags on flag.CommandLine.
	origArgs := os.Args
	origFlags := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(origArgs[0], flag.ContinueOnError)
	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origFlags
	})
}

func TestEntryPointDisableSecurityCheck(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "-disable-security-check"}

	status := entryPoint()
	require.Equal(t, 0, status)
}

func TestEntryPointEnableSecurityCheck(t *testing.T) {
	withEntryPointEnv(t)

	// First disable (this is what creates the "disabled" cookie). Then enabling
	// removes it and returns 0. Enabling without a prior disable is an error in
	// the cookie store, so the two calls must share the same HOME/cache dir.
	os.Args = []string{"plakar", "-disable-security-check"}
	require.Equal(t, 0, entryPoint())

	// reset the global flag set before the second entryPoint() call
	flag.CommandLine = flag.NewFlagSet("plakar", flag.ContinueOnError)

	os.Args = []string{"plakar", "-enable-security-check"}
	require.Equal(t, 0, entryPoint())
}

func TestEntryPointNoSubcommand(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointInvalidCPU(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "-cpu", "0", "version"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointTooManyCPU(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "-cpu", "100000", "version"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointInvalidConcurrency(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "-concurrency", "0", "version"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointBadFlag(t *testing.T) {
	withEntryPointEnv(t)
	// An unknown flag makes flag.Parse fail. flag.CommandLine was created with
	// ContinueOnError, so Parse returns an error rather than exiting; entryPoint
	// continues with default (empty) flags and then reports "a subcommand must
	// be provided".
	os.Args = []string{"plakar", "-not-a-flag"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointKeyfileMissing(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "-keyfile", filepath.Join(t.TempDir(), "nope.key"), "version"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointCPUProfileBadPath(t *testing.T) {
	withEntryPointEnv(t)
	// A profile path inside a non-existent directory cannot be created.
	bad := filepath.Join(t.TempDir(), "missing", "cpu.prof")
	os.Args = []string{"plakar", "-profile-cpu", bad, "version"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointUnknownCommand(t *testing.T) {
	withEntryPointEnv(t)
	os.Args = []string{"plakar", "this-command-does-not-exist"}

	status := entryPoint()
	require.Equal(t, 1, status)
}

func TestEntryPointVersion(t *testing.T) {
	withEntryPointEnv(t)
	// "version" is a BeforeRepositoryOpen command, so it runs without opening a
	// repository. This drives the full happy-path dispatch end to end.
	os.Args = []string{"plakar", "version"}

	status := entryPoint()
	require.Equal(t, 0, status)
}

func TestEntryPointAtVersion(t *testing.T) {
	withEntryPointEnv(t)
	// Exercise the "at <location> <command>" argument-parsing branch. version is
	// BeforeRepositoryOpen so the location is parsed but never opened.
	os.Args = []string{"plakar", "at", t.TempDir(), "version"}

	status := entryPoint()
	require.Equal(t, 0, status)
}

func TestEntryPointOpenMissingRepository(t *testing.T) {
	withEntryPointEnv(t)
	// "ls" needs to open a repository; the temp location is empty, so opening
	// fails and entryPoint returns the RepoNotFound exit code (non-zero).
	os.Args = []string{"plakar", "at", t.TempDir(), "ls"}

	status := entryPoint()
	require.NotEqual(t, 0, status)
}

// --- getPassphraseFromEnv -------------------------------------------------

func newBareContext(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.SetCookies(cookies.NewManager(t.TempDir()))
	t.Cleanup(func() { ctx.Close() })
	return ctx
}

func TestGetPassphraseFromEnv(t *testing.T) {
	t.Run("from KeyFromFile takes precedence", func(t *testing.T) {
		ctx := newBareContext(t)
		ctx.KeyFromFile = "secretkey"
		got, err := getPassphraseFromEnv(ctx, map[string]string{"passphrase": "other"})
		require.NoError(t, err)
		require.Equal(t, "secretkey", got)
	})

	t.Run("from params passphrase", func(t *testing.T) {
		ctx := newBareContext(t)
		params := map[string]string{"passphrase": "frompar"}
		got, err := getPassphraseFromEnv(ctx, params)
		require.NoError(t, err)
		require.Equal(t, "frompar", got)
		_, ok := params["passphrase"]
		require.False(t, ok, "passphrase key should be consumed")
	})

	t.Run("from PLAKAR_PASSPHRASE env", func(t *testing.T) {
		ctx := newBareContext(t)
		t.Setenv("PLAKAR_PASSPHRASE", "fromenv")
		got, err := getPassphraseFromEnv(ctx, map[string]string{})
		require.NoError(t, err)
		require.Equal(t, "fromenv", got)
	})

	t.Run("none available", func(t *testing.T) {
		ctx := newBareContext(t)
		os.Unsetenv("PLAKAR_PASSPHRASE")
		got, err := getPassphraseFromEnv(ctx, map[string]string{})
		require.NoError(t, err)
		require.Equal(t, "", got)
	})

	t.Run("from passphrase_cmd", func(t *testing.T) {
		ctx := newBareContext(t)
		os.Unsetenv("PLAKAR_PASSPHRASE")
		params := map[string]string{"passphrase_cmd": "printf hunter2"}
		got, err := getPassphraseFromEnv(ctx, params)
		require.NoError(t, err)
		require.Equal(t, "hunter2", got)
		_, ok := params["passphrase_cmd"]
		require.False(t, ok, "passphrase_cmd key should be consumed")
	})
}

// --- setupEncryption ------------------------------------------------------

func TestSetupEncryptionNoEncryption(t *testing.T) {
	ctx := newBareContext(t)
	cfg := &storage.Configuration{Encryption: nil}
	require.NoError(t, setupEncryption(ctx, cfg))
	require.Nil(t, ctx.GetSecret())
}

func TestSetupEncryptionWithKeyfile(t *testing.T) {
	// Build an encrypted repository to obtain a valid encryption configuration,
	// then verify setupEncryption unlocks it from ctx.KeyFromFile.
	pass := []byte("correct horse battery staple")
	repo, ctx := ptesting.GenerateRepository(t, &bytes.Buffer{}, &bytes.Buffer{}, &pass)

	cfg := repo.Configuration()
	require.NotNil(t, cfg.Encryption)

	// fresh context that only has the passphrase, to exercise the keyfile branch
	ctx2 := appcontext.NewAppContext()
	ctx2.SetCookies(cookies.NewManager(t.TempDir()))
	t.Cleanup(func() { ctx2.Close() })
	ctx2.KeyFromFile = string(pass)

	require.NoError(t, setupEncryption(ctx2, &cfg))
	require.NotNil(t, ctx2.GetSecret())

	// wrong passphrase must fail to unlock
	ctx3 := appcontext.NewAppContext()
	ctx3.SetCookies(cookies.NewManager(t.TempDir()))
	t.Cleanup(func() { ctx3.Close() })
	ctx3.KeyFromFile = "wrong passphrase"
	err := setupEncryption(ctx3, &cfg)
	require.ErrorIs(t, err, ErrCantUnlock)

	_ = ctx // silence unused if helper changes
}

// --- listCmds -------------------------------------------------------------

func TestListCmds(t *testing.T) {
	var buf bytes.Buffer
	listCmds(&buf, "  ")
	out := buf.String()
	require.NotEmpty(t, out)
	// a well-known top-level command should appear
	require.Contains(t, out, "version")
	// the "diag" and "cached" commands are intentionally hidden
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			require.NotEqual(t, "diag", fields[0])
			require.NotEqual(t, "cached", fields[0])
		}
	}
	// every emitted line is prefixed
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		require.True(t, strings.HasPrefix(line, "  "), "line %q lacks prefix", line)
	}
}

func TestListCmdsWriterReceivesEverything(t *testing.T) {
	// sanity: listCmds writes to the provided writer only
	var buf bytes.Buffer
	listCmds(io.Writer(&buf), "> ")
	require.Contains(t, buf.String(), "> ")
}
