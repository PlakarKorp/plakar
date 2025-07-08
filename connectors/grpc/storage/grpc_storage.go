/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package grpc

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/storage"
	"google.golang.org/grpc"

	grpc_storage "github.com/PlakarKorp/plakar/connectors/grpc/storage/pkg"
)

type GrpcStorage struct {
	GrpcClient grpc_storage.StoreClient
	Ctx        context.Context
}

const bufferSize = 16 * 1024

func NewStorage(ctx context.Context, client grpc.ClientConnInterface, proto string, config map[string]string) (storage.Store, error) {
	storage := &GrpcStorage{
		GrpcClient: grpc_storage.NewStoreClient(client),
		Ctx:        ctx,
	}

	_, err := storage.GrpcClient.Init(ctx, &grpc_storage.InitRequest{
		Proto:  proto,
		Config: config,
	})
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *GrpcStorage) Create(ctx context.Context, config []byte) error {
	_, err := s.GrpcClient.Create(ctx, &grpc_storage.CreateRequest{Config: config})
	if err != nil {
		return err
	}
	return nil
}

func (s *GrpcStorage) Open(ctx context.Context) ([]byte, error) {
	resp, err := s.GrpcClient.Open(ctx, &grpc_storage.OpenRequest{})
	if err != nil {
		return nil, err
	}
	return resp.Config, nil
}

func (s *GrpcStorage) Location() string {
	resp, err := s.GrpcClient.GetLocation(s.Ctx, &grpc_storage.GetLocationRequest{})
	if err != nil { // TODO: asdd error logging
		return ""
	}
	return resp.Location
}

func (s *GrpcStorage) Mode() storage.Mode {
	resp, err := s.GrpcClient.GetMode(s.Ctx, &grpc_storage.GetModeRequest{})
	if err != nil {
		return storage.Mode(0)
	}
	return storage.Mode(resp.Mode)
}

func (s *GrpcStorage) Size() int64 {
	resp, err := s.GrpcClient.GetSize(s.Ctx, &grpc_storage.GetSizeRequest{})
	if err != nil {
		return -1
	}
	return resp.Size
}

func SendChunks(rd io.Reader, chunkSendFn func(chunk []byte) error) (int64, error) {
	buffer := make([]byte, bufferSize)
	var totalBytes int64

	for {
		n, err := rd.Read(buffer)
		if n > 0 {
			if sendErr := chunkSendFn(buffer[:n]); sendErr != nil {
				return totalBytes, fmt.Errorf("failed to send chunk: %w", sendErr)
			}
			totalBytes += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return totalBytes, fmt.Errorf("failed to read: %w", err)
		}
	}
	return totalBytes, nil
}

type grpcChunkReader struct {
	streamRecv func() ([]byte, error)
	buf        bytes.Buffer
}

func (g *grpcChunkReader) Read(p []byte) (int, error) {
	totalRead := 0
	for totalRead < len(p) {
		//if there is data in the internal buffer -> read from it first
		if g.buf.Len() > 0 {
			n, err := g.buf.Read(p[totalRead:])
			totalRead += n
			if err != nil {
				return totalRead, err
			}
			//if the buffer is full -> done
			if totalRead == len(p) {
				return totalRead, nil
			}
		}

		//receive the next chunk of data
		chunk, err := g.streamRecv()
		if err != nil {
			if err == io.EOF {
				if totalRead > 0 {
					return totalRead, nil //return what we have before signaling EOF
				}
				return 0, io.EOF
			}
			return totalRead, fmt.Errorf("failed to receive file data: %w", err)
		}

		//add chunk to the internal buffer
		g.buf.Write(chunk)
	}

	return totalRead, nil
}

func ReceiveChunks(chunkReceiverFn func() ([]byte, error)) io.Reader {
	streamReader := &grpcChunkReader{
		streamRecv: chunkReceiverFn,
	}
	return streamReader
}

func toGrpcMAC(mac objects.MAC) *grpc_storage.MAC {
	return &grpc_storage.MAC{Value: mac[:]}
}

func (s *GrpcStorage) GetStates() ([]objects.MAC, error) {
	resp, err := s.GrpcClient.GetStates(s.Ctx, &grpc_storage.GetStatesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get states: %w", err)
	}

	var states []objects.MAC
	for _, mac := range resp.Macs {
		if len(mac.Value) != len(objects.MAC{}) {
			return nil, fmt.Errorf("invalid MAC length: %d", len(mac.Value))
		}
		states = append(states, objects.MAC(mac.Value))
	}
	return states, nil
}

func (s *GrpcStorage) PutState(mac objects.MAC, rd io.Reader) (int64, error) {
	stream, err := s.GrpcClient.PutState(s.Ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to start PutState stream: %w", err)
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_storage.PutStateRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send MAC: %w", err)
	}

	return SendChunks(rd, func(chunk []byte) error {
		return stream.Send(&grpc_storage.PutStateRequest{
			Chunk: chunk,
		})
	})
}

