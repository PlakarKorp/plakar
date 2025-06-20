//go:build !windows

package plugins

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	"github.com/PlakarKorp/kloset/snapshot/importer"
	grpc_importer "github.com/PlakarKorp/plakar/connectors/grpc/importer"
	grpc_importer_pkg "github.com/PlakarKorp/plakar/connectors/grpc/importer/pkg"

	"github.com/PlakarKorp/kloset/snapshot/exporter"
	grpc_exporter "github.com/PlakarKorp/plakar/connectors/grpc/exporter"
	grpc_exporter_pkg "github.com/PlakarKorp/plakar/connectors/grpc/exporter/pkg"

	"github.com/PlakarKorp/kloset/storage"
	grpc_storage "github.com/PlakarKorp/plakar/connectors/grpc/storage"
	grpc_storage_pkg "github.com/PlakarKorp/plakar/connectors/grpc/storage/pkg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Load(ctx context.Context, pluginPath string) error {
	dirEntries, err := os.ReadDir(pluginPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	pluginAcceptTypes := map[string]bool{"importer": true, "exporter": true, "storage": true}
	re := regexp.MustCompile(`^([a-z0-9][a-zA-Z0-9\+.\-]*)-(v[0-9]+\.[0-9]+\.[0-9]+)\.ptar$`)

	for _, pluginEntry := range dirEntries {
		if !pluginEntry.IsDir() || !pluginAcceptTypes[pluginEntry.Name()] {
			continue
		}
		pluginFolderPath := filepath.Join(pluginPath, pluginEntry.Name())
		pluginFiles, err := os.ReadDir(pluginFolderPath)
		if err != nil {
			return fmt.Errorf("failed to read plugin folder %s: %w", pluginEntry.Name(), err)
		}

		for _, entry := range pluginFiles {
			matches := re.FindStringSubmatch(entry.Name())
			if matches == nil {
				continue
			}
			key := matches[1]
			pluginFileName := matches[0]

			switch pluginEntry.Name() {
			case "importer":
				importer.Register(key, 0, func(ctx context.Context, o *importer.Options, s string, config map[string]string) (importer.Importer, error) {
					client, err := connectPlugin(filepath.Join(pluginFolderPath, pluginFileName), config)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return &grpc_importer.GrpcImporter{
						GrpcClientScan:   grpc_importer_pkg.NewImporterClient(client),
						GrpcClientReader: grpc_importer_pkg.NewImporterClient(client),
						Ctx:              ctx,
					}, nil
				})
			case "exporter":
				exporter.Register(key, 0, func(ctx context.Context, o *exporter.Options, s string, config map[string]string) (exporter.Exporter, error) {
					client, err := connectPlugin(filepath.Join(pluginFolderPath, pluginFileName), config)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return &grpc_exporter.GrpcExporter{
						GrpcClient: grpc_exporter_pkg.NewExporterClient(client),
						Ctx:        ctx,
					}, nil
				})
			case "storage":
				storage.Register(func(ctx context.Context, s string, config map[string]string) (storage.Store, error) {
					client, err := connectPlugin(filepath.Join(pluginFolderPath, pluginFileName), config)
					if err != nil {
						return nil, fmt.Errorf("failed to connect to plugin: %w", err)
					}

					return &grpc_storage.GrpcStorage{
						GrpcClient: grpc_storage_pkg.NewStoreClient(client),
						Ctx:        ctx,
					}, nil
				}, 0, key)
			default:
				return fmt.Errorf("unknown plugin type: %s", pluginEntry.Name())
			}
		}
	}
	return nil
}

func connectPlugin(pluginPath string, config map[string]string) (grpc.ClientConnInterface, error) {
	fd, err := forkChild(pluginPath, config)
	if err != nil {
		return nil, err
	}

	connFile := os.NewFile(uintptr(fd), "grpc-conn")
	conn, err := net.FileConn(connFile)
	if err != nil {
		return nil, fmt.Errorf("net.FileConn failed: %w", err)
	}

	clientConn, err := grpc.NewClient("127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return conn, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc client creation failed: %w", err)
	}
	return clientConn, nil
}

func forkChild(pluginPath string, config map[string]string) (int, error) {
	sp, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return -1, fmt.Errorf("failed to create socketpair: %w", err)
	}

	childFile := os.NewFile(uintptr(sp[0]), "child-conn")

	cmd := exec.Command(pluginPath, fmt.Sprintf("%v", config))
	cmd.Stdin = childFile
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("failed to start plugin: %w", err)
	}

	childFile.Close()
	return sp[1], nil
}
