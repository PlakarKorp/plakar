// Package grpcconn is a wrapper for integration-sdk/client
// It adds the graceful termination handling that is currently
// not in the integration-sdk lib.
package grpcconn

import (
	"context"
	"errors"
	"time"

	"github.com/PlakarKorp/integration-sdk/client"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"google.golang.org/grpc"
)

// closeTimeout is a var for tests.
var closeTimeout = 30 * time.Second

func NewStorage(ctx context.Context, conn *grpc.ClientConn, proto string, config map[string]string) (storage.Store, error) {
	s, err := client.NewStorage(ctx, conn, proto, config)
	if err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return &store{Store: s, conn: conn}, nil
}

func NewImporter(ctx context.Context, conn *grpc.ClientConn, opts *connectors.Options, proto string, config map[string]string) (importer.Importer, error) {
	imp, err := client.NewImporter(ctx, conn, opts, proto, config)
	if err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return &importerConn{Importer: imp, conn: conn}, nil
}

func NewExporter(ctx context.Context, conn *grpc.ClientConn, opts *connectors.Options, proto string, config map[string]string) (exporter.Exporter, error) {
	exp, err := client.NewExporter(ctx, conn, opts, proto, config)
	if err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return &exporterConn{Exporter: exp, conn: conn}, nil
}

type store struct {
	storage.Store
	conn *grpc.ClientConn
}

func (s *store) Close(ctx context.Context) error {
	return closeConn(ctx, s.Store.Close, s.conn)
}

type importerConn struct {
	importer.Importer
	conn *grpc.ClientConn
}

func (i *importerConn) Close(ctx context.Context) error {
	return closeConn(ctx, i.Importer.Close, i.conn)
}

type exporterConn struct {
	exporter.Exporter
	conn *grpc.ClientConn
}

func (e *exporterConn) Close(ctx context.Context) error {
	return closeConn(ctx, e.Exporter.Close, e.conn)
}

// closeConn overrides the context cancel with a custom 30 seconds timeout.
// if the context is already canceled, it leaves time for the plugin to cleanup.
func closeConn(ctx context.Context, rpcClose func(context.Context) error, conn *grpc.ClientConn) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), closeTimeout)
	defer cancel()
	return errors.Join(rpcClose(ctx), conn.Close())
}
