package plugins

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	fsexporter "github.com/PlakarKorp/plakar/connectors/fs/exporter"
	grpc_exporter "github.com/PlakarKorp/plakar/connectors/grpc/exporter"
	grpc_importer "github.com/PlakarKorp/plakar/connectors/grpc/importer"
	grpc_storage "github.com/PlakarKorp/plakar/connectors/grpc/storage"
	"github.com/PlakarKorp/plakar/utils"
	"gopkg.in/yaml.v3"
)

type Manifest struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Connectors []struct {
		Type          string   `yaml:"type"`
		Protocols     []string `yaml:"protocols"`
		LocationFlags []string `yaml:"location_flags"`
		Executable    string   `yaml:"executable"`
		ExtraFiles    []string `yaml:"extra_files"`
		Homepage      string   `yaml:"homepage"`
		License       string   `yaml:"license"`
	} `yaml:"connectors"`
}

var re = regexp.MustCompile(`^([a-z0-9][a-zA-Z0-9\+.\-]*)-(v[0-9]+\.[0-9]+\.[0-9]+)\.ptar$`)

func ValidateName(name string) bool {
	return re.MatchString(name)
}

func ListDir(ctx *appcontext.AppContext, pluginsDir string) ([]string, error) {
	var names []string

	dirEntries, err := os.ReadDir(pluginsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return names, nil
		}
		return names, err
	}

	for _, entry := range dirEntries {
		if !entry.Type().IsRegular() {
			continue
		}

		if !re.MatchString(entry.Name()) {
			continue
		}

		names = append(names, entry.Name())
	}
	return names, nil
}

func LoadDir(ctx *appcontext.AppContext, pluginsDir, cacheDir string) error {
	names, err := ListDir(ctx, pluginsDir)
	if err != nil {
		return err
	}

	for _, name := range names {
		if err := Load(ctx, pluginsDir, cacheDir, name); err != nil {
			return err
		}
	}

	return nil
}

func Load(ctx *appcontext.AppContext, pluginsDir, cacheDir, name string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	extlen := len(filepath.Ext(name))
	plugin := filepath.Join(cacheDir, name[:len(name)-extlen])

	if _, err := os.Stat(plugin); err != nil {
		path := filepath.Join(pluginsDir, name)
		if err := extract(ctx, path, plugin); err != nil {
			return err
		}
	}

	fp, err := os.Open(filepath.Join(plugin, "manifest.yaml"))
	if err != nil {
		return fmt.Errorf("can't open the manifest: %w", err)
	}
	defer fp.Close()

	manifest := Manifest{}
	if err := yaml.NewDecoder(fp).Decode(&manifest); err != nil {
		return fmt.Errorf("failed to decode the manifest: %w", err)
	}

	for _, conn := range manifest.Connectors {
		exe := filepath.Join(plugin, conn.Executable)
		if !strings.HasPrefix(exe, plugin) {
			return fmt.Errorf("bad executable path %q in plugin %s", conn.Executable, name)
		}

		var flags location.Flags
		for _, flag := range conn.LocationFlags {
			f, err := location.ParseFlag(flag)
			if err != nil {
				return fmt.Errorf("unknown flag %q in plugin %s", flag, name)
			}
			flags |= f
		}

		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "importer":
				importer.Register(proto, flags, func(ctx context.Context, o *importer.Options, s string, config map[string]string) (importer.Importer, error) {
					client, err := connectPlugin(exe)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return grpc_importer.NewImporter(ctx, client, o, s, config)
				})
			case "exporter":
				exporter.Register(proto, flags, func(ctx context.Context, o *exporter.Options, s string, config map[string]string) (exporter.Exporter, error) {
					client, err := connectPlugin(exe)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return grpc_exporter.NewExporter(ctx, client, o, s, config)
				})
			case "storage":
				storage.Register(proto, flags, func(ctx context.Context, s string, config map[string]string) (storage.Store, error) {
					client, err := connectPlugin(exe)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return grpc_storage.NewStorage(ctx, client, s, config)
				})
			default:
				return fmt.Errorf("unknown plugin type: %s", conn.Type)
			}
		}
	}

	return nil
}

func extract(ctx *appcontext.AppContext, plugin, destDir string) error {
	opts := map[string]string{
		"location": "ptar://" + plugin,
	}

	store, serializedConfig, err := storage.Open(ctx.GetInner(), opts)
	if err != nil {
		return err
	}

	repo, err := repository.New(ctx.GetInner(), nil, store, serializedConfig)
	if err != nil {
		return err
	}

	locopts := utils.NewDefaultLocateOptions()
	snapids, err := utils.LocateSnapshotIDs(repo, locopts)
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
