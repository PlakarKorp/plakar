package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"maps"
)

const repositoryEnvPrefix = "PLAKAR_REPOSITORIES_"

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

	repositoryName := name[1:]
	kv, ok := c.Repositories[repositoryName]
	if !ok {
		return nil, fmt.Errorf("could not resolve repository: %s", name)
	}

	res := make(map[string]string)
	maps.Copy(res, kv)

	if _, ok := res["location"]; !ok {
		return nil, fmt.Errorf("repository %s has no location", name)
	}

	location, err := applyRootOverride(res["location"], rootOverride)
	if err != nil {
		return nil, err
	}
	res["location"] = location
	return res, nil
}

func (c *Config) GetRepositoryPassphrase(name string) (string, bool, error) {
	return c.repositoryEnvironmentValue(name, "PASSPHRASE")
}

func (c *Config) GetRepositoryPassphraseCommand(name string) (string, bool, error) {
	return c.repositoryEnvironmentValue(name, "PASSPHRASE_CMD")
}

func (c *Config) repositoryEnvironmentValue(name string, key string) (string, bool, error) {
	if !strings.HasPrefix(name, "@") {
		return "", false, nil
	}

	name, _ = resolveRootOverride(name)
	repositoryName := name[1:]
	envKey := repositoryEnvPrefix + encodeRepositoryEnvironmentName(repositoryName) + "_" + key
	value, ok := os.LookupEnv(envKey)
	if !ok {
		return "", false, nil
	}
	if c.repositoryEnvironmentNameAmbiguous(repositoryName) {
		return "", false, fmt.Errorf("ambiguous environment override for repository %q", repositoryName)
	}
	return value, ok, nil
}

func (c *Config) repositoryEnvironmentNameAmbiguous(name string) bool {
	envName := encodeRepositoryEnvironmentName(name)
	for other := range c.Repositories {
		if other != name && encodeRepositoryEnvironmentName(other) == envName {
			return true
		}
	}
	return false
}

func encodeRepositoryEnvironmentName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
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
