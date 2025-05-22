package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PlakarKorp/go-kloset-sdk/sdk"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/exporter"
)


type FSExporter struct {
	rootDir string
}

func NewFSExporter(appCtx *appcontext.AppContext, name string, config map[string]string) (exporter.Exporter, error) {
	return &FSExporter{
		rootDir: strings.TrimPrefix(config["location"], "fs://"),
	}, nil
}

func (p *FSExporter) Root() string {
	return p.rootDir
}

func (p *FSExporter) CreateDirectory(pathname string) error {
	return os.MkdirAll(pathname, 0700)
}

func (p *FSExporter) StoreFile(pathname string, fp io.Reader, size int64) error {
	f, err := os.Create(pathname)
	if err != nil {
		return err
	}

	if _, err := io.Copy(f, fp); err != nil {
		f.Close()
		return err
	}
	return nil
}

func (p *FSExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	if err := os.Chmod(pathname, fileinfo.Mode()); err != nil {
		return err
	}
	if os.Getuid() == 0 {
		if err := os.Chown(pathname, int(fileinfo.Uid()), int(fileinfo.Gid())); err != nil {
			return err
		}
	}
	return nil
}

func (p *FSExporter) Close() error {
	return nil
}


func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <scan-dir>\n", os.Args[0])
		os.Exit(1)
	}

	scanDir := os.Args[1]
	fsExporter, err := NewFSExporter(appcontext.NewAppContext(), "fs", map[string]string{"location": scanDir})
	if err != nil {
		panic(err)
	}

	if err := sdk.RunExporter(fsExporter); err != nil {
		panic(err)
	}
}
