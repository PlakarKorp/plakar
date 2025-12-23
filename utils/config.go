package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/ini.v1"

	"github.com/PlakarKorp/plakar/config"
	"go.yaml.in/yaml/v3"
)

type configHandler struct {
	Path string
}

const CONFIG_VERSION = "v1.0.0"

type storesConfig struct {
	Version string                       `yaml:"version"`
	Default string                       `yaml:"default,omitempty"`
	Stores  map[string]map[string]string `yaml:"stores"`
}

type sourcesConfig struct {
	Version string                       `yaml:"version"`
	Sources map[string]map[string]string `yaml:"sources"`
}

type destinationsConfig struct {
	Version      string                       `yaml:"version"`
	Destinations map[string]map[string]string `yaml:"destinations"`
}

func newConfigHandler(path string) *configHandler {
	return &configHandler{
		Path: path,
	}
}

func (cl *configHandler) Load() (*config.Config, error) {
	sources := sourcesConfig{}
	destinations := destinationsConfig{}
	stores := storesConfig{}

	err := cl.load("sources.yml", &sources)
	if err != nil {
		if os.IsNotExist(err) {
			return cl.LoadFallback()
		}
		return nil, err
	}

	err = cl.load("destinations.yml", &destinations)
	if err != nil {
		if os.IsNotExist(err) {
			return cl.LoadFallback()
		}
		return nil, err
	}

	err = cl.load("stores.yml", &stores)
	if err != nil && os.IsNotExist(err) {
		// try to load former file
		err = cl.load("klosets.yml", &stores)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return cl.LoadFallback()
		}
		return nil, err
	}

	cfg := config.NewConfig()
	cfg.Sources = sources.Sources
	cfg.Destinations = destinations.Destinations
	cfg.Repositories = stores.Stores
	cfg.DefaultRepository = stores.Default
	return cfg, nil
}

func (cl *configHandler) LoadFallback() (*config.Config, error) {
	// Load old config if found
	oldpath := filepath.Join(cl.Path, "plakar.yml")
	cfg, err := LoadOldConfigIfExists(oldpath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", oldpath, err)
	}

	// Save the config in the new format right now
	err = SaveConfig(cl.Path, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update config file: %w", err)
	}
	// Do we want to remove the old file?
	return cfg, nil
}

func (cl *configHandler) Save(cfg *config.Config) error {
	err := cl.save("sources.yml", sourcesConfig{
		Version: CONFIG_VERSION,
		Sources: cfg.Sources,
	})
	if err != nil {
		return err
	}
	err = cl.save("destinations.yml", destinationsConfig{
		Version:      CONFIG_VERSION,
		Destinations: cfg.Destinations,
	})
	if err != nil {
		return err
	}
	err = cl.save("stores.yml", storesConfig{
		Version: CONFIG_VERSION,
		Default: cfg.DefaultRepository,
		Stores:  cfg.Repositories,
	})
	if err != nil {
		return err
	}
	return nil
}

func (cl *configHandler) load(filename string, dst any) error {
	path := filepath.Join(cl.Path, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", path, err)
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

func (cl *configHandler) save(filename string, src any) error {
	path := filepath.Join(cl.Path, filename)
	tmpFile, err := os.CreateTemp(cl.Path, "config.*.yml")
	if err != nil {
		return err
	}

	err = yaml.NewEncoder(tmpFile).Encode(src)
	tmpFile.Close()

	if err == nil {
		err = os.Rename(tmpFile.Name(), path)
	}

	if err != nil {
		os.Remove(tmpFile.Name())
		return err
	}

	return nil
}

func LoadConfig(configDir string) (*config.Config, error) {
	cl := newConfigHandler(configDir)
	cfg, err := cl.Load()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(configDir string, cfg *config.Config) error {
	return newConfigHandler(configDir).Save(cfg)
}

// toString converts various primitive types to string.
func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case int, int64, float64, bool:
		return fmt.Sprintf("%v", t)
	default:
		return ""
	}
}

func LoadINI(rd io.Reader) (map[string]map[string]string, error) {
	cfg, err := ini.Load(rd)
	if err != nil {
		return nil, err
	}

	keysMap := make(map[string]struct{})
	result := make(map[string]map[string]string)
	for _, section := range cfg.Sections() {
		name := section.Name()
		if name == ini.DefaultSection {
			continue
		}
		keysMap[name] = struct{}{}
		result[name] = make(map[string]string)
		for _, key := range section.Keys() {
			result[name][key.Name()] = key.Value()
		}
	}
	return result, nil
}

func LoadYAML(rd io.Reader) (map[string]map[string]string, error) {
	var raw map[string]interface{}
	decoder := yaml.NewDecoder(rd)
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string)
	for section, value := range raw {
		sectionMap, ok := value.(map[string]interface{})
		if !ok {
			continue // skip non-object top-level keys
		}
		result[section] = make(map[string]string)
		for k, v := range sectionMap {
			result[section][k] = toString(v)
		}
	}

	return result, nil
}

// LoadJSON loads a JSON object and returns a nested map[string]map[string]string.
func LoadJSON(rd io.Reader) (map[string]map[string]string, error) {
	var raw map[string]map[string]string
	decoder := json.NewDecoder(rd)
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func GetConf(rd io.Reader, thirdParty string) (map[string]map[string]string, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, fmt.Errorf("failed to read config data: %w", err)
	}

	var configMap map[string]map[string]string
	if configMap, err = LoadYAML(bytes.NewReader(data)); err == nil {
	} else if configMap, err = LoadJSON(bytes.NewReader(data)); err == nil {
	} else if configMap, err = LoadINI(bytes.NewReader(data)); err != nil {
		return nil, fmt.Errorf("failed to parse config data: %w", err)
	}

	if thirdParty != "" {
		for _, section := range configMap {
			var ignore []string
			for key, value := range section {
				if slices.Contains(ignore, key) {
					continue
				}
				if value != "" {
					newKey := thirdParty + "_" + key
					section[newKey] = value
					ignore = append(ignore, newKey)
				}
				delete(section, key)
			}
			section["location"] = thirdParty + "://"
		}
	}

	seenLocation := false
	for _, section := range configMap {
		for key, value := range section {
			if key == "location" {
				seenLocation = true
			}
			if value == "" {
				delete(section, key)
			}
		}
	}
	if !seenLocation {
		return nil, fmt.Errorf("missing 'location' key in config data, `-rclone` import option missing ?")
	}
	return configMap, nil
}
