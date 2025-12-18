package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const CONFIG_VERSION = "v1.0.0"

type (
	storesConfig struct {
		Version string                       `yaml:"version"`
		Default string                       `yaml:"default,omitempty"`
		Stores  map[string]map[string]string `yaml:"stores"`
	}

	sourcesConfig struct {
		Version string                       `yaml:"version"`
		Sources map[string]map[string]string `yaml:"sources"`
	}

	destinationsConfig struct {
		Version      string                       `yaml:"version"`
		Destinations map[string]map[string]string `yaml:"destinations"`
	}
)

func load(file string, dst any) error {
	f, err := os.Open(file)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("failed to open file %s: %w", file, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", file, err)
	}
	if info.Size() == 0 {
		return nil
	}

	// try to load the new format
	err = yaml.NewDecoder(f).Decode(dst)
	var version string
	switch t := dst.(type) {
	case *storesConfig:
		version = t.Version
	case *destinationsConfig:
		version = t.Version
	case *sourcesConfig:
		version = t.Version
	default:
		return fmt.Errorf("invalid configuration type %v", t)
	}
	if err == nil && version == CONFIG_VERSION {
		return nil
	}

	// fallback to the previous format
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to rewind config file: %w", err)
	}

	switch t := dst.(type) {
	case *storesConfig:
		err = yaml.NewDecoder(f).Decode(&t.Stores)
		if err == nil {
			for k, v := range t.Stores {
				if _, ok := v[".isDefault"]; ok {
					if t.Default != "" {
						return fmt.Errorf("multiple default store")
					}
					t.Default = k
					delete(v, ".isDefault")
				}
			}
		}
	case *destinationsConfig:
		err = yaml.NewDecoder(f).Decode(&t.Destinations)
	case *sourcesConfig:
		err = yaml.NewDecoder(f).Decode(&t.Sources)
	}
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

func loadFallback(dir string) (*Config, error) {
	// Load old config if found
	oldpath := filepath.Join(dir, "plakar.yml")
	cfg, err := LoadOldConfigIfExists(oldpath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", oldpath, err)
	}

	// Save the config in the new format right now
	if err := Save(dir, cfg); err != nil {
		return nil, fmt.Errorf("failed to update config file: %w", err)
	}
	// Do we want to remove the old file?
	return cfg, nil
}

func Load(dir string) (*Config, error) {
	sources := sourcesConfig{}
	destinations := destinationsConfig{}
	stores := storesConfig{}

	err := load(filepath.Join(dir, "sources.yml"), &sources)
	if err != nil {
		if os.IsNotExist(err) {
			return loadFallback(dir)
		}
		return nil, err
	}

	err = load(filepath.Join(dir, "destinations.yml"), &destinations)
	if err != nil {
		if os.IsNotExist(err) {
			return loadFallback(dir)
		}
		return nil, err
	}

	err = load(filepath.Join(dir, "stores.yml"), &stores)
	if err != nil && os.IsNotExist(err) {
		// try to load former file
		err = load("klosets.yml", &stores)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return loadFallback(dir)
		}
		return nil, err
	}

	cfg := NewConfig()
	cfg.Sources = sources.Sources
	cfg.Destinations = destinations.Destinations
	cfg.Repositories = stores.Stores
	cfg.DefaultRepository = stores.Default

	return cfg, nil
}

func save(file string, src any) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(file), "config.*.yml")
	if err != nil {
		return err
	}

	err = yaml.NewEncoder(tmpFile).Encode(src)
	tmpFile.Close()

	if err == nil {
		err = os.Rename(tmpFile.Name(), file)
	}

	if err != nil {
		os.Remove(tmpFile.Name())
		return err
	}

	return nil
}

func Save(dir string, cfg *Config) error {
	err := save(filepath.Join(dir, "sources.yml"), sourcesConfig{
		Version: CONFIG_VERSION,
		Sources: cfg.Sources,
	})
	if err != nil {
		return err
	}
	err = save(filepath.Join(dir, "destinations.yml"), destinationsConfig{
		Version:      CONFIG_VERSION,
		Destinations: cfg.Destinations,
	})
	if err != nil {
		return err
	}
	err = save(filepath.Join(dir, "stores.yml"), storesConfig{
		Version: CONFIG_VERSION,
		Default: cfg.DefaultRepository,
		Stores:  cfg.Repositories,
	})
	if err != nil {
		return err
	}
	return nil
}
