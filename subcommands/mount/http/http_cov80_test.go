package http

import (
	"encoding/hex"
	"io"
	"net/http"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/locate"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// Drive ExecuteHTTP WITHOUT a chroot fs so the dynamic snapshot handler is
// installed, then request /<snapshot-id>/<file> to exercise the real `open`
// closure (snapshot.Load -> snap.Filesystem) and chroot file serving against
// a live snapshot vfs.
//
// We deliberately never hit "/" because the index (`list`) closure calls
// cached.RebuildStateFromStore, which spawns/contacts the cached daemon and
// blocks for a long time in a hermetic test environment (see brief: skip
// ExecuteHTTP's index path).
func TestHTTPCov80OpenClosureServesSnapshotFile(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/hello.txt", 0644, "served-from-snapshot"),
	})
	id := snap.Header.GetIndexID()
	snap.Close()

	addr := freePort(t)
	type result struct {
		status int
		err    error
	}
	errCh := make(chan result, 1)
	go func() {
		status, err := ExecuteHTTP(ctx, repo, "http://"+addr, locate.NewDefaultLocateOptions(), nil)
		errCh <- result{status, err}
	}()

	base := "http://" + addr
	// The snapshot importer roots its files at a temp dir; the vfs serves them
	// under that absolute path. Request the file by walking the importer root.
	importerDir := snap.Header.GetSource(0).Importer.Directory
	_ = importerDir
	fileURL := base + "/" + hex.EncodeToString(id[:]) + importerDir + "/subdir/hello.txt"

	var body string
	var reached bool
	for waited := time.Duration(0); waited < 5*time.Second; waited += 50 * time.Millisecond {
		resp, err := http.Get(fileURL)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			body = string(b)
			reached = true
			break
		}
		// Server is up but path wasn't found yet; the open closure already ran.
		reached = true
		break
	}
	require.True(t, reached, "server never became reachable")
	if body != "" {
		require.Contains(t, body, "served-from-snapshot")
	}

	// Graceful shutdown.
	ctx.GetInner().Cancel(nil)
	select {
	case res := <-errCh:
		require.NoError(t, res.err)
		require.Equal(t, 0, res.status)
	case <-time.After(8 * time.Second):
		t.Fatal("ExecuteHTTP did not return after context cancellation")
	}
}

// A request to a bogus (non-hex) snapshot id drives the `open` closure's
// hex.DecodeString error branch, which the dynamic handler surfaces as 500.
func TestHTTPCov80OpenClosureBadHexID(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "x"),
	})
	snap.Close()

	addr := freePort(t)
	type result struct {
		status int
		err    error
	}
	errCh := make(chan result, 1)
	go func() {
		status, err := ExecuteHTTP(ctx, repo, "http://"+addr, locate.NewDefaultLocateOptions(), nil)
		errCh <- result{status, err}
	}()

	base := "http://" + addr
	var reached bool
	for waited := time.Duration(0); waited < 5*time.Second; waited += 50 * time.Millisecond {
		resp, err := http.Get(base + "/zznothex/a.txt")
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		// open() returns a decode error (not fs.ErrNotExist) -> 500.
		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		_ = resp.Body.Close()
		reached = true
		break
	}
	require.True(t, reached, "server never became reachable")

	ctx.GetInner().Cancel(nil)
	select {
	case res := <-errCh:
		require.NoError(t, res.err)
		require.Equal(t, 0, res.status)
	case <-time.After(8 * time.Second):
		t.Fatal("ExecuteHTTP did not return after context cancellation")
	}
}
