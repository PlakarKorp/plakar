package utils

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------- GetPassphraseFromCommand ----------

func TestGetPassphraseFromCommandSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh syntax")
	}
	pass, err := GetPassphraseFromCommand("printf 'sekret'")
	require.NoError(t, err)
	require.Equal(t, "sekret", pass)
}

func TestGetPassphraseFromCommandSingleLineEcho(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh syntax")
	}
	pass, err := GetPassphraseFromCommand("echo hunter2")
	require.NoError(t, err)
	require.Equal(t, "hunter2", pass)
}

func TestGetPassphraseFromCommandMultilineError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh syntax")
	}
	// Two output lines must be rejected.
	_, err := GetPassphraseFromCommand("printf 'a\\nb\\n'")
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many lines")
}

func TestGetPassphraseFromCommandEmptyOutputError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh syntax")
	}
	// No output -> zero lines -> "too many lines" (lines != 1).
	_, err := GetPassphraseFromCommand("true")
	require.Error(t, err)
}

func TestGetPassphraseFromCommandFailingCmd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh syntax")
	}
	// A command that prints one line but exits non-zero -> Wait() errors.
	_, err := GetPassphraseFromCommand("echo oops; exit 3")
	require.Error(t, err)
}
