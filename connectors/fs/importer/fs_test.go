package fs

import (
	"io"
	"os"
	"sort"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestFSImporter(t *testing.T) {
	tmpImportDir, err := os.MkdirTemp("/tmp", "tmp_import*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpImportDir)
	})

	err = os.WriteFile(tmpImportDir+"/dummy.txt", []byte("test importer fs"), 0644)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()

	importer, err := NewFSImporter(ctx, ctx.ImporterOpts(), "fs", map[string]string{"location": tmpImportDir})
	require.NoError(t, err)
	require.NotNil(t, importer)

	orig, err := importer.Origin(ctx)
	require.NoError(t, err)
	require.Equal(t, ctx.Hostname, orig)

	root, err := importer.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, tmpImportDir, root)

	typ, err := importer.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, "fs", typ)

	scanChan, err := importer.Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, scanChan)

	paths := []string{}
	for record := range scanChan {
		require.Nil(t, record.Error)
		if record.Record.IsXattr {
			continue
		}
		paths = append(paths, record.Record.Pathname)

		if record.Record.FileInfo.Mode().IsRegular() {
			content, err := io.ReadAll(record.Record.Reader)
			require.NoError(t, err)
			require.Equal(t, content, []byte("test importer fs"))
			record.Record.Reader.Close()
		}
	}
	expected := []string{"/", "/tmp", tmpImportDir, tmpImportDir + "/dummy.txt"}
	sort.Strings(paths)
	require.Equal(t, expected, paths)

	err = importer.Close(ctx)
	require.NoError(t, err)
}
