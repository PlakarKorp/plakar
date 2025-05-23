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

	appCtx := appcontext.NewAppContext()

	importer, err := NewFSImporter(appCtx, "fs", map[string]string{"location": tmpImportDir})
	require.NoError(t, err)
	require.NotNil(t, importer)

	origin := importer.Origin()
	require.NotEmpty(t, origin)

	root := importer.Root()
	require.NoError(t, err)
	require.Equal(t, tmpImportDir, root)

	typ := importer.Type()
	require.Equal(t, "fs", typ)

	scanChan, err := importer.Scan()
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

	// cannot test this that as filesystem does not necessarly have the xattr enabled
	// err = xattr.Set(tmpImportDir+"/dummy.txt", "user.plakar.test", []byte("random.value"))
	// require.NoError(t, err)
	// extendedAttrReader, err := importer.NewExtendedAttributeReader(tmpImportDir+"/dummy.txt", "user.plakar.test")
	// require.NoError(t, err)
	// require.NotNil(t, extendedAttrReader)
	// defer extendedAttrReader.Close()

	err = importer.Close()
	require.NoError(t, err)
}
