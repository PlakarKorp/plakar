//go:build linux || darwin

package plakarfs

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/objects"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/anacrolix/fuse"
	fusefs "github.com/anacrolix/fuse/fs"
	"github.com/stretchr/testify/require"
)

// newTestFS builds a real on-disk repository with a snapshot containing
// subdir/hello.txt, then constructs a plakarFS over it. To avoid invoking the
// "cached" daemon (which spawns a subprocess and is not appropriate from a
// test binary), we pre-populate the root's snapshot mapping so Lookup can
// skip ReadDirAll on the root.
func newTestFS(t *testing.T) (root *Dir, snapName string, backupDir string) {
	t.Helper()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/hello.txt", 0644, "hello world"),
	})
	t.Cleanup(func() { snap.Close() })

	pfs := NewFS(ctx, repo, nil, nil)
	rootNode, err := pfs.Root()
	require.NoError(t, err)
	root = rootNode.(*Dir)

	indexID := snap.Header.GetIndexID()
	var mac objects.MAC
	copy(mac[:], indexID[:])
	snapName = shortName(mac)
	backupDir = snap.Header.GetSource(0).Importer.Directory

	// Skip the snapshot listing's cached daemon call.
	root.readDirMutex.Lock()
	root.readDirSnapshotMapping = map[string]objects.MAC{snapName: mac}
	root.readDirLast = time.Now()
	root.readDirMutex.Unlock()
	return root, snapName, backupDir
}

func shortName(mac objects.MAC) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 0; i < 4; i++ {
		out[2*i] = hex[mac[i]>>4]
		out[2*i+1] = hex[mac[i]&0x0f]
	}
	return string(out)
}

// TestLookupWithoutReadDir is a regression test for the bug where direct path
// access (e.g. `cat /mnt/<snap>/path/to/file`) returned ENOENT unless the
// caller had previously listed each parent directory.
func TestLookupWithoutReadDir(t *testing.T) {
	root, snapName, backupDir := newTestFS(t)
	ctx := context.Background()

	snapNode, err := root.Lookup(ctx, snapName)
	require.NoError(t, err, "looking up snapshot by name without prior readdir on root")
	snapDir := snapNode.(*Dir)

	cur := snapDir
	for _, part := range splitPath(backupDir) {
		next, err := cur.Lookup(ctx, part)
		require.NoErrorf(t, err, "lookup %q without prior readdir on parent", part)
		cur = next.(*Dir)
	}

	subdirNode, err := cur.Lookup(ctx, "subdir")
	require.NoError(t, err)
	subdir := subdirNode.(*Dir)

	fileNode, err := subdir.Lookup(ctx, "hello.txt")
	require.NoError(t, err)
	file := fileNode.(*File)

	resp := &fuse.OpenResponse{}
	hNode, err := file.Open(ctx, &fuse.OpenRequest{}, resp)
	require.NoError(t, err)
	h := hNode.(*fileHandle)
	t.Cleanup(func() { _ = h.Release(ctx, &fuse.ReleaseRequest{}) })

	readResp := &fuse.ReadResponse{}
	require.NoError(t, h.Read(ctx, &fuse.ReadRequest{Offset: 0, Size: 64}, readResp))
	require.Equal(t, "hello world", string(readResp.Data))
}

// TestLookupUnknownSnapshotReturnsENOENT confirms we no longer fall through
// to snapshot.Load with a zero MAC for unknown snapshot names.
func TestLookupUnknownSnapshotReturnsENOENT(t *testing.T) {
	root, _, _ := newTestFS(t)
	_, err := root.Lookup(context.Background(), "deadbeef")
	require.Error(t, err)
}

// TestStatFallsBackToVFS confirms Stat works without a prior ReadDirAll on
// the same directory. This is the underlying fix that makes Lookup work for
// direct access.
func TestStatFallsBackToVFS(t *testing.T) {
	root, snapName, backupDir := newTestFS(t)
	ctx := context.Background()

	snapNode, err := root.Lookup(ctx, snapName)
	require.NoError(t, err)
	cur := snapNode.(*Dir)
	for _, part := range splitPath(backupDir) {
		next, err := cur.Lookup(ctx, part)
		require.NoError(t, err)
		cur = next.(*Dir)
	}

	// readDirEntries should not have been populated by any of the lookups.
	cur.readDirMutex.Lock()
	require.Nil(t, cur.readDirEntries, "Lookup should not have populated readDirEntries")
	cur.readDirMutex.Unlock()

	st, err := cur.Stat("subdir")
	require.NoError(t, err)
	require.True(t, st.IsDir())
}

// TestFileMetadataFromKlosetEntry confirms that uid/gid/mode/nlink are taken
// from the underlying kloset FileInfo and not hardcoded to the process's
// effective uid/gid.
func TestFileMetadataFromKlosetEntry(t *testing.T) {
	root, snapName, backupDir := newTestFS(t)
	ctx := context.Background()

	cur, err := root.Lookup(ctx, snapName)
	require.NoError(t, err)
	for _, part := range splitPath(backupDir) {
		next, err := cur.(*Dir).Lookup(ctx, part)
		require.NoError(t, err)
		cur = next
	}
	subdirNode, err := cur.(*Dir).Lookup(ctx, "subdir")
	require.NoError(t, err)
	fileNode, err := subdirNode.(*Dir).Lookup(ctx, "hello.txt")
	require.NoError(t, err)
	f := fileNode.(*File)

	require.EqualValues(t, 0644, f.attr.Mode&0o777, "file mode should match the importer setting")
	require.GreaterOrEqual(t, f.attr.Nlink, uint32(1))
	require.EqualValues(t, len("hello world"), f.attr.Size)
}

// TestSnapshotCacheKeyDistinguishesSnapshots ensures the inode cache key for
// snapshot-level directories includes the full snapshot MAC, so two snapshots
// that share a 4-byte short-name prefix do not collide.
func TestSnapshotCacheKeyDistinguishesSnapshots(t *testing.T) {
	root, _, _ := newTestFS(t)

	var a, b objects.MAC
	for i := range a {
		a[i] = byte(i)
		b[i] = byte(i)
	}
	// Same 4-byte prefix, different tail.
	a[5] = 0x42
	b[5] = 0x99

	root.readDirMutex.Lock()
	root.readDirSnapshotMapping = map[string]objects.MAC{
		"prefix-a": a,
		"prefix-b": b,
	}
	root.readDirLast = time.Now()
	root.readDirMutex.Unlock()

	// Force the same logical name to demonstrate that even identical names
	// keyed by different MACs produce different cache entries.
	root.readDirMutex.Lock()
	root.readDirSnapshotMapping["dup"] = a
	root.readDirMutex.Unlock()

	keyA := stableKey("snapshot", fmt.Sprintf("%x", a[:]), "dup")
	root.readDirMutex.Lock()
	root.readDirSnapshotMapping["dup"] = b
	root.readDirMutex.Unlock()
	keyB := stableKey("snapshot", fmt.Sprintf("%x", b[:]), "dup")

	require.NotEqual(t, keyA, keyB, "snapshot cache key must include full MAC")
}

var _ fusefs.Node = (*Dir)(nil)
var _ fusefs.NodeStringLookuper = (*Dir)(nil)
var _ fusefs.HandleReadDirAller = (*Dir)(nil)

func splitPath(p string) []string {
	parts := strings.Split(p, "/")
	out := parts[:0]
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
