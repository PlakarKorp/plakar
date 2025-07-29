package ftp

import (
	"bytes"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
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

	// Create the exporter
	ctx := appcontext.NewAppContext()
	exporter, err := NewFTPExporter(ctx, nil, "ftp", map[string]string{
		"location": "ftp://" + server.Addr + "/",
		"username": "test",
		"password": "test",
	})
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	defer exporter.Close(ctx)

	// Test root path
	root, err := exporter.Root(ctx)
	require.NoError(t, err)
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
	for name, content := range testFiles {
		fp := bytes.NewReader([]byte(content))
		if err := exporter.StoreFile(name, fp, int64(len(content))); err != nil {
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
