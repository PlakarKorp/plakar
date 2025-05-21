package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"bytes"
	"path"
	"runtime"

	"github.com/PlakarKorp/go-kloset-sdk/sdk"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/pkg/xattr"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
)

type FSImporter struct {
	ctx     *appcontext.AppContext
	rootDir string

	uidToName map[uint64]string
	gidToName map[uint64]string
	mu        sync.RWMutex
}

func NewFSImporter(appCtx *appcontext.AppContext, name string, config map[string]string) (importer.Importer, error) {
	location := config["location"]
	rootDir := strings.TrimPrefix(location, "fs://")

	if !path.IsAbs(rootDir) {
		return nil, fmt.Errorf("not an absolute path %s", location)
	}

	rootDir = path.Clean(rootDir)

	return &FSImporter{
		ctx:       appCtx,
		rootDir:   rootDir,
		uidToName: make(map[uint64]string),
		gidToName: make(map[uint64]string),
	}, nil
}

func (p *FSImporter) Origin() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return hostname
}

func (p *FSImporter) Type() string {
	return "fs"
}

func (p *FSImporter) Scan() (<-chan *importer.ScanResult, error) {
	realp, err := p.realpathFollow(p.rootDir)
	if err != nil {
		return nil, err
	}

	results := make(chan *importer.ScanResult, 1000)
	go p.walkDir_walker(results, p.rootDir, realp, 256)
	return results, nil
}

func (f *FSImporter) walkDir_walker(results chan<- *importer.ScanResult, rootDir, realp string, numWorkers int) {
	jobs := make(chan string, 1000) // Buffered channel to feed paths to workers
	var wg sync.WaitGroup
	for range numWorkers {
		wg.Add(1)
		go f.walkDir_worker(jobs, results, &wg)
	}

	// Add prefix directories first
	walkDir_addPrefixDirectories(realp, jobs, results)
	if realp != rootDir {
		jobs <- rootDir
		walkDir_addPrefixDirectories(rootDir, jobs, results)
	}

	err := filepath.WalkDir(realp, func(path string, d fs.DirEntry, err error) error {
		if f.ctx.Err() != nil {
			return err
		}

		if err != nil {
			results <- importer.NewScanError(path, err)
			return nil
		}
		jobs <- path
		return nil
	})
	if err != nil {
		results <- importer.NewScanError(realp, err)
	}

	close(jobs)
	wg.Wait()
	close(results)
}

func (f *FSImporter) walkDir_worker(jobs <-chan string, results chan<- *importer.ScanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		var (
			path string
			ok   bool
		)

		select {
		case path, ok = <-jobs:
			if !ok {
				return
			}
		case <-f.ctx.Done():
			return
		}

		info, err := os.Lstat(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		extendedAttributes, err := xattr.List(path)
		if err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		fileinfo := objects.FileInfoFromStat(info)
		fileinfo.Lusername, fileinfo.Lgroupname = f.lookupIDs(fileinfo.Uid(), fileinfo.Gid())

		var originFile string
		if fileinfo.Mode()&os.ModeSymlink != 0 {
			originFile, err = os.Readlink(path)
			if err != nil {
				results <- importer.NewScanError(path, err)
				continue
			}
		}
		results <- importer.NewScanRecord(filepath.ToSlash(path), originFile, fileinfo, extendedAttributes)
		for _, attr := range extendedAttributes {
			results <- importer.NewScanXattr(filepath.ToSlash(path), attr, objects.AttributeExtended)
		}
	}
}

func walkDir_addPrefixDirectories(rootDir string, jobs chan<- string, results chan<- *importer.ScanResult) {
	atoms := strings.Split(rootDir, string(os.PathSeparator))

	for i := range len(atoms) - 1 {
		path := filepath.Join(atoms[0 : i+1]...)

		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}

		if _, err := os.Stat(path); err != nil {
			results <- importer.NewScanError(path, err)
			continue
		}

		jobs <- path
	}
}


func (p *FSImporter) lookupIDs(uid, gid uint64) (uname, gname string) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if name, ok := p.uidToName[uid]; !ok {
		if u, err := user.LookupId(fmt.Sprint(uid)); err == nil {
			uname = u.Username

			p.mu.RUnlock()
			p.mu.Lock()
			p.uidToName[uid] = uname
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		uname = name
	}

	if name, ok := p.gidToName[gid]; !ok {
		if g, err := user.LookupGroupId(fmt.Sprint(gid)); err == nil {
			gname = g.Name

			p.mu.RUnlock()
			p.mu.Lock()
			p.gidToName[gid] = name
			p.mu.Unlock()
			p.mu.RLock()
		}
	} else {
		gname = name
	}

	return
}

func (f *FSImporter) realpathFollow(path string) (resolved string, err error) {
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

func (p *FSImporter) NewReader(pathname string) (io.ReadCloser, error) {
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}
	return os.Open(pathname)
}

func (p *FSImporter) NewExtendedAttributeReader(pathname string, attribute string) (io.ReadCloser, error) {
	if pathname[0] == '/' && runtime.GOOS == "windows" {
		pathname = pathname[1:]
	}

	data, err := xattr.Get(pathname, attribute)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (p *FSImporter) Close() error {
	return nil
}

func (p *FSImporter) Root() string {
	return p.rootDir
}


func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <scan-dir>\n", os.Args[0])
		os.Exit(1)
	}

	scanDir := os.Args[1]
	fsImporter, err := NewFSImporter(appcontext.NewAppContext(), "fs", map[string]string{"location": scanDir})
	if err != nil {
		panic(err)
	}

	if err := sdk.RunImporter(fsImporter); err != nil {
		panic(err)
	}
}
