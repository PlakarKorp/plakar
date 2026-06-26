package utils

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/PlakarKorp/plakar/config"
	"github.com/stretchr/testify/require"
)

// failWriter always fails on Write, to exercise encoder error paths.
type failWriter struct{}

func (failWriter) Write(p []byte) (int, error) { return 0, errors.New("boom") }

func TestLoadConfigMalformedSourcesReturnsError(t *testing.T) {
	dir := t.TempDir()
	// A malformed sources.yml that is neither valid new nor old format makes
	// load() return a non-IsNotExist error, which Load() propagates.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("sources: [unterminated"), 0600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse config file")
}

func TestLoadConfigMalformedDestinationsReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destinations.yml"),
		[]byte("destinations: [unterminated"), 0600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
}

func TestLoadConfigMalformedStoresReturnsError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destinations.yml"),
		[]byte("version: "+CONFIG_VERSION+"\ndestinations: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stores.yml"),
		[]byte("stores: [unterminated"), 0600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
}

func TestLoadConfigEmptyFilesAreTreatedAsEmpty(t *testing.T) {
	dir := t.TempDir()
	// Zero-length files exercise the info.Size()==0 early-return in load().
	for _, f := range []string{"sources.yml", "destinations.yml", "stores.yml"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, f), nil, 0600))
	}
	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Empty(t, cfg.Repositories)
	require.Empty(t, cfg.Sources)
}

func TestLoadFallbackInvalidOldConfig(t *testing.T) {
	dir := t.TempDir()
	// No new-format files; the old plakar.yml is malformed, so LoadFallback
	// surfaces the read error.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plakar.yml"),
		[]byte("default-repo: [unterminated"), 0600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error reading file")
}

func TestSaveConfigMkdirFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission semantics differ on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission checks")
	}
	parent := t.TempDir()
	// Make the parent unwritable so MkdirAll of a child fails.
	require.NoError(t, os.Chmod(parent, 0500))
	t.Cleanup(func() { os.Chmod(parent, 0700) })

	cfg := config.NewConfig()
	err := SaveConfig(filepath.Join(parent, "sub", "cfg"), cfg)
	require.Error(t, err)
}

func TestGetConfReadError(t *testing.T) {
	_, err := GetConf(failReader{}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to read config data")
}

type failReader struct{}

func (failReader) Read(p []byte) (int, error) { return 0, errors.New("read boom") }

func TestGetConfUnparseable(t *testing.T) {
	// Content that fails YAML, JSON and INI parsing.
	_, err := GetConf(bytes.NewReader([]byte("\x00\x01\x02not valid anything\x00")), "")
	require.Error(t, err)
}

func TestPolicySetTimeUnsetKeepsZero(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	require.NoError(t, c.Set("p", "since", "2025-01-02"))
	require.False(t, c.Policies["p"].Filters.Since.IsZero())
	require.NoError(t, c.Unset("p", "since"))
	require.True(t, c.Policies["p"].Filters.Since.IsZero())
}

func TestPolicyDumpEncodeError(t *testing.T) {
	c := newPolicies()
	c.Add("p")
	err := c.Dump(failWriter{}, "json", []string{"p"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to encode")
}

func TestPolicySaveToFileBadDir(t *testing.T) {
	// A filename whose directory does not exist makes CreateTemp fail.
	err := newPolicies().SaveToFile(filepath.Join(t.TempDir(), "no-such-dir", "p.yml"))
	require.Error(t, err)
}

func TestPolicyUnsetMissingEntry(t *testing.T) {
	c := newPolicies()
	err := c.Unset("missing", "days")
	require.Error(t, err)
	require.Contains(t, err.Error(), "entry not found")
}

func TestToStringViaLoadYAML(t *testing.T) {
	// Exercises the default branch of toString (a nested object value gets
	// stringified to "").
	data := "remote:\n  nested:\n    a: b\n  ok: value\n"
	res, err := LoadYAML(bytes.NewReader([]byte(data)))
	require.NoError(t, err)
	require.Equal(t, "value", res["remote"]["ok"])
	require.Equal(t, "", res["remote"]["nested"])
}
