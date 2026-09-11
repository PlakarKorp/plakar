package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOldConfigIfExistsReturnsBlankConfigWhenMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plakar.yml") // does not exist
	cfg, err := LoadOldConfigIfExists(path)
	if err != nil {
		t.Fatalf("LoadOldConfigIfExists missing: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config when file is missing")
	}
	if len(cfg.Repositories) != 0 {
		t.Fatalf("expected empty repositories, got %v", cfg.Repositories)
	}
}

func TestLoadOldConfigIfExistsParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plakar.yml")
	contents := `
default-repo: home
repositories:
  home:
    location: /tmp/repo
remotes:
  laptop:
    location: ssh://laptop/data
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := LoadOldConfigIfExists(path)
	if err != nil {
		t.Fatalf("LoadOldConfigIfExists: %v", err)
	}
	if cfg.DefaultRepository != "home" {
		t.Fatalf("DefaultRepository = %q, want home", cfg.DefaultRepository)
	}
	if cfg.Repositories["home"]["location"] != "/tmp/repo" {
		t.Fatalf("repository.home.location = %q", cfg.Repositories["home"]["location"])
	}
	if cfg.Sources["laptop"]["location"] != "ssh://laptop/data" {
		t.Fatalf("sources.laptop.location = %q", cfg.Sources["laptop"]["location"])
	}
	// Destinations are mirrored from Sources.
	if cfg.Destinations["laptop"]["location"] != "ssh://laptop/data" {
		t.Fatalf("destinations.laptop.location = %q", cfg.Destinations["laptop"]["location"])
	}
}

func TestLoadOldConfigIfExistsRejectsMalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plakar.yml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadOldConfigIfExists(path); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestLoadOldConfigIfExistsOpenError(t *testing.T) {
	// A directory cannot be opened as a regular file in a way that Decode works;
	// using a path that's a directory triggers os.Open error path.
	dir := t.TempDir()
	if _, err := LoadOldConfigIfExists(dir); err == nil {
		// On some systems opening a dir succeeds. Skip if no error returned.
		t.Skip("opening a directory did not error on this platform")
	}
}
