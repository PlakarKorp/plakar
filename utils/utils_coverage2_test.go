package utils

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// --- utils.go: XDG env-driven dir resolution (the else branch) -----------

func TestGetCacheDirFromXDGEnv2(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", base)
	t.Setenv("LocalAppData", base)
	got, err := GetCacheDir("myapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "myapp"), got)
}

func TestGetConfigDirFromXDGEnv2(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("LocalAppData", base)
	got, err := GetConfigDir("myapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "myapp"), got)
}

func TestGetDataDirFromXDGEnv2(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_DATA_HOME", base)
	t.Setenv("LocalAppData", base)
	got, err := GetDataDir("myapp")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(base, "myapp"), got)
}

// --- utils.go: shouldCheckUpdate ----------------------------------------

func TestShouldCheckUpdateDevelFalse2(t *testing.T) {
	// VERSION contains "devel" in this build, so it always returns false
	// and never touches the cache dir.
	if !strings.Contains(VERSION, "devel") {
		t.Skip("release build, devel branch not exercised")
	}
	require.False(t, shouldCheckUpdate(t.TempDir()))
}

// --- utils.go: SanitizeText / issafe extra branches ----------------------

func TestSanitizeTextReplacesNonPrintable2(t *testing.T) {
	in := "ab\x00cd\x07ef"
	got := SanitizeText(in)
	require.Equal(t, "ab?cd?ef", got)
	require.False(t, issafe(in))
}

func TestSanitizeTextSafePassThrough2(t *testing.T) {
	in := "all printable 123 -_/"
	require.True(t, issafe(in))
	require.Equal(t, in, SanitizeText(in))
}

// --- utils.go: ParseSnapshotID -------------------------------------------

func TestParseSnapshotIDPatternNormalization2(t *testing.T) {
	prefix, pattern := ParseSnapshotID("abcd:foo/bar")
	require.Equal(t, "abcd", prefix)
	require.Equal(t, "/foo/bar", pattern)

	// already-absolute pattern is left untouched
	prefix, pattern = ParseSnapshotID("abcd:/already/abs")
	require.Equal(t, "abcd", prefix)
	require.Equal(t, "/already/abs", pattern)

	// no colon => no pattern
	prefix, pattern = ParseSnapshotID("justid")
	require.Equal(t, "justid", prefix)
	require.Equal(t, "", pattern)
}

// --- utils.go: ValidateEmail ---------------------------------------------

func TestValidateEmailNameAddrRejected2(t *testing.T) {
	// "Name <addr>" parses but Address != input -> error
	_, err := ValidateEmail("Bob <bob@example.com>")
	require.Error(t, err)
}

func TestValidateEmailOK2(t *testing.T) {
	got, err := ValidateEmail("bob@example.com")
	require.NoError(t, err)
	require.Equal(t, "bob@example.com", got)
}

// --- config_policy.go: SaveToFile + LoadPolicyConfigFile round trip ------

func TestPolicySaveLoadAndDumpRoundTrip2(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "policies.yml")

	cfg, err := LoadPolicyConfigFile(file) // missing -> empty
	require.NoError(t, err)
	require.False(t, cfg.Has("daily"))

	cfg.Add("daily")
	require.NoError(t, cfg.Set("daily", "days", "7"))
	require.NoError(t, cfg.Set("daily", "name", "nightly"))
	require.NoError(t, cfg.SaveToFile(file))

	reloaded, err := LoadPolicyConfigFile(file)
	require.NoError(t, err)
	require.True(t, reloaded.Has("daily"))

	var buf bytes.Buffer
	require.NoError(t, reloaded.Dump(&buf, "yaml", []string{"daily"}))
	require.Contains(t, buf.String(), "nightly")
}

func TestPolicySetTimeAndBool2(t *testing.T) {
	cfg, err := LoadPolicyConfigFile(filepath.Join(t.TempDir(), "m.yml"))
	require.NoError(t, err)
	cfg.Add("p")
	require.NoError(t, cfg.Set("p", "before", "2024-01-02"))
	require.NoError(t, cfg.Set("p", "since", "2023-01-01"))
	require.NoError(t, cfg.Set("p", "latest", "true"))
	// invalid time
	require.Error(t, cfg.Set("p", "before", "not-a-time"))
	// invalid bool
	require.Error(t, cfg.Set("p", "latest", "maybe"))
}

func TestPolicyUnsetUnknownPolicy2(t *testing.T) {
	cfg, err := LoadPolicyConfigFile(filepath.Join(t.TempDir(), "m.yml"))
	require.NoError(t, err)
	require.Error(t, cfg.Unset("nope", "days"))
}

func TestPolicySaveToFileBadDir2(t *testing.T) {
	cfg, err := LoadPolicyConfigFile(filepath.Join(t.TempDir(), "m.yml"))
	require.NoError(t, err)
	cfg.Add("p")
	// directory component does not exist -> CreateTemp fails
	err = cfg.SaveToFile(filepath.Join(t.TempDir(), "no-such-dir", "policies.yml"))
	require.Error(t, err)
}

func TestPolicyDumpJSONAllNames2(t *testing.T) {
	cfg, err := LoadPolicyConfigFile(filepath.Join(t.TempDir(), "missing.yml"))
	require.NoError(t, err)
	cfg.Add("a")
	require.NoError(t, cfg.Set("a", "days", "3"))

	var buf bytes.Buffer
	// nil names -> dump everything
	require.NoError(t, cfg.Dump(&buf, "json", nil))
	require.Contains(t, buf.String(), "\"a\"")
}
