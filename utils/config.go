package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"

	"github.com/PlakarKorp/kloset/config"
	"go.yaml.in/yaml/v3"
)

type configHandler struct {
	Path string
}

func newConfigHandler(path string) *configHandler {
	return &configHandler{
		Path: path,
	}
}

func (cl *configHandler) Load() (*config.Config, error) {
	cfg := config.NewConfig()
	err := cl.load("sources.yml", &cfg.Sources)
	if err != nil {
		if os.IsNotExist(err) {
			goto fallback
		}
		return nil, err
	}
	err = cl.load("destinations.yml", &cfg.Destinations)
	if err != nil {
		if os.IsNotExist(err) {
			goto fallback
		}
		return nil, err
	}
	err = cl.load("klosets.yml", &cfg.Repositories)
	if err != nil {
		if os.IsNotExist(err) {
			goto fallback
		}
		return nil, err
	}

	for k, v := range cfg.Repositories {
		if _, ok := v[".isDefault"]; ok {
			if cfg.DefaultRepository != "" {
				return nil, fmt.Errorf("multiple default store")
			}
			cfg.DefaultRepository = k
			delete(v, ".isDefault")
		}
	}

	return cfg, nil

fallback:
	// Load old config if found
	oldpath := filepath.Join(cl.Path, "plakar.yml")
	cfg, err = LoadOldConfigIfExists(oldpath)
	if err != nil {
		return nil, fmt.Errorf("error reading old config file: %w", err)
	}

	// Save the config in the new format right now
	err = SaveConfig(cl.Path, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to update old config file: %w", err)
	}
	// Do we want to remove the old file?
	return cfg, nil
}

func (cl *configHandler) Save(cfg *config.Config) error {
	err := cl.save("sources.yml", cfg.Sources)
	if err != nil {
		return err
	}
	err = cl.save("destinations.yml", cfg.Destinations)
	if err != nil {
		return err
	}
	for k, v := range cfg.Repositories {
		if k == cfg.DefaultRepository {
			v[".isDefault"] = "yes"
		}
	}
	err = cl.save("klosets.yml", cfg.Repositories)
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
		return fmt.Errorf("error reading config file: %w", err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to get config file info: %w", err)
	}
	if info.Size() == 0 {
		return nil
	}

	err = yaml.NewDecoder(f).Decode(dst)
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

func GetConf(rd io.Reader) (map[string]map[string]string, error) {
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

	for _, section := range configMap {
		if section["type"] != "" {
			section["location"] = "rclone://"
		}
	}
	for _, section := range configMap {
		for key, value := range section {
			if value == "" {
				delete(section, key)
			}
		}
	}
	return configMap, nil
}
