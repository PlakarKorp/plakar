package stdio

import (
	"io"
	"os"
	"testing"

	kimporter "github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestStdioImporter(t *testing.T) {
	// Create a pipe to capture stdin
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Test data
	testData := []byte("test importer stdin")
	_, err := w.Write(testData)
	require.NoError(t, err)
	w.Close()

	// Create the importer with a properly initialized AppContext
	appCtx := appcontext.NewAppContext()
	hostname, err := os.Hostname()
	require.NoError(t, err)
	appCtx.Hostname = hostname

	importer, err := kimporter.NewImporter(appCtx.GetInner(), &kimporter.ImporterOptions{
		Hostname: hostname,
	}, map[string]string{"location": "stdin:///test.txt"})
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
			require.Equal(t, content, []byte("test importer stdin"))
		}
	}
	require.Equal(t, []string{"/", "/test.txt"}, paths)

	// Test close
	err = importer.Close()
	require.NoError(t, err)

	// Restore stdin
	os.Stdin = oldStdin
}
