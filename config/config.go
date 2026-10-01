package config

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"maps"
)

type Config struct {
	DefaultRepository string
	Repositories      map[string]RepositoryConfig
	Sources           map[string]SourceConfig
	Destinations      map[string]DestinationConfig
}

type RepositoryConfig = map[string]string
type SourceConfig = map[string]string
type DestinationConfig = map[string]string

func NewConfig() *Config {
	return &Config{
		Repositories: make(map[string]RepositoryConfig),
		Sources:      make(map[string]SourceConfig),
		Destinations: make(map[string]DestinationConfig),
	}
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
