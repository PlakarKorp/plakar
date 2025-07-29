package fs

import (
	"io"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestExporter(t *testing.T) {
	tmpExportDir, err := os.MkdirTemp("/tmp", "tmp_export*")
	require.NoError(t, err)
	tmpOriginDir, err := os.MkdirTemp("/tmp", "tmp_origin*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpExportDir)
		os.RemoveAll(tmpOriginDir)
	})

	var exporterInstance exporter.Exporter
	ctx := appcontext.NewAppContext()

	// Register the fs backen
	exporterInstance, err = exporter.NewExporter(ctx.GetInner(), map[string]string{"location": tmpExportDir})
	require.NoError(t, err)
	defer exporterInstance.Close(ctx)

	root, err := exporterInstance.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, tmpExportDir, root)

	data := []byte("test exporter fs")
	datalen := int64(len(data))

	err = os.WriteFile(tmpOriginDir+"/dummy.txt", data, 0644)
	require.NoError(t, err)

	fpOrigin, err := os.Open(tmpOriginDir + "/dummy.txt")
	require.NoError(t, err)
	defer fpOrigin.Close()

	err = exporterInstance.StoreFile(ctx, tmpExportDir+"/dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	fpExported, err := os.Open(tmpExportDir + "/dummy.txt")
	require.NoError(t, err)
	defer fpExported.Close()

	newContent, err := io.ReadAll(fpExported)
	require.NoError(t, err)

	require.Equal(t, string(data), string(newContent))

	err = exporterInstance.CreateDirectory(ctx, tmpExportDir+"/subdir")
	require.NoError(t, err)

	err = exporterInstance.SetPermissions(ctx, tmpExportDir+"/dummy.txt", &objects.FileInfo{Lmode: 0644})
	require.NoError(t, err)
}
