package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PlakarKorp/go-kloset-sdk/sdk"
	"github.com/PlakarKorp/plakar/objects"
	impor "github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/pkg/xattr"
)

type PlakarImporterFS struct {
	rootDir string
}

func NewPlakarImporterFS(location string) (*PlakarImporterFS, error) {
	var err error

	if strings.HasPrefix(location, "fs://") {
		location = location[4:]
	}

	location, err = filepath.Abs(location)
	if err != nil {
		return nil, err
	}

	return &PlakarImporterFS{
		rootDir: location,
	}, nil
}

func (imp *PlakarImporterFS) Info(ctx context.Context, req *sdk.InfoRequest) (*sdk.InfoResponse, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return &sdk.InfoResponse{
		Type:   "fs",
		Origin: hostname,
		Root:   imp.rootDir,
	}, nil
}

func (imp *PlakarImporterFS) Scan(req *sdk.ScanRequest, stream sdk.ScanResponseStreamer) error {
	realp, err := realpathFollow(imp.rootDir)
	if err != nil {
		return err
	}

	results := make(chan *impor.ScanResult, 1000)
	go imp.walkDir_walker(results, imp.rootDir, realp, 256)

	for result := range results {
		switch {
		case result.Record != nil:
			if err := stream.Context().Err(); err != nil {
				fmt.Printf("Client connection closed: %v\n", err)
				return err
			}

			var xattr *sdk.ExtendedAttribute
			if result.Record.IsXattr {
				xattr = &sdk.ExtendedAttribute{
					Name: result.Record.XattrName,
					Type: sdk.ExtendedAttributeType(result.Record.XattrType),
				}
			} else {
				xattr = nil
			}

			if err := stream.Send(&sdk.ScanResponse{
				Pathname: result.Record.Pathname,
				Result: &sdk.ScanResponseRecord{
					Record: &sdk.ScanRecord{
						Target: result.Record.Target,
						Fileinfo: &sdk.ScanRecordFileInfo{
							Name:      result.Record.FileInfo.Lname,
							Size:      result.Record.FileInfo.Lsize,
							Mode:      uint32(result.Record.FileInfo.Lmode),
							ModTime:   result.Record.FileInfo.LmodTime,
							Dev:       result.Record.FileInfo.Ldev,
							Ino:       result.Record.FileInfo.Lino,
							Uid:       result.Record.FileInfo.Luid,
							Gid:       result.Record.FileInfo.Lgid,
							Nlink:     uint32(result.Record.FileInfo.Lnlink),
							Username:  result.Record.FileInfo.Lusername,
							Groupname: result.Record.FileInfo.Lgroupname,
							Flags:     result.Record.FileInfo.Flags,
						},
						FileAttributes: result.Record.FileAttributes,
						Xattr:          xattr,
					},
				},
			}); err != nil {
				return err
			}
		case result.Error != nil:
			if err := stream.Send(&sdk.ScanResponse{
				Pathname: result.Error.Pathname,
				Result: &sdk.ScanResponseError{
					Error: &sdk.ScanError{
						Message: result.Error.Err.Error(),
					},
				},
			}); err != nil {
				return err
			}
		default:
			panic("?? unknown result type ??")
		}
	}
	return nil
}

func (imp *PlakarImporterFS) walkDir_walker(results chan<- *impor.ScanResult, rootDir, realp string, numWorkers int) {
	jobs := make(chan string, 1000)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go walkDir_worker(jobs, results, &wg)
	}

	walkDir_addPrefixDirectories(realp, jobs, results)
	if realp != rootDir {
		jobs <- rootDir
		walkDir_addPrefixDirectories(rootDir, jobs, results)
	}

	err := filepath.WalkDir(realp, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			results <- impor.NewScanError(path, err)
			return nil
		}
		jobs <- path
		return nil
	})
	if err != nil {
		results <- impor.NewScanError(realp, err)
	}

	close(jobs)
	wg.Wait()
	close(results)
}

func realpathFollow(path string) (resolved string, err error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		realpath, err := os.Readlink(path)
		if err != nil {
			return "", err
		}

		if !filepath.IsAbs(realpath) {
			realpath = filepath.Join(filepath.Dir(path), realpath)
		}
		path = realpath
	}

	return path, nil
}

func walkDir_worker(jobs <-chan string, results chan<- *impor.ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range jobs {
		info, err := os.Lstat(path)
		if err != nil {
			results <- impor.NewScanError(path, err)
			continue
		}

		extendedAttributes, err := xattr.List(path)
		if err != nil {
			results <- impor.NewScanError(path, err)
			continue
		}

		fileinfo := objects.FileInfoFromStat(info)
		var originFile string
		if fileinfo.Mode()&os.ModeSymlink != 0 {
			originFile, err = os.Readlink(path)
			if err != nil {
				results <- impor.NewScanError(path, err)
				continue
			}
		}
		results <- impor.NewScanRecord(filepath.ToSlash(path), originFile, fileinfo, extendedAttributes)
		for _, attr := range extendedAttributes {
			results <- impor.NewScanXattr(filepath.ToSlash(path), attr, objects.AttributeExtended)
		}
	}
}

func walkDir_addPrefixDirectories(rootDir string, jobs chan<- string, results chan<- *impor.ScanResult) {
	atoms := strings.Split(rootDir, string(os.PathSeparator))

	for i := 0; i < len(atoms)-1; i++ {
		path := filepath.Join(atoms[0 : i+1]...)

		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		if _, err := os.Stat(path); err != nil {
			results <- impor.NewScanError(path, err)
			continue
		}

		jobs <- path
	}
}

func (imp *PlakarImporterFS) Read(req *sdk.ReadRequest, stream sdk.ReadResponseStramer) error {
	file, err := os.Open(req.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, 8192)
	for {
		n, err := file.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if err := stream.Send(&sdk.ReadResponse{
			Data: buf[:n],
		}); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <scan-dir>\n", os.Args[0])
		os.Exit(1)
	}

	scanDir := os.Args[1]
	fsImporter, err := NewPlakarImporterFS(scanDir)
	if err != nil {
		panic(err)
	}

	if err := sdk.RunImporter(fsImporter); err != nil {
		panic(err)
	}
}
