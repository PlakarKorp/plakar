package config

import (
	"fmt"
	"maps"
	"os"

	"go.yaml.in/yaml/v3"
)

type OldConfig struct {
	DefaultRepository string                      `yaml:"default-repo"`
	Repositories      map[string]RepositoryConfig `yaml:"repositories"`
	Remotes           map[string]SourceConfig     `yaml:"remotes"`
}

func LoadOldConfigIfExists(configFile string) (*Config, error) {
	cfg := NewConfig()

	f, err := os.Open(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("error reading old config file: %w", err)
	}
	defer f.Close()

	var old OldConfig
	if err := yaml.NewDecoder(f).Decode(&old); err != nil {
		return nil, fmt.Errorf("failed to parse old config file: %w", err)
	}

	cfg.DefaultRepository = old.DefaultRepository
	cfg.Repositories = old.Repositories
	cfg.Sources = old.Remotes
	cfg.Destinations = make(map[string]DestinationConfig)
	for key, val := range cfg.Sources {
		res := make(map[string]string)
		maps.Copy(res, val)
		cfg.Destinations[key] = res
	}

	return cfg, nil
}
