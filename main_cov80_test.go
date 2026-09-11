package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// runEntryPointCov80 mirrors the existing runEntryPoint harness but lets the
// caller pre-seed the hermetic HOME/config directory before entryPoint runs.
// The setup callback receives the base temp dir (which is HOME) and the
// resolved XDG_CONFIG_HOME path so it can drop a stores.yml in place.
func runEntryPointCov80(t *testing.T, setup func(home, configDir string), args ...string) (status int, stdout, stderr string) {
	t.Helper()

	base := t.TempDir()
	configDir := filepath.Join(base, "config")
	for _, kv := range [][2]string{
		{"HOME", base},
		{"XDG_CONFIG_HOME", configDir},
		{"XDG_CACHE_HOME", filepath.Join(base, "cache")},
		{"XDG_DATA_HOME", filepath.Join(base, "data")},
		{"TERM", "dumb"},
		{"PLAKAR_REPOSITORY", ""},
		{"PLAKAR_PASSPHRASE", ""},
	} {
		t.Setenv(kv[0], kv[1])
	}

	if setup != nil {
		setup(base, filepath.Join(configDir, "plakar"))
	}

	oldArgs := os.Args
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	oldFlag := flag.CommandLine
	t.Cleanup(func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		flag.CommandLine = oldFlag
	})

	flag.CommandLine = flag.NewFlagSet("plakar", flag.ContinueOnError)

	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan string)
	errCh := make(chan string)
	pump := func(r *os.File, ch chan string) {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, e := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if e != nil {
				break
			}
		}
		ch <- b.String()
	}
	go pump(rOut, outCh)
	go pump(rErr, errCh)

	os.Args = append([]string{"plakar"}, args...)
	status = entryPoint()

	_ = wOut.Close()
	_ = wErr.Close()
	stdout = <-outCh
	stderr = <-errCh
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	return status, stdout, stderr
}

// writeStoresYmlCov80 writes a v1.0.0 stores.yml declaring a single store with
// the given name + location and marking it as the default repository.
func writeStoresYmlCov80(t *testing.T, configDir, name, location string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(configDir, 0700))
	// The config loader (config.Load) reads sources.yml + destinations.yml
	// + stores.yml; if any is missing it falls back to the legacy single-file
	// format and ignores stores.yml. Write all three so our store is honoured.
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "sources.yml"),
		[]byte("version: v1.0.0\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "destinations.yml"),
		[]byte("version: v1.0.0\ndestinations: {}\n"), 0600))
	content := "version: v1.0.0\n" +
		"default: " + name + "\n" +
		"stores:\n" +
		"  " + name + ":\n" +
		"    location: " + location + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "stores.yml"), []byte(content), 0600))
}

// When no `at` and no PLAKAR_REPOSITORY are given but the config declares a
// default repository, entryPoint resolves the repo via the "@<name>" alias.
// This drives the `def != ""` branch and a successful GetRepository lookup.
func TestEntryPointDefaultRepositoryFromConfigCov80(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	t.Setenv(plaintextEnv, "1") // the repository below is plaintext

	// First create the repository at an explicit location.
	status, _, stderr := runEntryPointCov80(t, nil, "at", repoDir, "create", "-plaintext")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	// Now point the config's default store at it and run `info` with no `at`.
	status, stdout, stderr := runEntryPointCov80(t, func(home, configDir string) {
		writeStoresYmlCov80(t, configDir, "mydefault", "fs:"+repoDir)
	}, "info")
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
	require.NotEmpty(t, stdout+stderr)
}

// A config whose default store name does not resolve to a known repository
// drives the GetRepository error branch.
func TestEntryPointDefaultRepositoryUnknownAliasCov80(t *testing.T) {
	status, _, stderr := runEntryPointCov80(t, func(home, configDir string) {
		// declare default "ghost" but no store entry of that name resolvable
		require.NoError(t, os.MkdirAll(configDir, 0700))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "sources.yml"),
			[]byte("version: v1.0.0\nsources: {}\n"), 0600))
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "destinations.yml"),
			[]byte("version: v1.0.0\ndestinations: {}\n"), 0600))
		content := "version: v1.0.0\n" +
			"default: ghost\n" +
			"stores: {}\n"
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "stores.yml"), []byte(content), 0600))
	}, "info")
	require.NotEqual(t, 0, status)
	require.NotEmpty(t, stderr)
}

// Supplying PLAKAR_PASSPHRASE drives the `passphrase != ""` branch where the
// env passphrase is copied into ctx.KeyFromFile before the command runs. With a
// plaintext repo the secret is simply ignored, so the command still succeeds.
func TestEntryPointEnvPassphraseWithPlaintextRepoCov80(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	t.Setenv(plaintextEnv, "1") // the repository below is plaintext
	status, _, stderr := runEntryPointCov80(t, nil, "at", repoDir, "create", "-plaintext")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	status, _, stderr = runEntryPointCov80(t, func(home, configDir string) {
		// The harness blanks PLAKAR_PASSPHRASE in its env loop before calling
		// this callback, so set it here to drive the `passphrase != ""` branch.
		os.Setenv("PLAKAR_PASSPHRASE", "some-passphrase")
	}, "at", repoDir, "info")
	// The plaintext repo ignores the secret; info should succeed.
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
}

// Running a BeforeRepositoryOpen command (help) skips repository opening
// entirely: it must succeed without any repo on disk and print usage text.
func TestEntryPointHelpBeforeRepositoryOpenCov80(t *testing.T) {
	status, stdout, stderr := runEntryPointCov80(t, nil, "help")
	require.Equal(t, 0, status)
	require.NotEmpty(t, stdout+stderr)
}

// The deprecated -config flag, when it differs from the default, must print a
// deprecation notice and still be honoured as the config dir.
func TestEntryPointDeprecatedConfigNoticeCov80(t *testing.T) {
	altConfig := t.TempDir()
	status, _, stderr := runEntryPointCov80(t, nil, "-config", altConfig, "version")
	require.Equal(t, 0, status)
	require.Contains(t, stderr, "deprecated")
}
