package stdio

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
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

	// Test stdout exporter
	ctx := context.Background()
	stdoutExporter, err := NewStdioExporter(ctx, &exporter.Options{
		Stdout: &buf,
	}, "stdout", map[string]string{"location": "stdout://"})
	require.NoError(t, err)
	defer stdoutExporter.Close(ctx)

	root, err := stdoutExporter.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, "/", root)

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
	err = stdoutExporter.StoreFile(ctx, "/dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	// Verify the content
	require.Equal(t, string(data), buf.String())

	// Test directory creation (should succeed but do nothing)
	err = stdoutExporter.CreateDirectory(ctx, "/subdir")
	require.NoError(t, err)

	// Test setting permissions (should succeed but do nothing)
	err = stdoutExporter.SetPermissions(ctx, "/dummy.txt", &objects.FileInfo{Lmode: 0644})
	require.NoError(t, err)

	// Create a buffer to capture stderr
	buf.Reset()

	// Test stderr exporter
	stderrExporter, err := NewStdioExporter(ctx, &exporter.Options{
		Stderr: &buf,
	}, "stderr", map[string]string{
		"location": "stderr://",
	})
	require.NoError(t, err)
	defer stderrExporter.Close(ctx)

	root, err = stderrExporter.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, "/", root)

	// Reset the test file position
	fpOrigin.Seek(0, 0)

	// Store the file to stderr
	err = stderrExporter.StoreFile(ctx, "/dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	// Verify the content
	require.Equal(t, string(data), buf.String())

	// Test invalid backend
	_, err = NewStdioExporter(ctx, nil, "invalid",
		map[string]string{"location": "invalid://"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown stdio backend")
}
