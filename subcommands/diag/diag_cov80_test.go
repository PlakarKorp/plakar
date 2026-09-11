package diag

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

var cov80zeroTime = time.Unix(0, 0)

// cov80Snapshot builds a snapshot whose tree contains a regular file, a symlink
// and a file that fails to open (permission denied). This lets the diag vfs
// command exercise the SymlinkTarget rendering and the fs.Errors loop, both of
// which the existing tests do not reach. Returns the repo, ctx, output buffer
// and the snapshot index id (hex).
func cov80Snapshot(t *testing.T) (*repository.Repository, *appcontext.AppContext, *bytes.Buffer, string) {
	t.Helper()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	gen := func(ch chan<- *connectors.Record) {
		ch <- &connectors.Record{
			Pathname: "/",
			FileInfo: objects.NewFileInfo("/", 0, 0700|os.ModeDir, cov80zeroTime, 0, 0, 0, 0, 1),
		}
		ch <- &connectors.Record{
			Pathname: "/d",
			FileInfo: objects.NewFileInfo("d", 0, 0700|os.ModeDir, cov80zeroTime, 0, 0, 0, 0, 1),
		}
		ch <- connectors.NewRecord("/d/file.txt", "",
			objects.NewFileInfo("file.txt", 5, 0644, cov80zeroTime, 0, 0, 0, 0, 1),
			nil,
			func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("hello")), nil })
		// symlink pointing at the file: second NewRecord arg is the target.
		ch <- connectors.NewRecord("/d/link", "/d/file.txt",
			objects.NewFileInfo("link", 0, os.ModeSymlink|0777, cov80zeroTime, 0, 0, 0, 0, 1),
			nil, nil)
		// a file whose reader errors -> recorded as a scan error in the snapshot.
		ch <- connectors.NewRecord("/d/denied.txt", "",
			objects.NewFileInfo("denied.txt", 3, 0000, cov80zeroTime, 0, 0, 0, 0, 1),
			nil,
			func() (io.ReadCloser, error) { return nil, os.ErrPermission })
	}

	snap := ptesting.GenerateSnapshot(t, repo, nil, ptesting.WithGenerator(gen))
	t.Cleanup(func() { snap.Close() })
	indexID := snap.Header.GetIndexID()
	bufOut.Reset()
	return repo, ctx, bufOut, hex.EncodeToString(indexID[:])
}

// --- vfs: directory listing that surfaces a recorded scan error ------------
// The "denied.txt" reader fails during backup, so the snapshot records a scan
// error under /d. Running `diag vfs /d` walks children and the fs.Errors loop
// emits an "Error[...]" line, a path the existing tests do not reach.

func TestCov80DiagVFSScanErrors(t *testing.T) {
	repo, ctx, bufOut, id := cov80Snapshot(t)

	bufOut.Reset()
	status, err := runDiag80(t, ctx, repo, []string{"diag", "vfs", fmt.Sprintf("%s:/d", id)})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	out := bufOut.String()
	require.Contains(t, out, "[DirEntry]")
	require.Contains(t, out, "Error[")
}

// --- contenttype: listing over a populated tree (loop body) ----------------

func TestCov80DiagContentTypeListing(t *testing.T) {
	repo, ctx, bufOut, id := cov80Snapshot(t)

	bufOut.Reset()
	status, err := runDiag80(t, ctx, repo, []string{"diag", "contenttype", fmt.Sprintf("%s:/d/", id)})
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// --- search: single-arg form (no mime filter) over a populated subtree ------

func TestCov80DiagSearchNoMime(t *testing.T) {
	repo, ctx, bufOut, id := cov80Snapshot(t)

	bufOut.Reset()
	status, err := runDiag80(t, ctx, repo, []string{"diag", "search", fmt.Sprintf("%s:/d/", id)})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "file.txt")
}

func runDiag80(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository, args []string) (int, error) {
	t.Helper()
	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	if err := subcommand.Parse(ctx, rest); err != nil {
		return -1, err
	}
	return subcommand.Execute(ctx, repo)
}
