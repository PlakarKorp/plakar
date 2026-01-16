package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/utils"
)

func setupPkgManager(ctx *appcontext.AppContext, dataDir, cacheDir string) error {
	plugdir := filepath.Join(dataDir, "plugins", pkg.PLUGIN_API_VERSION)
	cachedir := filepath.Join(cacheDir, "plugins", pkg.PLUGIN_API_VERSION)

	backend, err := pkg.NewFlatBackend(ctx.GetInner(), plugdir, cachedir, &pkg.FlatBackendOptions{
		PreLoadHook: pkgpreloadhook,
		LoadHook:    pkgloadhook,
		UnloadHook:  pkgunloadhook,
	})
	if err != nil {
		return fmt.Errorf("failed to init the package manager: %w", err)
	}

	token, _ := ctx.GetCookies().GetAuthToken()
	manager, err := pkg.New(backend, &pkg.Options{
		InstallURL:      "https://plugins.plakar.io/",
		ApiURL:          "https://api.plakar.io/",
		BinaryNeedsAuth: true,
		UserAgent:       "plakar/" + utils.VERSION,
		RequestHook:     pkg.WithBearer(func() (string, error) { return token, nil }),
	})
	if err != nil {
		return fmt.Errorf("failed to init the package manager: %w", err)
	}

	if err := backend.LoadAll(); err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	ctx.SetPkgManager(manager)

	return nil
}

func pkgpreloadhook(m *pkg.Manifest) error {
	for _, conn := range m.Connectors {
		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "exporter":
				if slices.Contains(exporter.Backends(), proto) {
					return fmt.Errorf("protocol %s already provided "+
						"by another installed package", proto)
				}
			case "importer":
				if slices.Contains(importer.Backends(), proto) {
					return fmt.Errorf("protocol %s already provided "+
						"by another installed package", proto)
				}
			case "store":
				if slices.Contains(storage.Backends(), proto) {
					return fmt.Errorf("protocol %s already provided "+
						"by another installed package", proto)
				}
			}
		}
	}

	return nil
}

func pkgloadhook(m *pkg.Manifest, p *pkg.Package, pkgdir string) {
	if err := plugins.Load(m, pkgdir); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load %s@%s: %s\n",
			flag.CommandLine.Name(), m.Name, p.Version, err)
	}
}

func pkgunloadhook(m *pkg.Manifest, p *pkg.Package) {
	if err := plugins.Unload(m); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to unload %s@%s: %s\n",
			flag.CommandLine.Name(), m.Name, p.Version, err)
	}
}
