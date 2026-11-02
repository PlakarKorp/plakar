package plugins

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/PlakarKorp/integration-sdk/client"
	"github.com/PlakarKorp/integration-sdk/location"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/pkg"
)

func RegisterStorage(proto string, flags location.Flags, runner Runner) error {
	err := storage.Register(proto, flags, func(ctx context.Context, s string, config map[string]string) (storage.Store, error) {
		gclient, err := connectPlugin(ctx, runner)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		return client.NewStorage(ctx, gclient, s, config)
	})
	if err != nil {
		return err

	}
	return nil
}

func RegisterImporter(proto string, flags location.Flags, runner Runner) error {
	err := importer.Register(proto, flags, func(ctx context.Context, o *connectors.Options, s string, config map[string]string) (importer.Importer, error) {
		gclient, err := connectPlugin(ctx, runner)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}
		return client.NewImporter(ctx, gclient, o, s, config)
	})
	if err != nil {
		return err
	}
	return nil
}

func RegisterExporter(proto string, flags location.Flags, runner Runner) error {
	err := exporter.Register(proto, flags, func(ctx context.Context, o *connectors.Options, s string, config map[string]string) (exporter.Exporter, error) {
		gclient, err := connectPlugin(ctx, runner)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		return client.NewExporter(ctx, gclient, o, s, config)
	})
	if err != nil {
		return err
	}
	return nil
}

// Load registers the connectors of a package. Native plugins get stderr as
// their stderr, so a renderer that owns the terminal can keep their logs
// off it.
func Load(m *pkg.Manifest, pkgdir string, stderr io.Writer) error {
	for _, conn := range m.Connectors {
		if err := conn.Validate(); err != nil {
			return err
		}

		var runner Runner
		if conn.ImageID != "" {
			runner = &DockerRunner{
				Image: conn.ImageID,
				Args:  conn.Args,
			}
		} else {
			runner = &NativeRunner{
				Path:   filepath.Join(pkgdir, conn.Executable),
				Args:   conn.Args,
				Stderr: stderr,
			}
		}

		flags, err := conn.Flags()
		if err != nil {
			return err
		}

		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "importer":
				err = RegisterImporter(proto, flags, runner)
			case "exporter":
				err = RegisterExporter(proto, flags, runner)
			case "storage":
				err = RegisterStorage(proto, flags, runner)
			default:
				/* Ignore silently. */
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func Unload(m *pkg.Manifest) error {
	var err error
	for _, conn := range m.Connectors {
		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "importer":
				err = importer.Unregister(proto)
			case "exporter":
				err = exporter.Unregister(proto)
			case "storage":
				err = storage.Unregister(proto)
			default:
				/* ignore silently */
			}
		}
	}
	return err
}
