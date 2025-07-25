package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"gopkg.in/yaml.v3"
)

type InstalledPlugin struct {
	Name    string
	Version string
}

type InstalledPlugins []InstalledPlugin

func ListInstalledPlugins(ctx *appcontext.AppContext) (InstalledPlugins, error) {
	var plugins InstalledPlugins

	dataDir, err := utils.GetDataDir("plakar")
	if err != nil {
		return nil, fmt.Errorf("failed to get data directory: %w", err)
	}
	pluginsDir := filepath.Join(dataDir, "plugins")

	dirEntries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	for _, entry := range dirEntries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()

		if !strings.HasSuffix(name, ".ptar") {
			ctx.GetLogger().Warn("plugin name %q does not end with .ptar", name)
			continue
		}
		name = strings.TrimSuffix(name, ".ptar")
		atoms := strings.Split(name, "_")
		if len(atoms) != 4 {
			ctx.GetLogger().Warn("invalid plugin name %q", name)
			continue
		}
		if atoms[2] != runtime.GOOS {
			ctx.GetLogger().Warn("incorrect OS for plugin %q (expect %q)", name, runtime.GOOS)
			continue
		}
		if atoms[3] != runtime.GOARCH {
			ctx.GetLogger().Warn("incorrect architecture for plugin %q (expect %q)", name, runtime.GOARCH)
			continue
		}

		plugins = append(plugins, InstalledPlugin{
			Name:    atoms[0],
			Version: atoms[1],
		})

	}

	return plugins, nil
}

func (plugins InstalledPlugins) IsInstalled(name string) bool {
	return plugins.GetPlugin(name) != nil
}

func (plugins InstalledPlugins) GetPlugin(name string) *InstalledPlugin {
	for _, plugin := range plugins {
		if plugin.Name == name {
			return &plugin
		}
	}
	return nil
}

func (p InstalledPlugin) FullName() string {
	return fmt.Sprintf("%s_%s_%s_%s", p.Name, p.Version, runtime.GOOS, runtime.GOARCH)
}

func (p InstalledPlugin) PkgName() string {
	return fmt.Sprintf("%s.ptar", p.FullName())
}

func (p InstalledPlugin) Open(path string) (*os.File, error) {
	cachedir, err := utils.GetCacheDir("plakar")
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	path = filepath.Join(cachedir, "plugins", p.FullName(), path)
	return os.Open(path)
}

func (p InstalledPlugin) Manifest() (*Manifest, error) {
	fp, err := p.Open("manifest.yaml")
	if err != nil {
		return nil, fmt.Errorf("can't open the manifest: %w", err)
	}
	defer fp.Close()

	manifest := &Manifest{}
	if err := yaml.NewDecoder(fp).Decode(manifest); err != nil {
		return nil, fmt.Errorf("failed to decode the manifest: %w", err)
	}
	return manifest, nil
}
