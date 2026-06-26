package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOldConfigIfExistsMissing(t *testing.T) {
	// Non-existent file returns a fresh, empty config without error.
	cfg, err := LoadOldConfigIfExists(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.Repositories)
	require.Empty(t, cfg.Sources)
}

func TestLoadOldConfigIfExistsValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plakar.yml")
	content := `default-repo: main
repositories:
  main:
    location: /var/backups
remotes:
  laptop:
    location: fs:///home/user
    foo: bar
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	cfg, err := LoadOldConfigIfExists(path)
	require.NoError(t, err)
	require.Equal(t, "main", cfg.DefaultRepository)
	require.Equal(t, "/var/backups", cfg.Repositories["main"]["location"])

	// Remotes become Sources.
	require.Equal(t, "fs:///home/user", cfg.Sources["laptop"]["location"])
	require.Equal(t, "bar", cfg.Sources["laptop"]["foo"])

	// Destinations are derived (copied) from the sources.
	require.Equal(t, "fs:///home/user", cfg.Destinations["laptop"]["location"])
	require.Equal(t, "bar", cfg.Destinations["laptop"]["foo"])

	// The destination copy is independent of the source map.
	cfg.Destinations["laptop"]["foo"] = "changed"
	require.Equal(t, "bar", cfg.Sources["laptop"]["foo"])
}

func TestLoadOldConfigIfExistsInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plakar.yml")
	require.NoError(t, os.WriteFile(path, []byte("default-repo: [unterminated"), 0600))

	_, err := LoadOldConfigIfExists(path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse old config file")
}
