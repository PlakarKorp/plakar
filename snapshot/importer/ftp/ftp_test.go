package ftp

import (
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
)

func TestImporter(t *testing.T) {
	// Create a mock FTP server
	server, err := ptesting.NewMockFTPServer()
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}
	defer server.Close()

	// Allow anonymous login for the test
	server.SetAuth("anonymous", "anonymous")

	// Create some test files in the mock server
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}

	for name, content := range testFiles {
		server.Files[name] = []byte(content)
	}

	// Create the importer
	appCtx := appcontext.NewAppContext()
	importer, err := NewFTPImporter(appCtx, "ftp", map[string]string{
		"location": "ftp://" + server.Addr + "/",
	})
	if err != nil {
		t.Fatalf("Failed to create importer: %v", err)
	}
	defer importer.Close()

	// Test root path
	root := importer.Root()
	if root != "/" {
		t.Errorf("Expected root path '/', got '%s'", root)
	}

	// Test scanning files
	scanResults, err := importer.Scan()
	if err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	// Collect scan results
	scannedFiles := make(map[string]bool)
	for result := range scanResults {
		if result.Error != nil {
			if result.Error.Pathname == "/" {
				// Ignore scan errors for root directory
				continue
			}
			t.Errorf("Scan error for %s: %v", result.Error.Pathname, result.Error.Err)
			continue
		}
		if result.Record != nil {
			scannedFiles[result.Record.Pathname] = true
		}
	}

	// Verify all test files were scanned
	for name := range testFiles {
		if !scannedFiles["/"+name] {
			t.Errorf("File /%s was not scanned", name)
		}
	}

	// Test reading file contents
	for name, expectedContent := range testFiles {
		reader, err := importer.NewReader(name)
		if err != nil {
			t.Errorf("Failed to get reader for %s: %v", name, err)
			continue
		}
		defer reader.Close()

		content := make([]byte, len(expectedContent))
		_, err = reader.Read(content)
		if err != nil {
			t.Errorf("Failed to read content of %s: %v", name, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s", name, expectedContent, string(content))
		}
	}
}
