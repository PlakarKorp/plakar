package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PlakarKorp/plakar/chunkmap"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestSnapshotChunkmap(t *testing.T) {
	mux, repo, snap1, _ := server(t, "")
	defer snap1.Close()

	// A second snapshot with identical content: the shared files hash to the
	// same chunk MACs, so they show up as fully shared across the two.
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("top.txt", 0644, "top level"),
	})
	defer snap2.Close()

	id1 := snapid(snap1)
	id2 := snapid(snap2)

	t.Run("fully shared across snapshots", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path="+id1+":/top.txt&path="+id2+":/top.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// Wire format the frontend depends on: snake_case keys, hex MAC.
		body := w.Body.String()
		require.Contains(t, body, `"share_count"`)
		require.Contains(t, body, `"content_mac"`)
		require.Contains(t, body, `"fully_shared"`)

		var resp Items[chunkmap.FileShare]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 2, resp.Total)
		require.Len(t, resp.Items, 2)

		for _, f := range resp.Items {
			require.Greater(t, f.Stats.NChunks, 0)
			require.Equal(t, f.Stats.NChunks, f.Stats.FullyShared)
			require.Equal(t, 0, f.Stats.Unique)
			require.Equal(t, 0, f.Stats.PartiallyShared)
			require.Len(t, f.Chunks, f.Stats.NChunks)
			for _, c := range f.Chunks {
				require.Equal(t, 2, c.ShareCount)
				require.Len(t, c.ContentMAC.FormatHex(), 64)
			}
		}
	})

	t.Run("all unique across distinct files", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path="+id1+":/top.txt&path="+id1+":/subdir/dummy.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp Items[chunkmap.FileShare]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 2, resp.Total)
		require.Len(t, resp.Items, 2)

		for _, f := range resp.Items {
			require.Greater(t, f.Stats.NChunks, 0)
			require.Equal(t, f.Stats.NChunks, f.Stats.Unique)
			require.Equal(t, 0, f.Stats.FullyShared)
			for _, c := range f.Chunks {
				require.Equal(t, 1, c.ShareCount)
			}
		}
	})

	t.Run("single path is fully shared with itself", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path="+id1+":/top.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp Items[chunkmap.FileShare]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 1, resp.Total)
		require.Len(t, resp.Items, 1)
		require.Equal(t, id1+":/top.txt", resp.Items[0].Label)
		require.Equal(t, resp.Items[0].Stats.NChunks, resp.Items[0].Stats.FullyShared)
	})

	t.Run("missing path parameter", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("too many paths", func(t *testing.T) {
		url := "/api/snapshot/chunkmap?path=" + id1 + ":/top.txt"
		for i := 0; i < maxChunkmapPaths; i++ {
			url += "&path=" + id1 + ":/top.txt"
		}
		w := get(t, mux, url)
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("bad snapshot prefix", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path=ffffffff:/top.txt")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})

	t.Run("nonexistent path", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path="+id1+":/no-such-file.txt")
		require.Equal(t, http.StatusNotFound, w.Code, "body=%s", w.Body.String())
	})

	t.Run("directory has no content", func(t *testing.T) {
		w := get(t, mux, "/api/snapshot/chunkmap?path="+id1+":/subdir")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
	})
}
