package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
