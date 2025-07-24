package plugins

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

type Manifest struct {
	Name        string   `yaml:"name"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	Homepage    string   `yaml:"homepage"`
	License     string   `yaml:"license"`
	Tags        []string `yaml:"tags"`
	APIVersion  string   `yaml:"api_version"`
	Version     string   `yaml:"version"`

	Connectors []struct {
		Type          string   `yaml:"type"`
		Protocols     []string `yaml:"protocols"`
		LocationFlags []string `yaml:"location_flags"`
		Executable    string   `yaml:"executable"`
		ExtraFiles    []string `yaml:"extra_files"`
	} `yaml:"connectors"`
}

func ParseManifestFile(path string, manifest *Manifest) error {
	fp, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("can't open the manifest: %w", err)
	}
	defer fp.Close()

	if err := yaml.NewDecoder(fp).Decode(manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	// We really want version to start with a 'v'
	if manifest.Version != "" && manifest.Version[0] != 'v' {
		manifest.Version = "v" + manifest.Version
	}

	return nil
}
