package ftp

import (
	"io"
	"strings"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestImporter(t *testing.T) {
	// Create a mock FTP server
	server, err := ptesting.NewMockFTPServer()
	require.NoError(t, err)
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
	ctx := appcontext.NewAppContext()
	importer, err := NewFTPImporter(ctx, nil, "ftp", map[string]string{
		"location": "ftp://" + server.Addr + "/",
	})
	require.NoError(t, err)
	defer importer.Close(ctx)

	// Test root path
	root, err := importer.Root(ctx)
	require.NoError(t, err)
	if root != "/" {
		t.Errorf("Expected root path '/', got '%s'", root)
	}

	// Test scanning files
	scanResults, err := importer.Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, scanResults)

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

		if result.Record.FileInfo.Mode().IsRegular() {
			content, err := io.ReadAll(result.Record.Reader)
			require.NoError(t, err)
			require.Equal(t, string(content), testFiles[strings.TrimPrefix(result.Record.Pathname, "/")])
			result.Record.Reader.Close()
		}
	}

	// Verify all test files were scanned
	for name := range testFiles {
		if !scannedFiles["/"+name] {
			t.Errorf("File /%s was not scanned", name)
		}
	}
}
