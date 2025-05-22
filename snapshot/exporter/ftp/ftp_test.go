package ftp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/objects"
	ptesting "github.com/PlakarKorp/plakar/testing"
)

func TestExporter(t *testing.T) {
	// Create a mock FTP server
	server, err := ptesting.NewMockFTPServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "ftp-exporter-test-*")
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
	appCtx := appcontext.NewAppContext()
	exporter, err := NewFTPExporter(appCtx, "ftp", map[string]string{
		"location": "ftp://" + server.Addr + "/",
		"username": "test",
		"password": "test",
	})
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	defer exporter.Close()

	// Test root path
	root := exporter.Root()
	if root != "/" {
		t.Errorf("Expected root path '/', got '%s'", root)
	}

	// Test creating directories
	dirs := []string{"dir1", "dir2", "dir3"}
	for _, dir := range dirs {
		if err := exporter.CreateDirectory(dir); err != nil {
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

		if err := exporter.StoreFile(name, fp, fileInfo.Size()); err != nil {
			t.Errorf("Failed to store file %s: %v", name, err)
		}
	}

	// Verify stored files
	for name, expectedContent := range testFiles {
		content, exists := server.Files[name]
		if !exists {
			t.Errorf("File %s was not stored in the FTP server", name)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("File %s content mismatch. Expected: %s, Got: %s", name, expectedContent, string(content))
		}
	}

	// Test setting permissions
	fileInfo := &objects.FileInfo{
		Lmode: 0644,
	}
	if err := exporter.SetPermissions("file1.txt", fileInfo); err != nil {
		t.Errorf("Failed to set permissions: %v", err)
	}
}
