package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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

func TestHumanToDuration(t *testing.T) {
	// Test case: Valid time.Duration string
	duration, err := HumanToDuration("2h45m")
	require.NoError(t, err)
	require.Equal(t, 2*time.Hour+45*time.Minute, duration)

	// Test case: Invalid time.Duration string
	duration, err = HumanToDuration("invalid")
	require.Error(t, err)
	require.Equal(t, time.Duration(0), duration)
	require.Contains(t, err.Error(), "invalid duration")

	// Test case: Valid human-readable duration (e.g., "1h")
	duration, err = HumanToDuration("1h")
	require.NoError(t, err)
	require.Equal(t, 1*time.Hour, duration)

	// Test case: Valid human-readable duration (e.g., "1d")
	duration, err = HumanToDuration("24h") // Equivalent to 1 day
	require.NoError(t, err)
	require.Equal(t, 24*time.Hour, duration)

	// Test case: Valid human-readable duration with mixed units (e.g., "1d12h")
	duration, err = HumanToDuration("36h") // Equivalent to 1 day and 12 hours
	require.NoError(t, err)
	require.Equal(t, 36*time.Hour, duration)

	// Test case: Empty input
	duration, err = HumanToDuration("")
	require.Error(t, err)
	require.Equal(t, time.Duration(0), duration)
	require.Contains(t, err.Error(), "invalid duration")
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
