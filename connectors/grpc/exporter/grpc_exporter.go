package grpc

import (
	"context"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	grpc_exporter "github.com/PlakarKorp/plakar/connectors/grpc/exporter/pkg"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type GrpcExporter struct {
	GrpcClient 	grpc_exporter.ExporterClient
	ctx 		context.Context
}

func (g *GrpcExporter) Close() error {
	_, err := g.GrpcClient.Close(g.ctx, &grpc_exporter.CloseRequest{})
	return err
}

func (g *GrpcExporter) CreateDirectory(pathname string) error {
	_, err := g.GrpcClient.CreateDirectory(g.ctx, &grpc_exporter.CreateDirectoryRequest{Pathname: pathname})
	if err != nil {
		return err
	}
	return nil
}

func (g *GrpcExporter) Root() string {
	info, err := g.GrpcClient.Root(g.ctx, &grpc_exporter.RootRequest{})
	if err != nil {
		return ""
	}
	return info.RootPath
}

func (g *GrpcExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	_, err := g.GrpcClient.SetPermissions(g.ctx, &grpc_exporter.SetPermissionsRequest{
		Pathname: pathname,
		FileInfo: &grpc_exporter.FileInfo{
			Name: 		fileinfo.Lname,
			Mode:	 	uint32(fileinfo.Lmode),
			ModTime: 	timestamppb.New(fileinfo.LmodTime),
			Dev: 		fileinfo.Ldev,
			Ino: 		fileinfo.Lino,
			Uid: 		fileinfo.Luid,
			Gid: 		fileinfo.Lgid,
			Nlink: 		uint32(fileinfo.Lnlink),
			Username: 	fileinfo.Lusername,
			Groupname: 	fileinfo.Lgroupname,
			Flags: 		fileinfo.Flags,
		},
	})
	return err
}

func (g *GrpcExporter) StoreFile(pathname string, fp io.Reader, size int64) error {
	stream, err := g.GrpcClient.StoreFile(g.ctx)
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
