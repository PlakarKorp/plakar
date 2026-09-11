package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/pkg"
	"github.com/stretchr/testify/require"
)

func slicesContains2(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func importerBackends() []string { return importer.Backends() }
func exporterBackends() []string { return exporter.Backends() }

func manifestWith(typ, proto string) *pkg.Manifest {
	return &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: pkg.ConnectorType(typ), Protocols: []string{proto}},
		},
	}
}

// ---------------------------------------------------------------------------
// entryPoint: global flag branches that drive renderer selection and the
// post-command epilogue without needing a TTY or the network.
// ---------------------------------------------------------------------------

// TestEntryPoint_TimeFlag exercises the -time epilogue branch which prints the
// elapsed command duration after a successful command.
func TestEntryPoint_TimeFlag(t *testing.T) {
	status, stdout, stderr := runEntryPoint(t, "-time", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
	require.Contains(t, stdout+stderr, "time:")
}

// TestEntryPoint_QuietFlag forces the quiet path (stdio renderer, info logging
// disabled). version still succeeds.
func TestEntryPoint_QuietFlag(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-quiet", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
}

// TestEntryPoint_SilentFlag forces the silent renderer branch.
func TestEntryPoint_SilentFlag(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-silent", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
}

// TestEntryPoint_JSONFlag selects the JSON renderer. Because TERM=dumb is set
// by the harness the stdio branch wins for non-tty, so this also drives the
// !isTerminal() condition; version still succeeds.
func TestEntryPoint_JSONFlag(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-json", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
}

// TestEntryPoint_StdioFlag selects the stdio renderer explicitly.
func TestEntryPoint_StdioFlag(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-stdio", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
}

// TestEntryPoint_TraceFlag enables tracing, which also flips the renderer to
// stdio and calls logger.EnableTracing.
func TestEntryPoint_TraceFlag(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-trace", "all", "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
}

// TestEntryPoint_TooManyCPUs drives the "can't use more cores than available"
// guard by asking for an absurd core count.
func TestEntryPoint_TooManyCPUs(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-cpu", "100000", "version")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "can't use more cores than available")
}

// TestEntryPoint_DeprecatedConfigFlag drives the deprecated -config handling:
// when -config differs from the default and -configdir is left at the default,
// entryPoint prints the deprecation notice and adopts -config as the configdir.
func TestEntryPoint_DeprecatedConfigFlag(t *testing.T) {
	cfgdir := filepath.Join(t.TempDir(), "altconfig")
	require.NoError(t, os.MkdirAll(cfgdir, 0700))
	status, _, stderr := runEntryPoint(t, "-config", cfgdir, "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
	require.Contains(t, stderr, "Option -config is deprecated")
}

// TestEntryPoint_MemProfile drives the -profile-mem epilogue branch: after a
// successful command it writes a heap profile to the given path.
func TestEntryPoint_MemProfile(t *testing.T) {
	prof := filepath.Join(t.TempDir(), "mem.prof")
	status, _, stderr := runEntryPoint(t, "-profile-mem", prof, "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
	info, err := os.Stat(prof)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}

// TestEntryPoint_CPUProfile drives the -profile-cpu setup branch, which starts
// and stops the CPU profiler around the command and writes the profile file.
func TestEntryPoint_CPUProfile(t *testing.T) {
	prof := filepath.Join(t.TempDir(), "cpu.prof")
	status, _, stderr := runEntryPoint(t, "-profile-cpu", prof, "version")
	require.Equalf(t, 0, status, "stderr: %s", stderr)
	info, err := os.Stat(prof)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}

// TestEntryPoint_BadProfileCPUPath drives the error branch where the CPU
// profile file cannot be created (parent directory does not exist).
func TestEntryPoint_BadProfileCPUPath(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "no-such-dir", "cpu.prof")
	status, _, stderr := runEntryPoint(t, "-profile-cpu", bad, "version")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "could not create CPU profile")
}

// ---------------------------------------------------------------------------
// getPassphraseFromEnv: passphrase_cmd that fails.
// ---------------------------------------------------------------------------

func TestGetPassphraseFromEnv_PassphraseCmdFails(t *testing.T) {
	ctx := newTestCtx(t)
	params := map[string]string{"passphrase_cmd": "false"}
	_, err := getPassphraseFromEnv(ctx, params)
	require.Error(t, err)
	// the key is still consumed even on failure
	_, ok := params["passphrase_cmd"]
	require.False(t, ok)
}

// ---------------------------------------------------------------------------
// checkUpdate: second-run, security check enabled, no cached update info.
// utils.CheckUpdate fails (or finds nothing) so checkUpdate returns silently.
// ---------------------------------------------------------------------------

func TestCheckUpdate_SecondRunEnabledNoUpdateInfo(t *testing.T) {
	ctx := newTestCtx(t)
	// Mark not-first-run.
	require.NoError(t, ctx.GetCookies().SetFirstRun())
	// CacheDir is an empty temp dir, so CheckUpdate has nothing to read and
	// returns an error -> checkUpdate returns without printing anything.
	checkUpdate(ctx, false)
	out := ctx.Stdout.(interface{ String() string }).String()
	require.Empty(t, strings.TrimSpace(out))
}

// ---------------------------------------------------------------------------
// pkgpreloadhook: importer and exporter conflict branches.
// ---------------------------------------------------------------------------

func TestPkgPreloadHook_RejectsRegisteredImporter(t *testing.T) {
	if !slicesContains2(importerBackends(), "fs") {
		t.Skip("fs importer not registered")
	}
	m := manifestWith("importer", "fs")
	require.Error(t, pkgpreloadhook(m))
}

func TestPkgPreloadHook_RejectsRegisteredExporter(t *testing.T) {
	if !slicesContains2(exporterBackends(), "fs") {
		t.Skip("fs exporter not registered")
	}
	m := manifestWith("exporter", "fs")
	require.Error(t, pkgpreloadhook(m))
}
