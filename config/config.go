package config

import (
	"fmt"
	"maps"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/PlakarKorp/pkg"
)

type Config struct {
	DefaultRepository string
	Repositories      map[string]RepositoryConfig
	Sources           map[string]SourceConfig
	Destinations      map[string]DestinationConfig
	Registries        map[string]RegistryConfig
}

type RepositoryConfig = map[string]string
type SourceConfig = map[string]string
type DestinationConfig = map[string]string

// RegistryConfig is an additional package registry.
type RegistryConfig struct {
	URL string `yaml:"url"`
}

func NewConfig() *Config {
	return &Config{
		Repositories: make(map[string]RepositoryConfig),
		Sources:      make(map[string]SourceConfig),
		Destinations: make(map[string]DestinationConfig),
		Registries:   make(map[string]RegistryConfig),
	}
}

// PkgRegistries returns the additional package registries sorted by name,
// the order in which the package manager queries them.
func (c *Config) PkgRegistries() []pkg.Registry {
	var regs []pkg.Registry
	for _, name := range slices.Sorted(maps.Keys(c.Registries)) {
		regs = append(regs, pkg.Registry{Name: name, URL: c.Registries[name].URL})
	}
	return regs
}

func (c *Config) HasRepository(name string) bool {
	_, ok := c.Repositories[name]
	return ok
}

func (c *Config) GetRepository(name string) (map[string]string, error) {
	if !strings.HasPrefix(name, "@") {
		return map[string]string{"location": name}, nil
	}

	name, rootOverride := resolveRootOverride(name)

	kv, ok := c.Repositories[name[1:]]
	if !ok {
		return nil, fmt.Errorf("could not resolve repository: %s", name)
	}
	if _, ok := kv["location"]; !ok {
		return nil, fmt.Errorf("repository %s has no location", name)
	} else {
		res := make(map[string]string)
		maps.Copy(res, kv)

		location, err := applyRootOverride(res["location"], rootOverride)
		if err != nil {
			return nil, err
		}
		res["location"] = location
		return res, nil
	}
}

func (c *Config) HasSource(name string) bool {
	_, ok := c.Sources[name]
	return ok
}

func (c *Config) GetSource(name string) (map[string]string, bool) {
	name, rootOverride := resolveRootOverride(name)

	if kv, ok := c.Sources[name]; !ok {
		return nil, false
	} else {
		res := make(map[string]string)
		maps.Copy(res, kv)

		location, err := applyRootOverride(res["location"], rootOverride)
		if err != nil {
			return nil, false
		}
		res["location"] = location
		return res, ok
	}
}

func (c *Config) HasDestination(name string) bool {
	_, ok := c.Destinations[name]
	return ok
}

func (c *Config) GetDestination(name string) (map[string]string, bool) {
	name, rootOverride := resolveRootOverride(name)

	if kv, ok := c.Destinations[name]; !ok {
		return nil, false
	} else {
		res := make(map[string]string)
		maps.Copy(res, kv)

		location, err := applyRootOverride(res["location"], rootOverride)
		if err != nil {
			return nil, false
		}
		res["location"] = location
		return res, ok
	}
}

func resolveRootOverride(name string) (string, string) {
	if idx := strings.Index(name, ":"); idx == -1 {
		return name, ""
	} else {
		return name[:idx], name[idx+1:]
	}
}

func applyRootOverride(location string, rootOverride string) (string, error) {
	if rootOverride == "" {
		return location, nil
	}

	localPath := strings.HasPrefix(location, "/")

	if localPath {
		if strings.HasPrefix(rootOverride, "/") {
			location = rootOverride
		} else {
			location = path.Join(location, rootOverride)
		}
	} else {
		u, err := url.Parse(location)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(rootOverride, "/") {
			u.Path = rootOverride
		} else {
			u.Path = path.Join(u.Path, rootOverride)
		}
		location = u.String()
	}
	return location, nil
}
