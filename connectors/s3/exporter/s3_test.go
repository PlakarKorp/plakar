package s3

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/require"
)

func TestExporter(t *testing.T) {
	tmpOriginDir, err := os.MkdirTemp("/tmp", "tmp_origin*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpOriginDir)
	})

	// Start the fake S3 server
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())
	defer ts.Close()

	tmpExportBucket := "s3://" + ts.Listener.Addr().String() + "/bucket"

	var exporterInstance exporter.Exporter
	ctx := appcontext.NewAppContext()
	exporterInstance, err = exporter.NewExporter(ctx.GetInner(), map[string]string{"location": tmpExportBucket, "access_key": "", "secret_access_key": "", "use_tls": "false"})
	require.NoError(t, err)
	defer exporterInstance.Close(ctx)

	root, err := exporterInstance.Root(ctx)
	require.NoError(t, err)
	require.Equal(t, "/bucket", root)

	data := []byte("test exporter s3")
	datalen := int64(len(data))

	// create a temporary file to backup later
	err = os.WriteFile(tmpOriginDir+"/dummy.txt", data, 0644)
	require.NoError(t, err)

	fpOrigin, err := os.Open(tmpOriginDir + "/dummy.txt")
	require.NoError(t, err)
	defer fpOrigin.Close()

	err = exporterInstance.StoreFile(ctx, "dummy.txt", fpOrigin, datalen)
	require.NoError(t, err)

	err = exporterInstance.CreateDirectory(ctx, "/bucket/subdir")
	require.NoError(t, err)

	err = exporterInstance.SetPermissions(ctx, "bucket/subdir", &objects.FileInfo{Lmode: 0644})
	require.NoError(t, err)
}
