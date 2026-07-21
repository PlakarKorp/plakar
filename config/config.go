package config

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

var (
	ErrNoLocation = errors.New("location is missing")
	ErrNotFound   = errors.New("configuration not found")
)

type Config struct {
	DefaultRepository string
	Repositories      map[string]map[string]string
	Sources           map[string]map[string]string
	Destinations      map[string]map[string]string
}

func NewConfig() *Config {
	return &Config{
		Repositories: make(map[string]map[string]string),
		Sources:      make(map[string]map[string]string),
		Destinations: make(map[string]map[string]string),
	}
}

func (c *Config) HasRepository(name string) bool {
	_, ok := c.Repositories[name]
	return ok
}

func (c *Config) get(name, kind string, configs map[string]map[string]string) (map[string]string, error) {
	if !strings.HasPrefix(name, "@") {
		return map[string]string{"location": name}, nil
	}

	name, rootOverride := resolveRootOverride(name[1:])

	kv, ok := configs[name]
	if !ok {
		return nil, fmt.Errorf("%s %w: %q", kind, ErrNotFound, name)
	}

	if _, ok := kv["location"]; !ok {
		return nil, fmt.Errorf("%s %w", kind, ErrNoLocation)
	}

	return resolve(kv, rootOverride)
}

func (c *Config) GetRepository(name string) (map[string]string, error) {
	return c.get(name, "repository", c.Repositories)
}

func (c *Config) HasSource(name string) bool {
	_, ok := c.Sources[name]
	return ok
}

func (c *Config) GetSource(name string) (map[string]string, error) {
	if c == nil {
		panic("c is nil")
	}
	if c.Sources == nil {
		panic("sources is nil")
	}
	return c.get(name, "source", c.Sources)
}

func (c *Config) HasDestination(name string) bool {
	_, ok := c.Destinations[name]
	return ok
}

func (c *Config) GetDestination(name string) (map[string]string, error) {
	return c.get(name, "source", c.Destinations)
}

func resolveRootOverride(name string) (string, string) {
	if before, after, ok := strings.Cut(name, ":"); !ok {
		return name, ""
	} else {
		return before, after
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
			location = filepath.Join(location, rootOverride)
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
