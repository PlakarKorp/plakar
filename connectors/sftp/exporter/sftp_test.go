package sftp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
)

func TestExporter(t *testing.T) {
	// Create a mock SFTP server that accepts the public key
	server, err := ptesting.NewMockSFTPServer(t)
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "sftp-exporter-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some test files
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}

	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Create the exporter
	ctx := context.Background()
	opts := exporter.Options{}
	exporter, err := NewSFTPExporter(ctx, &opts, "sftp", map[string]string{
		"location":                 "sftp://" + server.Addr + "/",
		"username":                 "test",
		"identity":                 server.KeyFile,
		"insecure_ignore_host_key": "true",
	})
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	defer exporter.Close(ctx)

	// Test root path
	root, _ := exporter.Root(ctx)
	if root != "/" {
		t.Errorf("Expected root path '/', got '%s'", root)
	}

	// Test creating directories
	dirs := []string{"dir1", "dir2", "dir3"}
	for _, dir := range dirs {
		if err := exporter.CreateDirectory(ctx, dir); err != nil {
			t.Errorf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Test storing files
	for name, _ := range testFiles {
		path := filepath.Join(tmpDir, name)
		fp, err := os.Open(path)
		if err != nil {
			t.Fatalf("Failed to open file %s: %v", name, err)
		}
		defer fp.Close()

		fileInfo, err := fp.Stat()
		if err != nil {
			t.Fatalf("Failed to stat file %s: %v", name, err)
		}

		if err := exporter.StoreFile(ctx, name, fp, fileInfo.Size()); err != nil {
			t.Errorf("Failed to store file %s: %v", name, err)
		}
	}

	// Test setting permissions
	fileInfo := &objects.FileInfo{
		Lmode: 0644,
	}
	if err := exporter.SetPermissions(ctx, "file1.txt", fileInfo); err != nil {
		t.Errorf("Failed to set permissions: %v", err)
	}
}
