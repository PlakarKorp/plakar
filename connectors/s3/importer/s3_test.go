package s3

import (
	"net/http/httptest"
	"os"
	"sort"
	"testing"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/require"
)

func TestS3Importer(t *testing.T) {
	tmpImportDir, err := os.MkdirTemp("/tmp", "tmp_import*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpImportDir)
	})

	err = os.WriteFile(tmpImportDir+"/dummy.txt", []byte("test importer s3"), 0644)
	require.NoError(t, err)

	fpOrigin, err := os.Open(tmpImportDir + "/dummy.txt")
	require.NoError(t, err)
	defer fpOrigin.Close()

	// Start the fake S3 server
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	tmpImportBucket := "s3://" + ts.Listener.Addr().String() + "/bucket"

	backend.CreateBucket("bucket")
	_, err = backend.PutObject("bucket", "dummy.txt", nil, fpOrigin, 16)
	require.NoError(t, err)

	ctx := appcontext.NewAppContext()

	importer, err := NewS3Importer(ctx, ctx.ImporterOpts(), "s3", map[string]string{"location": tmpImportBucket, "access_key": "", "secret_access_key": "", "use_tls": "false"})
	require.NoError(t, err)
	require.NotNil(t, importer)

	origin, err := importer.Origin(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, origin)

	root, err := importer.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, "/", root)

	typ, err := importer.Type(ctx)
	require.NoError(t, err)
	require.Equal(t, "s3", typ)

	scanChan, err := importer.Scan(ctx)
	require.NoError(t, err)
	require.NotNil(t, scanChan)

	paths := []string{}
	for record := range scanChan {
		require.Nil(t, record.Error)
		paths = append(paths, record.Record.Pathname)

		// if record.Record.Pathname == "/dummy.txt" {
		// 	content, err := io.ReadAll(record.Record.Reader)
		// 	require.NoError(t, err)
		// 	require.Equal(t, content, []byte("test importer s3"))
		// 	record.Record.Reader.Close()
		// }
	}

	expected := []string{"/", "/dummy.txt"}
	sort.Strings(paths)
	require.Equal(t, expected, paths)

	err = importer.Close(ctx)
	require.NoError(t, err)
}
