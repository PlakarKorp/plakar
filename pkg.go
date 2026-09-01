package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/signify"
	"github.com/PlakarKorp/plakar/utils"
)

func setupPkgManager(ctx *appcontext.AppContext, configDir, dataDir, cacheDir string) error {
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

	// The trust store is built once per run from the keys compiled into
	// this binary plus any the user added. A failure to read it is fatal
	// rather than degraded: continuing with fewer keys than the user
	// configured would silently install packages they never trusted.
	trust, err := signify.LoadTrustStore(configDir)
	if err != nil {
		return fmt.Errorf("failed to load the package trust store: %w", err)
	}

	verifier := signify.NewVerifier(trust)

	token, _ := ctx.GetCookies().GetAuthToken()
	manager, err := pkg.New(backend, &pkg.Options{
		DistURL:         "https://plakar.io/dist/plugins/kloset/",
		Edition:         "community",
		ApiURL:          "https://api.plakar.io/",
		BinaryNeedsAuth: true,
		UserAgent:       "plakar/" + utils.VERSION,
		RequestHook:     pkg.WithBearer(func() (string, error) { return token, nil }),
		Verifier:        verifier,
	})
	if err != nil {
		return fmt.Errorf("failed to init the package manager: %w", err)
	}

	if err := backend.LoadAll(); err != nil {
		return fmt.Errorf("failed to load packages: %w", err)
	}

	ctx.SetPkgManager(manager)
	ctx.SetPkgVerifier(verifier)

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
			case "storage":
				if slices.Contains(storage.Backends(), proto) {
					return fmt.Errorf("protocol %s already provided "+
						"by another installed package", proto)
				}
			default:
				fmt.Fprintf(os.Stderr,
					"skipping unknown connector type: %s\n",
					conn.Type)
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
