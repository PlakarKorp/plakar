package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadEmptyDirFallsBackAndReturnsBlank(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := NewConfig()
	cfg.DefaultRepository = "home"
	cfg.Repositories["home"] = map[string]string{"location": "/var/data/repo"}
	cfg.Sources["src"] = map[string]string{"location": "fs:///var/source"}
	cfg.Destinations["dst"] = map[string]string{"location": "fs:///var/dest"}

	if err := Save(dir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// All three files should exist.
	for _, name := range []string{"sources.yml", "destinations.yml", "stores.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}
	if loaded.DefaultRepository != "home" {
		t.Fatalf("DefaultRepository = %q, want home", loaded.DefaultRepository)
	}
	if loaded.Repositories["home"]["location"] != "/var/data/repo" {
		t.Fatalf("repos[home].location = %q", loaded.Repositories["home"]["location"])
	}
	if loaded.Sources["src"]["location"] != "fs:///var/source" {
		t.Fatalf("sources[src].location = %q", loaded.Sources["src"]["location"])
	}
	if loaded.Destinations["dst"]["location"] != "fs:///var/dest" {
		t.Fatalf("destinations[dst].location = %q", loaded.Destinations["dst"]["location"])
	}
}

func TestLoadFallbackFromOldFormat(t *testing.T) {
	dir := t.TempDir()
	old := `
default-repo: legacy
repositories:
  legacy:
    location: /tmp/legacy
remotes:
  src:
    location: fs:///var/src
`
	if err := os.WriteFile(filepath.Join(dir, "plakar.yml"), []byte(old), 0o600); err != nil {
		t.Fatalf("write old config: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultRepository != "legacy" {
		t.Fatalf("DefaultRepository = %q, want legacy", cfg.DefaultRepository)
	}
	if cfg.Repositories["legacy"]["location"] != "/tmp/legacy" {
		t.Fatalf("legacy location = %q", cfg.Repositories["legacy"]["location"])
	}
	// LoadFallback also rewrites the config in new format.
	for _, name := range []string{"sources.yml", "destinations.yml", "stores.yml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to be written by fallback: %v", name, err)
		}
	}
}

func TestLoadKlosetsYmlFallback(t *testing.T) {
	dir := t.TempDir()
	// Provide sources.yml + destinations.yml + the older klosets.yml; loader
	// should fall through to klosets.yml when stores.yml is absent.
	mustWrite := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	mustWrite("sources.yml", "version: v1.0.0\nsources:\n  s:\n    location: fs:///s\n")
	mustWrite("destinations.yml", "version: v1.0.0\ndestinations:\n  d:\n    location: fs:///d\n")
	mustWrite("klosets.yml", "version: v1.0.0\ndefault: r\nstores:\n  r:\n    location: fs:///r\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultRepository != "r" {
		t.Fatalf("DefaultRepository = %q, want r (read from klosets.yml)", cfg.DefaultRepository)
	}
}

func TestLoadBackwardCompatPreviousStoreFormat(t *testing.T) {
	// Old store file format: top-level map keyed by name; an entry can carry
	// `.isDefault: true` to mark itself as default.
	dir := t.TempDir()
	mustWrite := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	mustWrite("sources.yml", "version: v1.0.0\nsources: {}\n")
	mustWrite("destinations.yml", "version: v1.0.0\ndestinations: {}\n")
	mustWrite("stores.yml", "home:\n  location: /var/h\n  .isDefault: \"true\"\nother:\n  location: /var/o\n")

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DefaultRepository != "home" {
		t.Fatalf("DefaultRepository = %q, want home (from .isDefault)", cfg.DefaultRepository)
	}
	if cfg.Repositories["home"]["location"] != "/var/h" {
		t.Fatalf("home.location = %q", cfg.Repositories["home"]["location"])
	}
	if _, has := cfg.Repositories["home"][".isDefault"]; has {
		t.Fatal(".isDefault should have been stripped from the map")
	}
}

func TestLoadMultipleDefaultsIsError(t *testing.T) {
	dir := t.TempDir()
	mustWrite := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	mustWrite("sources.yml", "version: v1.0.0\nsources: {}\n")
	mustWrite("destinations.yml", "version: v1.0.0\ndestinations: {}\n")
	mustWrite("stores.yml", "a:\n  .isDefault: \"true\"\nb:\n  .isDefault: \"true\"\n")

	if _, err := Load(dir); err == nil {
		t.Fatal("expected error for multiple default stores, got nil")
	}
}

func TestLoadINI(t *testing.T) {
	rd := strings.NewReader(`
[section1]
key1 = value1
key2 = value2

[section2]
foo = bar
`)
	got, err := loadINI(rd)
	if err != nil {
		t.Fatalf("LoadINI: %v", err)
	}
	if got["section1"]["key1"] != "value1" || got["section1"]["key2"] != "value2" {
		t.Fatalf("section1 mismatch: %+v", got["section1"])
	}
	if got["section2"]["foo"] != "bar" {
		t.Fatalf("section2 mismatch: %+v", got["section2"])
	}
}

func TestLoadINIBad(t *testing.T) {
	if _, err := loadINI(strings.NewReader("[unclosed")); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestLoadYAML(t *testing.T) {
	rd := strings.NewReader(`
remote:
  location: ssh://host
  port: 22
  ssl: true
`)
	got, err := loadYAML(rd)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if got["remote"]["location"] != "ssh://host" {
		t.Fatalf("location = %q", got["remote"]["location"])
	}
	if got["remote"]["port"] != "22" {
		t.Fatalf("port = %q (toString should convert int)", got["remote"]["port"])
	}
	if got["remote"]["ssl"] != "true" {
		t.Fatalf("ssl = %q (toString should convert bool)", got["remote"]["ssl"])
	}
}

func TestLoadYAMLSkipsScalarTopLevel(t *testing.T) {
	// Top-level scalar keys are skipped rather than producing an error.
	rd := strings.NewReader("version: v1.0.0\nremote:\n  location: x\n")
	got, err := loadYAML(rd)
	if err != nil {
		t.Fatalf("LoadYAML: %v", err)
	}
	if _, has := got["version"]; has {
		t.Fatal("scalar key 'version' should be skipped")
	}
	if got["remote"]["location"] != "x" {
		t.Fatalf("remote.location = %q", got["remote"]["location"])
	}
}

func TestLoadYAMLBad(t *testing.T) {
	if _, err := loadYAML(strings.NewReader("not: [valid: yaml")); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestLoadJSON(t *testing.T) {
	rd := strings.NewReader(`{"a":{"k":"v"},"b":{"x":"y"}}`)
	got, err := loadJSON(rd)
	if err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got["a"]["k"] != "v" || got["b"]["x"] != "y" {
		t.Fatalf("LoadJSON: %+v", got)
	}
}

func TestLoadJSONBad(t *testing.T) {
	if _, err := loadJSON(strings.NewReader("nope")); err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestToString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"hello", "hello"},
		{42, "42"},
		{int64(7), "7"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
		{[]int{1, 2}, ""},
	}
	for _, c := range cases {
		if got := toString(c.in); got != c.want {
			t.Errorf("toString(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLoadFileYAMLWithLocation(t *testing.T) {
	rd := strings.NewReader(`
remote:
  location: ssh://host
  port: 22
`)
	got, err := LoadFile(rd, "")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got["remote"]["location"] != "ssh://host" {
		t.Fatalf("location = %q", got["remote"]["location"])
	}
	if got["remote"]["port"] != "22" {
		t.Fatalf("port = %q", got["remote"]["port"])
	}
}

func TestLoadFileYAMLMissingLocationIsError(t *testing.T) {
	rd := strings.NewReader(`
remote:
  port: 22
`)
	if _, err := LoadFile(rd, ""); err == nil {
		t.Fatal("expected error for missing 'location', got nil")
	}
}

func TestLoadFileThirdPartyRewritesKeys(t *testing.T) {
	rd := strings.NewReader(`
remote:
  host: example.com
  port: 22
`)
	got, err := LoadFile(rd, "rclone")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got["remote"]["location"] != "rclone://" {
		t.Fatalf("location = %q, want rclone://", got["remote"]["location"])
	}
	if got["remote"]["rclone_host"] != "example.com" {
		t.Fatalf("rclone_host = %q", got["remote"]["rclone_host"])
	}
	if got["remote"]["rclone_port"] != "22" {
		t.Fatalf("rclone_port = %q", got["remote"]["rclone_port"])
	}
	// Original keys are stripped.
	if _, has := got["remote"]["host"]; has {
		t.Fatal("original key 'host' should be stripped")
	}
}

func TestLoadFileINISource(t *testing.T) {
	rd := strings.NewReader(`
[remote]
location = fs:///x
`)
	got, err := LoadFile(rd, "")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if got["remote"]["location"] != "fs:///x" {
		t.Fatalf("location = %q", got["remote"]["location"])
	}
}

func TestLoadFileStripsEmptyValues(t *testing.T) {
	rd := strings.NewReader(`
remote:
  location: fs:///x
  empty: ""
`)
	got, err := LoadFile(rd, "")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if _, has := got["remote"]["empty"]; has {
		t.Fatal("empty value should have been stripped")
	}
}
