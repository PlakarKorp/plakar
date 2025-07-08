package grpc

import (
	"context"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	grpc_exporter "github.com/PlakarKorp/plakar/connectors/grpc/exporter/pkg"
	"google.golang.org/grpc"

	// google being google I guess.  No idea why this is actually
	// required, but otherwise it breaks the workspace setup
	// c.f. https://github.com/googleapis/go-genproto/issues/1015
	_ "google.golang.org/genproto/protobuf/ptype"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcExporter struct {
	GrpcClient grpc_exporter.ExporterClient
	Ctx        context.Context
}

func NewExporter(ctx context.Context, client grpc.ClientConnInterface, opts *exporter.Options, proto string, config map[string]string) (exporter.Exporter, error) {
	exporter := &GrpcExporter{
		GrpcClient: grpc_exporter.NewExporterClient(client),
		Ctx:        ctx,
	}

	_, err := exporter.GrpcClient.Init(ctx, &grpc_exporter.InitRequest{
		Options: &grpc_exporter.Options{
			Maxconcurrency: int64(opts.MaxConcurrency),
		},
		Proto:  proto,
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	return exporter, nil
}

func (g *GrpcExporter) Close() error {
	_, err := g.GrpcClient.Close(g.Ctx, &grpc_exporter.CloseRequest{})
	return err
}

func (g *GrpcExporter) CreateDirectory(pathname string) error {
	_, err := g.GrpcClient.CreateDirectory(g.Ctx, &grpc_exporter.CreateDirectoryRequest{Pathname: pathname})
	if err != nil {
		return err
	}
	return nil
}

func (g *GrpcExporter) Root() string {
	info, err := g.GrpcClient.Root(g.Ctx, &grpc_exporter.RootRequest{})
	if err != nil {
		return ""
	}
	return info.RootPath
}

func (g *GrpcExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	_, err := g.GrpcClient.SetPermissions(g.Ctx, &grpc_exporter.SetPermissionsRequest{
		Pathname: pathname,
		FileInfo: &grpc_exporter.FileInfo{
			Name:      fileinfo.Lname,
			Mode:      uint32(fileinfo.Lmode),
			ModTime:   timestamppb.New(fileinfo.LmodTime),
			Dev:       fileinfo.Ldev,
			Ino:       fileinfo.Lino,
			Uid:       fileinfo.Luid,
			Gid:       fileinfo.Lgid,
			Nlink:     uint32(fileinfo.Lnlink),
			Username:  fileinfo.Lusername,
			Groupname: fileinfo.Lgroupname,
			Flags:     fileinfo.Flags,
		},
	})
	return err
}

func (g *GrpcExporter) StoreFile(pathname string, fp io.Reader, size int64) error {
	stream, err := g.GrpcClient.StoreFile(g.Ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(&grpc_exporter.StoreFileRequest{
		Type: &grpc_exporter.StoreFileRequest_Header{
			Header: &grpc_exporter.Header{
				Pathname: pathname,
				Size:     uint64(size),
			},
		},
	}); err != nil {
		return err
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := fp.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if err := stream.Send(&grpc_exporter.StoreFileRequest{
			Type: &grpc_exporter.StoreFileRequest_Data{
				Data: &grpc_exporter.Data{
					Chunk: buf[:n],
				},
			},
		}); err != nil {
			return err
		}
	}

	_, err = stream.CloseAndRecv()
	return err
}
