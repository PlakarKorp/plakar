package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"gopkg.in/ini.v1"

	"go.yaml.in/yaml/v3"
)

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
