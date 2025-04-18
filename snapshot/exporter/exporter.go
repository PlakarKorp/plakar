package exporter

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
)

type Exporter interface {
	Root() string
	CreateDirectory(pathname string) error
	StoreFile(pathname string, fp io.Reader) error
	SetPermissions(pathname string, fileinfo *objects.FileInfo) error
	Close() error
}

var muBackends sync.Mutex
var backends map[string]func(config map[string]string) (Exporter, error) = make(map[string]func(config map[string]string) (Exporter, error))

func Register(name string, backend func(config map[string]string) (Exporter, error)) {
	muBackends.Lock()
	defer muBackends.Unlock()

	if _, ok := backends[name]; ok {
		log.Fatalf("backend '%s' registered twice", name)
	}
	backends[name] = backend
}

func Backends() []string {
	muBackends.Lock()
	defer muBackends.Unlock()

	ret := make([]string, 0)
	for backendName := range backends {
		ret = append(ret, backendName)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

func NewExporter(ctx *appcontext.AppContext, config map[string]string) (Exporter, error) {

	location, ok := config["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}

	muBackends.Lock()
	defer muBackends.Unlock()

	var backendName string
	if !strings.HasPrefix(location, "/") {
		if strings.HasPrefix(location, "s3://") {
			backendName = "s3"
		} else if strings.HasPrefix(location, "fs://") {
			backendName = "fs"
		} else if strings.HasPrefix(location, "ftp://") {
			backendName = "ftp"
		} else if strings.HasPrefix(location, "sftp://") {
			backendName = "sftp"
		} else {
			if strings.Contains(location, "://") {
				return nil, fmt.Errorf("unsupported exporter protocol")
			} else {
				backendName = "fs"
			}
		}
	} else {
		backendName = "fs"
	}

	if backend, exists := backends[backendName]; !exists {
		return nil, fmt.Errorf("backend '%s' does not exist", backendName)
	} else {
		backendInstance, err := backend(config)
		if err != nil && backendName != "fs" {
			return nil, err
		} else if err != nil {
			location = strings.TrimPrefix(location, "fs://")
			if !filepath.IsAbs(location) {
				location = filepath.Join(ctx.CWD, location)
				config["location"] = location
				if backendInstance, err = backend(config); err == nil {
					return backendInstance, nil
				}
			}
			return nil, err
		}
		return backendInstance, nil
	}
}
