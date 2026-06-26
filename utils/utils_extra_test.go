package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	addr, err := ValidateEmail("user@example.com")
	require.NoError(t, err)
	require.Equal(t, "user@example.com", addr)

	_, err = ValidateEmail("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be empty")

	_, err = ValidateEmail("not-an-email")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid email address")

	// Address with display name: ParseAddress succeeds but the parsed
	// address won't equal the raw input, so it is rejected.
	_, err = ValidateEmail("Bob <bob@example.com>")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid email address")
}

func TestGetPassphraseFromCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}

	pass, err := GetPassphraseFromCommand("echo hunter2")
	require.NoError(t, err)
	require.Equal(t, "hunter2", pass)
}

func TestGetPassphraseFromCommandTooManyLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}
	_, err := GetPassphraseFromCommand("printf 'a\\nb\\n'")
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many lines")
}

func TestGetPassphraseFromCommandFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses /bin/sh")
	}
	// Command exits non-zero -> Wait returns an error.
	_, err := GetPassphraseFromCommand("echo x; exit 1")
	require.Error(t, err)
}

func TestGetDataDirWithEnv(t *testing.T) {
	tempDir := t.TempDir()
	dataEnv := "XDG_DATA_HOME"
	if runtime.GOOS == "windows" {
		dataEnv = "LocalAppData"
	}
	t.Setenv(dataEnv, tempDir)

	dataDir, err := GetDataDir("testapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(tempDir, "testapp"), dataDir)
	require.True(t, filepath.IsAbs(dataDir))
}

func TestGetDataDirDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("default path differs on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	dataDir, err := GetDataDir("testapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".local", "share", "testapp"), dataDir)
}

func TestGetCacheDirDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("default path differs on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", "")

	cacheDir, err := GetCacheDir("testapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".cache", "testapp"), cacheDir)
}

func TestGetConfigDirDefault(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("default path differs on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir, err := GetConfigDir("testapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".config", "testapp"), configDir)
}

func TestShouldCheckUpdateDevelVersion(t *testing.T) {
	// VERSION in this build contains "devel", so update checks are skipped
	// and no cookie is created.
	dir := t.TempDir()
	require.False(t, shouldCheckUpdate(dir))
	_, err := os.Stat(filepath.Join(dir, "last-update-check"))
	require.True(t, os.IsNotExist(err))
}

func TestSanitizeTextRoundTripPrintable(t *testing.T) {
	// Sanity check that unicode printable runes are preserved.
	in := "héllo wörld"
	require.Equal(t, in, SanitizeText(in))
	require.True(t, issafe(in))
}
