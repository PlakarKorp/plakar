package plugins

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	fsexporter "github.com/PlakarKorp/integration-fs/exporter"
	grpc_exporter "github.com/PlakarKorp/integration-grpc/exporter"
	grpc_importer "github.com/PlakarKorp/integration-grpc/importer"
	grpc_storage "github.com/PlakarKorp/integration-grpc/storage"
	"github.com/PlakarKorp/kloset/kcontext"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/kloset/storage"
)

type TearDownFunc func() error

type Plugin struct {
	teardown []TearDownFunc
}

func (plugin *Plugin) SetUp(ctx *kcontext.KContext, pluginFile, pluginName, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	pluginPath := filepath.Join(cacheDir, pluginName)
	if _, err := os.Stat(pluginPath); err != nil {
		if err := extractPlugin(ctx, pluginFile, pluginPath); err != nil {
			return err
		}
	}

	manifestFile := filepath.Join(pluginPath, "manifest.yaml")
	manifest := Manifest{}
	if err := ParseManifestFile(manifestFile, &manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	for _, conn := range manifest.Connectors {
		exe := filepath.Join(pluginPath, conn.Executable)
		if !strings.HasPrefix(exe, pluginPath) {
			return fmt.Errorf("bad executable path %q in plugin %s", conn.Executable, pluginName)
		}

		var flags location.Flags
		for _, flag := range conn.LocationFlags {
			f, err := location.ParseFlag(flag)
			if err != nil {
				return fmt.Errorf("unknown flag %q in plugin %s", flag, pluginName)
			}
			flags |= f
		}

		var err error
		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "importer":
				err = plugin.registerImporter(proto, flags, exe, conn.Args)
			case "exporter":
				err = plugin.registerExporter(proto, flags, exe, conn.Args)
			case "storage":
				err = plugin.registerStorage(proto, flags, exe, conn.Args)
			default:
				err = fmt.Errorf("unknown plugin type: %s", conn.Type)
			}
			if err != nil {
				plugin.TearDown(ctx)
				return err
			}
		}
	}

	return nil
}

func (plugin *Plugin) TearDown(ctx *kcontext.KContext) {
	for _, fn := range plugin.teardown {
		err := fn()
		if err != nil {
			ctx.GetLogger().Warn("%v", err)
		}
	}
	plugin.teardown = nil
}

func (plugin *Plugin) registerStorage(proto string, flags location.Flags, exe string, args []string) error {
	err := storage.Register(proto, flags, func(ctx context.Context, s string, config map[string]string) (storage.Store, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		return grpc_storage.NewStorage(ctx, client, s, config)
	})
	if err != nil {
		return err

	}
	plugin.teardown = append(plugin.teardown, func() error { return storage.Unregister(proto) })
	return nil
}

func (plugin *Plugin) registerImporter(proto string, flags location.Flags, exe string, args []string) error {
	err := importer.Register(proto, flags, func(ctx context.Context, o *importer.Options, s string, config map[string]string) (importer.Importer, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}
		return grpc_importer.NewImporter(ctx, client, o, s, config)
	})
	if err != nil {
		return err
	}
	plugin.teardown = append(plugin.teardown, func() error { return importer.Unregister(proto) })
	return nil
}

func (plugin *Plugin) registerExporter(proto string, flags location.Flags, exe string, args []string) error {
	err := exporter.Register(proto, flags, func(ctx context.Context, o *exporter.Options, s string, config map[string]string) (exporter.Exporter, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		return grpc_exporter.NewExporter(ctx, client, o, s, config)
	})
	if err != nil {
		return err
	}
	plugin.teardown = append(plugin.teardown, func() error { return exporter.Unregister(proto) })
	return nil
}

func extractPlugin(ctx *kcontext.KContext, pluginFile, destDir string) error {
	opts := map[string]string{
		"location": "ptar://" + pluginFile,
	}

	store, serializedConfig, err := storage.Open(ctx, opts)
	if err != nil {
		return err
	}

	repo, err := repository.New(ctx, nil, store, serializedConfig)
	if err != nil {
		return err
	}

	locopts := locate.NewDefaultLocateOptions()
	snapids, err := locate.LocateSnapshotIDs(repo, locopts)
	if len(snapids) != 1 {
		return fmt.Errorf("too many snapshot in ptar plugin: %d",
			len(snapids))
	}

	snapid := snapids[0]
	snap, err := snapshot.Load(repo, snapid)
	if err != nil {
		return err
	}

	fsexp, err := fsexporter.NewFSExporter(ctx, &exporter.Options{
		MaxConcurrency: 1,
	}, "fs", opts)
	if err != nil {
		return err
	}

	tmpdir, err := os.MkdirTemp(filepath.Dir(destDir), "plugin-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	base := snap.Header.GetSource(0).Importer.Directory
	err = snap.Restore(fsexp, tmpdir, base, &snapshot.RestoreOptions{
		MaxConcurrency: 1,
		Strip:          base,
	})
	if err != nil {
		return err
	}

	if err := os.Rename(tmpdir, destDir); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	return nil
}

func installPlugin(filename, pluginFile string) error {

	if err := os.MkdirAll(path.Dir(pluginFile), 0755); err != nil {
		return fmt.Errorf("failed to create plugin dir: %w", err)
	}

	if err := os.Link(filename, pluginFile); err == nil {
		// load
		return nil
	}

	fp, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fp.Close()

	// maybe a different filesystem
	tmp, err := os.CreateTemp(path.Dir(pluginFile), "pkg-add-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, fp); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), pluginFile)
}
