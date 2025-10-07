package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSnapshotID(t *testing.T) {
	// Test case: Snapshot ID with prefix and pattern
	id := "snapshot123:/path/to/file"
	prefix, pattern := ParseSnapshotID(id)
	require.Equal(t, "snapshot123", prefix)
	if runtime.GOOS == "windows" {
		require.Equal(t, "path/to/file", pattern) // No leading slash on Windows
	} else {
		require.Equal(t, "/path/to/file", pattern)
	}

	// Test case: Snapshot ID without pattern
	id = "snapshot123"
	prefix, pattern = ParseSnapshotID(id)
	require.Equal(t, "snapshot123", prefix)
	require.Equal(t, "", pattern)

	// Test case: Empty input
	id = ""
	prefix, pattern = ParseSnapshotID(id)
	require.Equal(t, "", prefix)
	require.Equal(t, "", pattern)

	// Test case: Pattern without leading slash on non-Windows systems
	if runtime.GOOS != "windows" {
		id = "snapshot123:path/to/file"
		prefix, pattern = ParseSnapshotID(id)
		require.Equal(t, "snapshot123", prefix)
		require.Equal(t, "/path/to/file", pattern)
	}
}

func TestGetVersion(t *testing.T) {
	// Use the VERSION constant from the package
	expectedVersion := VERSION

	// Call GetVersion and compare the result
	version := GetVersion()
	require.Equal(t, expectedVersion, version, "GetVersion should return the correct version string")
}

func TestIssafe(t *testing.T) {
	// Test case: Safe string (all printable characters)
	require.True(t, issafe("Hello, World!"))

	// Test case: Unsafe string (contains non-printable characters)
	require.False(t, issafe("Hello\x00World"))
	require.False(t, issafe("Hello\x1FWorld"))

	// Test case: Empty string
	require.True(t, issafe(""))
}

func TestSanitizeText(t *testing.T) {
	// Test case: Safe string (no changes expected)
	input := "Hello, World!"
	output := SanitizeText(input)
	require.Equal(t, input, output)

	// Test case: Unsafe string (non-printable characters replaced with '?')
	input = "Hello\x00World"
	expected := "Hello?World"
	output = SanitizeText(input)
	require.Equal(t, expected, output)

	// Test case: String with multiple unsafe characters
	input = "Hello\x00\x1FWorld"
	expected = "Hello??World"
	output = SanitizeText(input)
	require.Equal(t, expected, output)

	// Test case: Empty string
	input = ""
	output = SanitizeText(input)
	require.Equal(t, input, output)
}

func TestGetConfigDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override the XDG_CONFIG_HOME environment variable
	originalXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", originalXDGConfigHome) // Restore the original value after the test
	os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Call GetConfigDir with a test app name
	appName := "testapp"
	configDir, err := GetConfigDir(appName)
	require.NoError(t, err)

	// Verify that the config directory is inside the temporary directory
	expectedDir := filepath.Join(tempDir, appName)
	require.Equal(t, expectedDir, configDir)

	// Verify that the directory was created
	_, err = os.Stat(configDir)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(configDir), "Config directory should be an absolute path")
}

func TestGetCacheDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Override the XDG_CACHE_HOME environment variable
	originalXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	defer os.Setenv("XDG_CACHE_HOME", originalXDGCacheHome) // Restore the original value after the test
	os.Setenv("XDG_CACHE_HOME", tempDir)

	// Call GetCacheDir with a test app name
	appName := "testapp"
	cacheDir, err := GetCacheDir(appName)
	require.NoError(t, err)

	// Verify that the cache directory is inside the temporary directory
	expectedDir := filepath.Join(tempDir, appName)
	require.Equal(t, expectedDir, cacheDir)

	// Verify that the directory was created
	_, err = os.Stat(cacheDir)
	require.NoError(t, err)
	require.True(t, filepath.IsAbs(cacheDir), "Cache directory should be an absolute path")
}
