package testing

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/importer"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/stretchr/testify/require"
)

type MockFile struct {
	Path    string
	IsDir   bool
	Mode    os.FileMode
	Content []byte
}

func NewMockDir(path string) MockFile {
	return MockFile{
		Path:  path,
		IsDir: true,
		Mode:  0755,
	}
}

func NewMockFile(path string, mode os.FileMode, content string) MockFile {
	return MockFile{
		Path:    path,
		Mode:    mode,
		Content: []byte(content),
	}
}

func (m *MockFile) ScanResult() *importer.ScanResult {
	switch {
	case m.IsDir:
		return &importer.ScanResult{
			Record: &importer.ScanRecord{
				Pathname: m.Path,
				FileInfo: objects.FileInfo{
					Lname:      path.Base(m.Path),
					Lmode:      os.ModeDir | 0755,
					Lnlink:     1,
					Lusername:  "flan",
					Lgroupname: "hacker",
				},
			},
		}
	default:
		info := objects.FileInfo{
			Lname:      path.Base(m.Path),
			Lsize:      int64(len(m.Content)),
			Lmode:      m.Mode,
			Lnlink:     1,
			Lusername:  "flan",
			Lgroupname: "hacker",
		}
		return importer.NewScanRecord(m.Path, "", info, nil, func() (io.ReadCloser, error) {
			if m.Mode&0400 == 0 {
				return nil, os.ErrPermission
			}
			return io.NopCloser(bytes.NewReader(m.Content)), nil
		})
	}
}

type testingOptions struct {
	name string
	gen  func(chan<- *importer.ScanResult)
}

func newTestingOptions() *testingOptions {
	return &testingOptions{
		name: "test_backup",
	}
}

type TestingOptions func(o *testingOptions)

func WithName(name string) TestingOptions {
	return func(o *testingOptions) {
		o.name = name
	}
}

func GenerateFiles(t *testing.T, files []MockFile) string {
	tmpBackupDir, err := os.MkdirTemp("", "tmp_to_backup")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpBackupDir)
	})

	for _, file := range files {
		dest := filepath.Join(tmpBackupDir, filepath.FromSlash(file.Path))
		if file.IsDir {
			err = os.MkdirAll(dest, file.Mode)
		} else {
			err = os.WriteFile(dest, file.Content, file.Mode)
		}
	}

	return tmpBackupDir
}

func WithGenerator(gen func(chan<- *importer.ScanResult)) TestingOptions {
	return func(o *testingOptions) {
		o.gen = gen
	}
}

func GenerateSnapshot(t *testing.T, repo *repository.Repository, files []MockFile, opts ...TestingOptions) *snapshot.Snapshot {
	o := newTestingOptions()
	for _, f := range opts {
		f(o)
	}

	// create a snapshot
	builder, err := snapshot.Create(repo, repository.DefaultType, "", objects.NilMac, &snapshot.BuilderOptions{Name: o.name})
	require.NoError(t, err)
	require.NotNil(t, builder)

	impopts := &importer.Options{
		Hostname:        "localhost",
		OperatingSystem: runtime.GOOS,
		Architecture:    runtime.GOARCH,
		CWD:             "/",
		MaxConcurrency:  1,
		Stdout:          os.Stdout,
		Stderr:          os.Stderr,
	}

	imp, err := NewMockImporter(repo.AppContext(), impopts, "mock", map[string]string{"location": "mock://place"})
	require.NoError(t, err)
	require.NotNil(t, imp)

	if o.gen != nil {
		imp.(*MockImporter).SetGenerator(o.gen)
	} else {
		imp.(*MockImporter).SetFiles(files)
	}

	builder.Backup(imp)

	err = builder.Repository().RebuildState()
	require.NoError(t, err)

	builder.Close()

	// reopen it
	snap, err := snapshot.Load(repo, builder.Header.Identifier)
	require.NoError(t, err)
	require.NotNil(t, snap)

	checkCache, err := repo.AppContext().GetCache().Check()
	require.NoError(t, err)
	snap.SetCheckCache(checkCache)

	return snap
}
