package stdio

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	kimporter "github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/stretchr/testify/require"
)

func TestStdioImporter(t *testing.T) {
	// Test data
	origContent := []byte("test importer stdin")
	r := bytes.NewReader(origContent)

	// Create the importer with a properly initialized AppContext
	ctx := context.Background()

	hostname, err := os.Hostname()
	require.NoError(t, err)

	importer, err := NewStdioImporter(ctx, &kimporter.Options{
		Hostname: hostname,
		Stdin:    r,
	}, "stdin", map[string]string{"location": "stdin:///test.txt"})
	require.NoError(t, err)
	require.NotNil(t, importer)

	// Test basic properties
	require.Equal(t, "/", importer.Root())
	require.Equal(t, "stdin", importer.Type())
	require.Equal(t, hostname, importer.Origin())

	// Test scanning
	scanChan, err := importer.Scan()
	require.NoError(t, err)
	require.NotNil(t, scanChan)

	// Collect scan results
	paths := []string{}
	for record := range scanChan {
		require.Nil(t, record.Error)
		paths = append(paths, record.Record.Pathname)

		if record.Record.FileInfo.Mode().IsRegular() {
			defer record.Record.Reader.Close()
			content, err := io.ReadAll(record.Record.Reader)
			require.NoError(t, err)
			require.Equal(t, origContent, content)
		}
	}
	require.Equal(t, []string{"/", "/test.txt"}, paths)

	// Test close
	err = importer.Close()
	require.NoError(t, err)
}
