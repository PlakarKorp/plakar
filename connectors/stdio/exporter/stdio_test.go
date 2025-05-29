package stdio

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestExporter(t *testing.T) {
	// Create a temporary directory for test files
	tmpOriginDir, err := os.MkdirTemp("/tmp", "tmp_origin*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpOriginDir)
	})

	// Create a buffer to capture stdout
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test stdout exporter
	appCtx := appcontext.NewAppContext()
	stdoutExporter, err := exporter.NewExporter(appCtx.GetInner(), map[string]string{"location": "stdout://"})
	require.NoError(t, err)
	defer stdoutExporter.Close()

	require.Equal(t, "/", stdoutExporter.Root())

	// Create test data
	data := []byte("test exporter stdout")
	datalen := int64(len(data))

	// Write test data to a temporary file
	err = os.WriteFile(tmpOriginDir+"/dummy.txt", data, 0644)
	require.NoError(t, err)

	// Open the test file
	fpOrigin, err := os.Open(tmpOriginDir + "/dummy.txt")
	require.NoError(t, err)
	defer fpOrigin.Close()

	// Store the file
	err = stdoutExporter.StoreFile("/dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	// Restore stdout and read the output
	w.Close()
	os.Stdout = oldStdout
	io.Copy(&buf, r)

	// Verify the content
	require.Equal(t, string(data), buf.String())

	// Test directory creation (should succeed but do nothing)
	err = stdoutExporter.CreateDirectory("/subdir")
	require.NoError(t, err)

	// Test setting permissions (should succeed but do nothing)
	err = stdoutExporter.SetPermissions("/dummy.txt", &objects.FileInfo{Lmode: 0644})
	require.NoError(t, err)

	// Create a buffer to capture stderr
	buf.Reset()
	oldStderr := os.Stderr
	r, w, _ = os.Pipe()
	os.Stderr = w

	// Test stderr exporter
	stderrExporter, err := exporter.NewExporter(appCtx.GetInner(), map[string]string{"location": "stderr://"})
	require.NoError(t, err)
	defer stderrExporter.Close()

	require.Equal(t, "/", stderrExporter.Root())

	// Reset the test file position
	fpOrigin.Seek(0, 0)

	// Store the file to stderr
	err = stderrExporter.StoreFile("/dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	// Restore stderr and read the output
	w.Close()
	os.Stderr = oldStderr
	io.Copy(&buf, r)

	// Verify the content
	require.Equal(t, string(data), buf.String())

	// Test invalid backend
	_, err = exporter.NewExporter(appCtx.GetInner(), map[string]string{"location": "invalid://"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported exporter protocol")
}
