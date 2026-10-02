package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/pkg"
	"github.com/stretchr/testify/require"
)

func TestRegistriesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := NewConfig()
	cfg.Registries["lab"] = RegistryConfig{URL: "https://lab.example.org/"}
	cfg.Registries["acme"] = RegistryConfig{URL: "https://acme.example.org/dist/"}
	require.NoError(t, Save(dir, cfg))

	data, err := os.ReadFile(filepath.Join(dir, "registries.yml"))
	require.NoError(t, err)
	require.Contains(t, string(data), "version: "+CONFIG_VERSION)
	require.Contains(t, string(data), "url: https://lab.example.org/")

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.Equal(t, cfg.Registries, loaded.Registries)
	require.Equal(t, []pkg.Registry{
		{Name: "acme", URL: "https://acme.example.org/dist/"},
		{Name: "lab", URL: "https://lab.example.org/"},
	}, loaded.PkgRegistries())
}

func TestRegistriesMissingFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg := NewConfig()
	cfg.Sources["src"] = map[string]string{"location": "fs:///new"}
	require.NoError(t, Save(dir, cfg))
	require.NoFileExists(t, filepath.Join(dir, "registries.yml"))

	// A missing registries.yml must not trigger the fallback to the
	// old configuration file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plakar.yml"),
		[]byte("sources:\n  old:\n    location: fs:///old\n"), 0o600))

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.Empty(t, loaded.Registries)
	require.Empty(t, loaded.PkgRegistries())
	require.Contains(t, loaded.Sources, "src")
	require.NotContains(t, loaded.Sources, "old")
}

func TestRegistriesRemovedWithTheLast(t *testing.T) {
	dir := t.TempDir()
	cfg := NewConfig()
	cfg.Registries["lab"] = RegistryConfig{URL: "https://lab.example.org/"}
	require.NoError(t, Save(dir, cfg))
	require.FileExists(t, filepath.Join(dir, "registries.yml"))

	delete(cfg.Registries, "lab")
	require.NoError(t, Save(dir, cfg))
	require.NoFileExists(t, filepath.Join(dir, "registries.yml"))
}

func TestRegistriesEmptyFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, Save(dir, NewConfig()))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registries.yml"), nil, 0o600))

	loaded, err := Load(dir)
	require.NoError(t, err)
	require.Empty(t, loaded.Registries)
}

func TestRegistriesBadFile(t *testing.T) {
	for name, content := range map[string]string{
		"not yaml":      "registries: [\n",
		"wrong version": "version: v0.0.1\nregistries:\n  lab:\n    url: https://lab.example.org/\n",
		"no version":    "lab:\n  url: https://lab.example.org/\n",
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, Save(dir, NewConfig()))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "registries.yml"), []byte(content), 0o600))

			_, err := Load(dir)
			require.Error(t, err)
			require.Contains(t, err.Error(), "registries.yml")
		})
	}
}

// A config dir holding only registries.yml goes through the fallback on
// the first Load, which must keep the registries.
func TestRegistriesSurviveFallback(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "registries.yml"),
		[]byte("version: "+CONFIG_VERSION+"\nregistries:\n  lab:\n    url: https://lab.example.org/\n"), 0o600))

	want := map[string]RegistryConfig{"lab": {URL: "https://lab.example.org/"}}
	for range 2 {
		loaded, err := Load(dir)
		require.NoError(t, err)
		require.Equal(t, want, loaded.Registries)
		require.FileExists(t, filepath.Join(dir, "registries.yml"))
	}
}
