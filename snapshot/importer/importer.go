/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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

package importer

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"context"
	"io/fs"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/location"
	"github.com/PlakarKorp/plakar/objects"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"github.com/PlakarKorp/plakar/snapshot/importer/grpc/pkg"	
)

type ScanResult struct {
	Record *ScanRecord
	Error  *ScanError
}

type ExtendedAttributes struct {
	Name  string
	Value []byte
}

type ScanRecord struct {
	Pathname           string
	Target             string
	FileInfo           objects.FileInfo
	ExtendedAttributes []string
	FileAttributes     uint32
	IsXattr            bool
	XattrName          string
	XattrType          objects.Attribute
}

type ScanError struct {
	Pathname string
	Err      error
}

type Importer interface {
	Origin() string
	Type() string
	Root() string
	Scan() (<-chan *ScanResult, error)
	NewReader(string) (io.ReadCloser, error)
	NewExtendedAttributeReader(string, string) (io.ReadCloser, error)
	Close() error
}

type ImporterFn func(*appcontext.AppContext, string, map[string]string) (Importer, error)

var backends = location.New[ImporterFn]("fs")

func Register(name string, backend ImporterFn) {
	if !backends.Register(name, backend) {
		log.Fatalf("backend '%s' registered twice", name)
	}
}

func Backends() []string {
	return backends.Names()
}

type grpc_importer struct {
	grpcClient importer.ImporterClient
}

func (g *grpc_importer) Origin() string {
	info, err := g.grpcClient.Info(context.Background(), &importer.InfoRequest{})
	if err != nil {
		return ""
	}
	return info.GetOrigin()
}

func (g *grpc_importer) Type() string {
	info, err := g.grpcClient.Info(context.Background(), &importer.InfoRequest{})
	if err != nil {
		return ""
	}
	return info.GetType()
}

func (g *grpc_importer) Root() string {
	info, err := g.grpcClient.Info(context.Background(), &importer.InfoRequest{})
	if err != nil {
		return ""
	}
	return info.GetRoot()
}

func (g *grpc_importer) Scan() (<-chan *ScanResult, error) {
	stream, err := g.grpcClient.Scan(context.Background(), &importer.ScanRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to start scan: %w", err)
	}

	results := make(chan *ScanResult, 100)

	go func() {
		defer close(results)
		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				results <- NewScanError("", fmt.Errorf("failed to receive scan response: %w", err))
				return
			}
			if response.GetError() != nil {
				results <- NewScanError(response.GetPathname(), fmt.Errorf("scan error: %s", response.GetError().GetMessage()))
				continue
			}
			if response.GetRecord() == nil {
				results <- NewScanError(response.GetPathname(), fmt.Errorf("scan error: no record"))
				continue
			}
			isXattr := false
			if response.GetRecord().GetXattr() != nil {
				isXattr = true
			}
			results <- &ScanResult{
				Record: &ScanRecord{
					Pathname: response.GetPathname(),
					FileInfo: objects.FileInfo{
						Lname:		response.GetRecord().GetFileinfo().GetName(),
						Lsize:		response.GetRecord().GetFileinfo().GetSize(),
						Lmode:		fs.FileMode(response.GetRecord().GetFileinfo().GetMode()),
						LmodTime:	response.GetRecord().GetFileinfo().GetModTime().AsTime(),
						Ldev:		response.GetRecord().GetFileinfo().GetDev(),
						Lino:		response.GetRecord().GetFileinfo().GetIno(),
						Luid:		response.GetRecord().GetFileinfo().GetUid(),
						Lgid:		response.GetRecord().GetFileinfo().GetGid(),
						Lnlink:		uint16(response.GetRecord().GetFileinfo().GetNlink()),
						Lusername:	response.GetRecord().GetFileinfo().GetUsername(),
						Lgroupname:	response.GetRecord().GetFileinfo().GetGroupname(),
					},
					Target:             response.GetRecord().Target,	
					FileAttributes:     response.GetRecord().GetFileAttributes(),
					IsXattr:            isXattr,
					XattrName:          response.GetRecord().GetXattr().GetName(),
					XattrType:          objects.Attribute(response.GetRecord().GetXattr().GetType()),
				},
				Error: nil,
			}
		}
	}()
	return results, nil
}

// func (g *grpc_importer) NewReader(path string) (io.ReadCloser, error) {
// 	req := &importer.ReadRequest{
// 		Pathname: path,
// 	}
// 	resp, err := g.grpcClient.Read(context.Background(), req)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read file: %w", err)
// 	}
// 	if resp.GetError() != nil {
// 		return nil, fmt.Errorf("read error: %s", resp.GetError().GetMessage())
// 	}
// 	return io.NopCloser(strings.NewReader(resp.GetData())), nil	
// }

func (g *grpc_importer) NewExtendedAttributeReader(path string, xattr string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("not implemented")
}

func (g *grpc_importer) Close() error {
	if g.grpcClient != nil {
		if err := g.grpcClient.Close(); err != nil {
			return fmt.Errorf("failed to close grpc client: %w", err)
		}
	}
	return nil
}

func NewImporter(ctx *appcontext.AppContext, config map[string]string) (Importer, error) {
	location, ok := config["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}

	dirEntries, err := os.ReadDir(ctx.PluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}
	for _, entry := range dirEntries {
		if !entry.IsDir() {
			continue
		}
		if entry.Name() == ctx.PluginsVersion {
			pluginDir := filepath.Join(ctx.PluginsDir, entry.Name())
			PluginEntries, err := os.ReadDir(pluginDir)
			if err != nil {
				return nil, fmt.Errorf("failed to read plugin directory: %w", err)
			}
			for _, entry := range PluginEntries {
				if entry.IsDir() {
					continue
				}
				fmt.Println("name", entry.Name()[:strings.Index(entry.Name(), "-")])
				Register(entry.Name()[:strings.Index(entry.Name(), "-")], func (appCtx *appcontext.AppContext, name string, config map[string]string) (Importer, error) {
					serverAddr := "localhost:50052"
					conn, err := grpc.NewClient(serverAddr,
						grpc.WithTransportCredentials(insecure.NewCredentials()),
					)
					if err != nil {
						panic(err)
					}
					client := importer.NewImporterClient(conn)
					return &grpc_importer{
						grpcClient: client,
					}, nil
				})
			}
			break
		}
	}

	proto, location, backend, ok := backends.Lookup(location)
	if !ok {
		return nil, fmt.Errorf("unsupported importer protocol")
	}

	if proto == "fs" && !filepath.IsAbs(location) {
		location = filepath.Join(ctx.CWD, location)
		config["location"] = "fs://" + location
	} else {
		config["location"] = proto + "://" + location
	}
	return backend(ctx, proto, config)
}

func NewScanRecord(pathname, target string, fileinfo objects.FileInfo, xattr []string) *ScanResult {
	return &ScanResult{
		Record: &ScanRecord{
			Pathname:           pathname,
			Target:             target,
			FileInfo:           fileinfo,
			ExtendedAttributes: xattr,
		},
	}
}

func NewScanXattr(pathname, xattr string, kind objects.Attribute) *ScanResult {
	return &ScanResult{
		Record: &ScanRecord{
			Pathname:  pathname,
			IsXattr:   true,
			XattrName: xattr,
			XattrType: kind,
		},
	}
}

func NewScanError(pathname string, err error) *ScanResult {
	return &ScanResult{
		Error: &ScanError{
			Pathname: pathname,
			Err:      err,
		},
	}
}