func (s *GrpcStorage) GetState(mac objects.MAC) (io.Reader, error) {
	stream, err := s.GrpcClient.GetState(s.Ctx, &grpc_storage.GetStateRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return nil, fmt.Errorf("get state: %w", err)
	}

	return ReceiveChunks(func() ([]byte, error) {
		resp, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return resp.Chunk, nil
	}), nil
}

func (s *GrpcStorage) DeleteState(mac objects.MAC) error {
	_, err := s.GrpcClient.DeleteState(s.Ctx, &grpc_storage.DeleteStateRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}
	return nil
}

func (s *GrpcStorage) GetPackfiles() ([]objects.MAC, error) {
	resp, err := s.GrpcClient.GetPackfiles(s.Ctx, &grpc_storage.GetPackfilesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get packfiles: %w", err)
	}

	var packfiles []objects.MAC
	for _, mac := range resp.Macs {
		if len(mac.Value) != len(objects.MAC{}) {
			return nil, fmt.Errorf("invalid MAC length: %d", len(mac.Value))
		}
		packfiles = append(packfiles, objects.MAC(mac.Value))
	}
	return packfiles, nil
}

func (s *GrpcStorage) PutPackfile(mac objects.MAC, rd io.Reader) (int64, error) {
	stream, err := s.GrpcClient.PutPackfile(s.Ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to start PutPackfile stream: %w", err)
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_storage.PutPackfileRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send MAC: %w", err)
	}

	return SendChunks(rd, func(chunk []byte) error {
		return stream.Send(&grpc_storage.PutPackfileRequest{
			Chunk: chunk,
		})
	})
}

func (s *GrpcStorage) GetPackfile(mac objects.MAC) (io.Reader, error) {
	stream, err := s.GrpcClient.GetPackfile(s.Ctx, &grpc_storage.GetPackfileRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get packfile: %w", err)
	}

	return ReceiveChunks(func() ([]byte, error) {
		resp, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return resp.Chunk, nil
	}), nil
}

func (s *GrpcStorage) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	stream, err := s.GrpcClient.GetPackfileBlob(s.Ctx, &grpc_storage.GetPackfileBlobRequest{
		Mac:    toGrpcMAC(mac),
		Offset: offset,
		Length: length,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get packfile blob: %w", err)
	}

	return ReceiveChunks(func() ([]byte, error) {
		resp, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return resp.Chunk, nil
	}), nil
}

func (s *GrpcStorage) DeletePackfile(mac objects.MAC) error {
	_, err := s.GrpcClient.DeletePackfile(s.Ctx, &grpc_storage.DeletePackfileRequest{
		Mac: toGrpcMAC(mac),
	})
	if err != nil {
		return fmt.Errorf("failed to delete packfile: %w", err)
	}
	return nil
}

func (s *GrpcStorage) GetLocks() ([]objects.MAC, error) {
	resp, err := s.GrpcClient.GetLocks(s.Ctx, &grpc_storage.GetLocksRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get locks: %w", err)
	}

	var locks []objects.MAC
	for _, mac := range resp.Macs {
		if len(mac.Value) != len(objects.MAC{}) {
			return nil, fmt.Errorf("invalid MAC length: %d", len(mac.Value))
		}
		locks = append(locks, objects.MAC(mac.Value))
	}
	return locks, nil
}

func (s *GrpcStorage) PutLock(lockID objects.MAC, rd io.Reader) (int64, error) {
	stream, err := s.GrpcClient.PutLock(s.Ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to start PutLock stream: %w", err)
	}
	defer stream.CloseSend()

	err = stream.Send(&grpc_storage.PutLockRequest{
		Mac: toGrpcMAC(lockID),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send MAC: %w", err)
	}

	return SendChunks(rd, func(chunk []byte) error {
		return stream.Send(&grpc_storage.PutLockRequest{
			Chunk: chunk,
		})
	})
}

func (s *GrpcStorage) GetLock(lockID objects.MAC) (io.Reader, error) {
	stream, err := s.GrpcClient.GetLock(s.Ctx, &grpc_storage.GetLockRequest{
		Mac: toGrpcMAC(lockID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get lock: %w", err)
	}

	return ReceiveChunks(func() ([]byte, error) {
		resp, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return resp.Chunk, nil
	}), nil
}

func (s *GrpcStorage) DeleteLock(lockID objects.MAC) error {
	_, err := s.GrpcClient.DeleteLock(s.Ctx, &grpc_storage.DeleteLockRequest{
		Mac: toGrpcMAC(lockID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete lock: %w", err)
	}
	return nil
}

func (s *GrpcStorage) Close() error {
	_, err := s.GrpcClient.Close(s.Ctx, &grpc_storage.CloseRequest{})
	if err != nil {
		return err
	}
	return nil
}
