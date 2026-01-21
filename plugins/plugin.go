package plugins

import (
	"context"
	"fmt"
	"path/filepath"

	gexporter "github.com/PlakarKorp/integration-grpc/exporter"
	gimporter "github.com/PlakarKorp/integration-grpc/importer"
	gstorage "github.com/PlakarKorp/integration-grpc/storage"
	gexporterv2 "github.com/PlakarKorp/integration-grpc/v2/exporter"
	gimporterv2 "github.com/PlakarKorp/integration-grpc/v2/importer"
	gstoragev2 "github.com/PlakarKorp/integration-grpc/v2/storage"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/pkg"
	"golang.org/x/mod/semver"
)

func RegisterStorage(v2 bool, proto string, flags location.Flags, exe string, args []string) error {
	err := storage.Register(proto, flags, func(ctx context.Context, s string, config map[string]string) (storage.Store, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		if v2 {
			return gstoragev2.NewStorage(ctx, client, s, config)
		}
		return gstorage.NewStorage(ctx, client, s, config)
	})
	if err != nil {
		return err

	}
	return nil
}

func RegisterImporter(v2 bool, proto string, flags location.Flags, exe string, args []string) error {
	err := importer.Register(proto, flags, func(ctx context.Context, o *connectors.Options, s string, config map[string]string) (importer.Importer, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}
		if v2 {
			return gimporterv2.NewImporter(ctx, client, o, s, config)
		}
		return gimporter.NewImporter(ctx, client, o, s, config)
	})
	if err != nil {
		return err
	}
	return nil
}

func RegisterExporter(v2 bool, proto string, flags location.Flags, exe string, args []string) error {
	err := exporter.Register(proto, flags, func(ctx context.Context, o *connectors.Options, s string, config map[string]string) (exporter.Exporter, error) {
		client, err := connectPlugin(ctx, exe, args)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to plugin: %w", err)
		}

		if v2 {
			return gexporterv2.NewExporter(ctx, client, o, s, config)
		}
		return gexporter.NewExporter(ctx, client, o, s, config)
	})
	if err != nil {
		return err
	}
	return nil
}

func Load(m *pkg.Manifest, pkgdir string) error {
	var v2 bool

	if m.APIVersion != "" {
		switch semver.Major(m.APIVersion) {
		case "v1":
			// nothing
		case "v2":
			v2 = true
		default:
			return fmt.Errorf("unknown api version %s", m.APIVersion)
		}
	}

	for _, conn := range m.Connectors {
		exe := filepath.Join(pkgdir, conn.Executable)

		flags, err := conn.Flags()
		if err != nil {
			return err
		}

		for _, proto := range conn.Protocols {
			switch conn.Type {
			case "importer":
				err = RegisterImporter(v2, proto, flags, exe, conn.Args)
			case "exporter":
				err = RegisterExporter(v2, proto, flags, exe, conn.Args)
			case "storage":
				err = RegisterStorage(v2, proto, flags, exe, conn.Args)
			default:
				err = fmt.Errorf("unknown connector type: %s",
					conn.Type)
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
				err = fmt.Errorf("unknown connector type: %s",
					conn.Type)
			}
		}
	}
	return err
}
