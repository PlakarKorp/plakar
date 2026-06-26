package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	cfg.DefaultRepository = "main"
	cfg.Repositories["main"] = map[string]string{"location": "/var/backups"}
	cfg.Sources["laptop"] = map[string]string{"location": "fs:///home"}
	cfg.Destinations["restore"] = map[string]string{"location": "fs:///restore"}

	require.NoError(t, SaveConfig(dir, cfg))

	// Files were written in the new format.
	for _, f := range []string{"sources.yml", "destinations.yml", "stores.yml"} {
		_, err := os.Stat(filepath.Join(dir, f))
		require.NoError(t, err, "expected %s to exist", f)
	}

	loaded, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "main", loaded.DefaultRepository)
	require.Equal(t, "/var/backups", loaded.Repositories["main"]["location"])
	require.Equal(t, "fs:///home", loaded.Sources["laptop"]["location"])
	require.Equal(t, "fs:///restore", loaded.Destinations["restore"]["location"])
}

func TestLoadConfigFallbackFromOldFile(t *testing.T) {
	dir := t.TempDir()

	// No new-format files exist, but an old plakar.yml does.
	old := `default-repo: main
repositories:
  main:
    location: /var/backups
remotes:
  src:
    location: fs:///home
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plakar.yml"), []byte(old), 0600))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultRepository)
	require.Equal(t, "/var/backups", cfg.Repositories["main"]["location"])
	require.Equal(t, "fs:///home", cfg.Sources["src"]["location"])

	// The fallback path migrates the config to the new format on disk.
	_, err = os.Stat(filepath.Join(dir, "stores.yml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "sources.yml"))
	require.NoError(t, err)
}

func TestLoadConfigEmptyDirGivesEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "", cfg.DefaultRepository)
	require.Empty(t, cfg.Repositories)
}

func TestLoadConfigLegacyKlosetsFile(t *testing.T) {
	dir := t.TempDir()

	// New-format sources/destinations present...
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destinations.yml"),
		[]byte("version: "+CONFIG_VERSION+"\ndestinations: {}\n"), 0600))
	// ...but the stores live in the legacy klosets.yml file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "klosets.yml"),
		[]byte("version: "+CONFIG_VERSION+"\ndefault: main\nstores:\n  main:\n    location: /repo\n"), 0600))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultRepository)
	require.Equal(t, "/repo", cfg.Repositories["main"]["location"])
}

func TestLoadConfigOldFormatStoresWithDefaultMarker(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destinations.yml"),
		[]byte("version: "+CONFIG_VERSION+"\ndestinations: {}\n"), 0600))

	// Old-format stores file (no version, top-level is the store map, with
	// a .isDefault marker). This exercises the fallback decode path.
	stores := `main:
  location: /repo
  .isDefault: "true"
backup:
  location: /backup
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stores.yml"), []byte(stores), 0600))

	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultRepository)
	require.Equal(t, "/repo", cfg.Repositories["main"]["location"])
	require.Equal(t, "/backup", cfg.Repositories["backup"]["location"])
	// The marker key is stripped out.
	_, ok := cfg.Repositories["main"][".isDefault"]
	require.False(t, ok)
}

func TestLoadConfigOldFormatMultipleDefaultsErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sources.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nsources: {}\n"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "destinations.yml"),
		[]byte("version: "+CONFIG_VERSION+"\ndestinations: {}\n"), 0600))

	stores := `main:
  location: /repo
  .isDefault: "true"
backup:
  location: /backup
  .isDefault: "true"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stores.yml"), []byte(stores), 0600))

	_, err := LoadConfig(dir)
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple default store")
}
