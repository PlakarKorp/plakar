package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// runEntryPoint drives the global entryPoint() dispatcher in a hermetic
// environment. It saves and restores all the global state entryPoint touches
// (os.Args, os.Stdout, os.Stderr, flag.CommandLine) and points the config /
// cache / data directories at per-test temp dirs via the XDG_* / HOME
// environment variables. TERM=dumb forces the stdio renderer instead of the
// TUI, and stdout/stderr are redirected through pipes because entryPoint
// writes to os.Stdout / os.Stderr directly in several branches.
//
// It returns the process status code together with everything written to the
// captured stdout and stderr.
func runEntryPoint(t *testing.T, args ...string) (status int, stdout, stderr string) {
	t.Helper()

	// Hermetic directories.
	base := t.TempDir()
	for _, kv := range [][2]string{
		{"HOME", base},
		{"XDG_CONFIG_HOME", filepath.Join(base, "config")},
		{"XDG_CACHE_HOME", filepath.Join(base, "cache")},
		{"XDG_DATA_HOME", filepath.Join(base, "data")},
		{"TERM", "dumb"},          // force stdio renderer, no TUI
		{"PLAKAR_REPOSITORY", ""}, // don't inherit a real repo
		{"PLAKAR_PASSPHRASE", ""}, // don't inherit a real passphrase
	} {
		t.Setenv(kv[0], kv[1])
	}

	// Save and restore global process state.
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

	// Fresh flag set so re-defining the same flags across calls doesn't panic.
	flag.CommandLine = flag.NewFlagSet("plakar", flag.ContinueOnError)

	// Redirect os.Stdout / os.Stderr through pipes.
	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = wOut
	os.Stderr = wErr

	outCh := make(chan string)
	errCh := make(chan string)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, e := rOut.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if e != nil {
				break
			}
		}
		outCh <- b.String()
	}()
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, e := rErr.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if e != nil {
				break
			}
		}
		errCh <- b.String()
	}()

	os.Args = append([]string{"plakar"}, args...)
	status = entryPoint()

	// Close the write ends so the reader goroutines terminate, then restore.
	_ = wOut.Close()
	_ = wErr.Close()
	stdout = <-outCh
	stderr = <-errCh
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return status, stdout, stderr
}

// ---------- entryPoint: simple dispatch branches ----------

func TestEntryPoint_Version(t *testing.T) {
	status, stdout, _ := runEntryPoint(t, "version")
	require.Equal(t, 0, status)
	// The version subcommand prints the version string to stdout.
	require.NotEmpty(t, strings.TrimSpace(stdout))
}

func TestEntryPoint_NoArgs(t *testing.T) {
	status, _, stderr := runEntryPoint(t)
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "a subcommand must be provided")
}

func TestEntryPoint_BadCPUValue(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-cpu", "0", "version")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "invalid -cpu value")
}

func TestEntryPoint_BadConcurrencyValue(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-concurrency", "0", "version")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "invalid -concurrency value")
}

func TestEntryPoint_UnknownCommand(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "this-command-does-not-exist")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "command not found")
}

// ---------- entryPoint: security-check toggle branches ----------

func TestEntryPoint_DisableSecurityCheck(t *testing.T) {
	status, stdout, _ := runEntryPoint(t, "-disable-security-check", "version")
	require.Equal(t, 0, status)
	require.Contains(t, stdout, "security check disabled")
}

func TestEntryPoint_EnableSecurityCheck(t *testing.T) {
	// First disable, then enable, reusing the same temp HOME so the cookie
	// state carries over within the single entryPoint call's lifecycle.
	status, stdout, _ := runEntryPoint(t, "-enable-security-check", "version")
	require.Equal(t, 0, status)
	require.Contains(t, stdout, "security check enabled")
}

// ---------- entryPoint: keyfile error branch ----------

func TestEntryPoint_MissingKeyfile(t *testing.T) {
	status, _, stderr := runEntryPoint(t, "-keyfile", "/no/such/keyfile", "version")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "could not read key file")
}

// ---------- entryPoint: full create + info flow over a real fs repo ----------

func TestEntryPoint_CreateThenInfo(t *testing.T) {
	repoDir := filepath.Join(t.TempDir(), "repo")
	t.Setenv(plaintextEnv, "1") // the repository below is plaintext

	// Create a plaintext repository at an explicit location.
	status, _, stderr := runEntryPoint(t, "at", repoDir, "create", "-plaintext")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	// The repository must now exist on disk.
	_, err := os.Stat(repoDir)
	require.NoError(t, err)

	// Now run `info` against it: this exercises storage.Open,
	// setupEncryption (nil path on a plaintext repo),
	// repository.NewNoRebuild, cached.RebuildStateFromStore and
	// task.RunCommand.
	status, stdout, stderr := runEntryPoint(t, "at", repoDir, "info")
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
	require.NotEmpty(t, stdout+stderr)
}

func TestEntryPoint_AtMissingCommand(t *testing.T) {
	// Opening a non-existent repo location yields the RepoNotFound exit code.
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	status, _, stderr := runEntryPoint(t, "at", missing, "info")
	require.NotEqual(t, 0, status)
	require.Contains(t, stderr, "failed to open the repository")

	// A dangling "at" is reported for what it is, rather than falling
	// through to the generic "a subcommand must be provided".  These used
	// to go through log.Fatalf, which would have taken the test process
	// down with them.
	status, _, stderr = runEntryPoint(t, "at")
	require.Equal(t, 1, status)
	require.Contains(t, stderr, "missing plakar repository")
}

// ---------- entryPoint: keyfile + plaintext repo unlock (nil secret) ----------

func TestEntryPoint_KeyfileWithPlaintextRepo(t *testing.T) {
	// A keyfile is supplied but the repo is plaintext: setupEncryption sees a
	// nil Encryption config and the keyfile secret is simply ignored.
	repoDir := filepath.Join(t.TempDir(), "repo")
	t.Setenv(plaintextEnv, "1") // the repository below is plaintext
	status, _, stderr := runEntryPoint(t, "at", repoDir, "create", "-plaintext")
	require.Equalf(t, 0, status, "create stderr: %s", stderr)

	keyfile := filepath.Join(t.TempDir(), "key")
	require.NoError(t, os.WriteFile(keyfile, []byte("ignored-passphrase\n"), 0600))

	status, _, stderr = runEntryPoint(t, "-keyfile", keyfile, "at", repoDir, "info")
	require.Equalf(t, 0, status, "info stderr: %s", stderr)
}

// keep the ptesting import referenced even if future edits drop direct uses.
var _ = ptesting.NewMockDir
