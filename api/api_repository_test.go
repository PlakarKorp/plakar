package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/resources"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// TestRepositoryEmpty checks some basic things on an empty kloset.
func TestRepositoryEmpty(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	mux := http.NewServeMux()
	SetupRoutes(mux, repo, ctx, "", true)

	t.Run("empty repository serves 200", func(t *testing.T) {
		for _, path := range []string{
			"/api/repository/snapshots",
			"/api/repository/importer-types",
			"/api/repository/sources",
			"/api/repository/locate-pathname",
			"/api/repository/info",
		} {
			w := get(t, mux, path)
			require.Equal(t, http.StatusOK, w.Code, "path=%s body=%s", path, w.Body.String())
		}
	})

	t.Run("info empty efficiency", func(t *testing.T) {
		// A repository with no snapshots has logicalSize == 0, which drives the
		// efficiency = -1 branch in repositoryInfo.
		w := get(t, mux, "/api/repository/info")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp Item[RepositoryInfoResponse]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, float64(-1), resp.Item.Snapshots.Efficiency)
		require.Equal(t, 0, resp.Item.Snapshots.Total)
		require.Len(t, resp.Item.Snapshots.SnapshotsPerDay, 30)
	})
}

// TestRepository drives the repository endpoints against a single populated
// repository.
func TestRepository(t *testing.T) {
	mux, _, snap, _ := server(t, "")
	defer snap.Close()

	t.Run("info", func(t *testing.T) {
		// The populated-efficiency branch (logicalSize > 0).
		w := get(t, mux, "/api/repository/info")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var resp Item[RepositoryInfoResponse]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.GreaterOrEqual(t, resp.Item.Snapshots.Total, 1)
		require.NotEmpty(t, resp.Item.OS)
		require.NotEmpty(t, resp.Item.Arch)
		require.True(t, resp.Item.Browsable)
	})

	t.Run("importer types", func(t *testing.T) {
		w := get(t, mux, "/api/repository/importer-types")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[map[string]string]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.Equal(t, items.Total, len(items.Items))
	})

	t.Run("sources", func(t *testing.T) {
		repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
		snap := ptesting.GenerateSnapshot(t, repo, nil)
		defer snap.Close()
		secondSnap := ptesting.GenerateSnapshot(t, repo, nil, ptesting.WithName("second"))
		defer secondSnap.Close()
		baseSource := snap.Header.Sources[0]
		sortByType := baseSource
		sortByType.Importer.Type = "a"
		sortByType.Importer.Origin = "z"
		sortByType.Importer.Directory = "z"
		sortByOrigin := sortByType
		sortByOrigin.Importer.Origin = "a"
		sortByDirectory := sortByOrigin
		sortByDirectory.Importer.Directory = "a"
		snap.Header.Sources = append(snap.Header.Sources, sortByType, sortByOrigin, sortByDirectory)
		serializedHeader, err := snap.Header.Serialize()
		require.NoError(t, err)
		cache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
		require.NoError(t, err)
		require.NoError(t, cache.PutSnapshot(snap.Header.Identifier, serializedHeader))

		mux := http.NewServeMux()
		SetupRoutes(mux, repo, ctx, "", true)

		w := get(t, mux, "/api/repository/sources")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		var items Items[RepositorySource]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.Equal(t, []RepositorySource{
			{Type: "a", Origin: "a", Directory: "a"},
			{Type: "a", Origin: "a", Directory: "z"},
			{Type: "a", Origin: "z", Directory: "z"},
			{Type: baseSource.Importer.Type, Origin: baseSource.Importer.Origin, Directory: baseSource.Importer.Directory},
		}, items.Items)
		require.Equal(t, len(items.Items), items.Total)
	})

	t.Run("sources refresh error", func(t *testing.T) {
		repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
		mux := http.NewServeMux()
		SetupRoutes(mux, repo, ctx, "", false)

		w := get(t, mux, "/api/repository/sources")
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	})

	t.Run("sources snapshot load error", func(t *testing.T) {
		repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
		snap := ptesting.GenerateSnapshot(t, repo, nil)
		packfiles, err := snap.ListPackfiles()
		require.NoError(t, err)
		packfile, err := func() (objects.MAC, error) {
			for packfile, err := range packfiles {
				return packfile, err
			}
			return objects.MAC{}, nil
		}()
		require.NoError(t, err)
		require.NotEqual(t, objects.MAC{}, packfile)
		cache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
		require.NoError(t, err)
		require.NoError(t, cache.DelSnapshot(snap.Header.Identifier))
		require.NoError(t, repo.Store().Delete(repo.AppContext(), storage.StorageResourcePackfile, packfile))
		require.NoError(t, snap.Close())

		mux := http.NewServeMux()
		SetupRoutes(mux, repo, ctx, "", true)
		w := get(t, mux, "/api/repository/sources")
		require.Equal(t, http.StatusInternalServerError, w.Code, "body=%s", w.Body.String())
	})

	t.Run("sources snapshot list error", func(t *testing.T) {
		repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
		snap := ptesting.GenerateSnapshot(t, repo, nil)
		require.NoError(t, snap.Close())
		stateCache, err := caching.NewSQLState(filepath.Join(ctx.CacheDir, caching.CACHE_VERSION, "store", repo.Configuration().RepositoryID.String()), false)
		require.NoError(t, err)
		require.NoError(t, stateCache.PutDelta(resources.RT_SNAPSHOT, objects.MAC{1}, objects.MAC{2}, []byte("invalid delta")))

		mux := http.NewServeMux()
		SetupRoutes(mux, repo, ctx, "", true)
		w := get(t, mux, "/api/repository/sources")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), `"total":1`)
	})

	t.Run("snapshots", func(t *testing.T) {
		w := get(t, mux, "/api/repository/snapshots")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")
	})

	t.Run("snapshots exact window", func(t *testing.T) {
		// limit=1 with a single snapshot drives offset+limit == len(headers), the
		// exact-window slice branch of repositorySnapshots.
		w := get(t, mux, "/api/repository/snapshots?limit=1&offset=0")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.GreaterOrEqual(t, items.Total, 1)
	})

	t.Run("snapshots importer match", func(t *testing.T) {
		// First find the actual importer type via the header.
		importer := snap.Header.GetSource(0).Importer.Type

		w := get(t, mux, "/api/repository/snapshots?importer="+importer)
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.GreaterOrEqual(t, items.Total, 1)

		// since in the past -> snapshot is kept.
		w = get(t, mux, "/api/repository/snapshots?since=2000-01-01T00:00:00Z")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("snapshots filters", func(t *testing.T) {
		// importer filter that matches nothing -> total counts only matching ones.
		w := get(t, mux, "/api/repository/snapshots?importer=nonexistent")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// since in the future -> no matching snapshots but still 200.
		w = get(t, mux, "/api/repository/snapshots?since=2999-01-01T00:00:00Z")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// descending sort.
		w = get(t, mux, "/api/repository/snapshots?sort=-Timestamp")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// offset beyond the number of headers -> empty items.
		w = get(t, mux, "/api/repository/snapshots?offset=1000")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// invalid sort key -> 400.
		w = get(t, mux, "/api/repository/snapshots?sort=Bogus")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// invalid offset -> error.
		w = get(t, mux, "/api/repository/snapshots?offset=abc")
		require.NotEqual(t, http.StatusOK, w.Code)
	})

	// The parameter-validation errors in repositorySnapshots short-circuit
	// before any storage access.
	t.Run("snapshots param errors", func(t *testing.T) {
		for _, c := range []struct {
			name  string
			query string
		}{
			{"bad offset", "offset=abc"},
			{"bad limit", "limit=abc"},
			{"bad since", "since=not-a-date"},
			{"bad sort", "sort=NoSuchKey"},
		} {
			t.Run(c.name, func(t *testing.T) {
				w := get(t, mux, "/api/repository/snapshots?"+c.query)
				require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())
			})
		}
	})

	t.Run("locate-pathname exact window", func(t *testing.T) {
		// A resource that resolves in the single snapshot. With limit=1 and offset=0,
		// offset+limit == len(locations), exercising the exact-window slice branch
		// (locations[offset:offset+limit]) rather than the tail branch.
		w := get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&limit=1&offset=0")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		var items Items[json.RawMessage]
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &items))
		require.GreaterOrEqual(t, items.Total, 1)
	})

	t.Run("locate-pathname default sort asc", func(t *testing.T) {
		// No explicit sort -> default ascending Timestamp sortFunc branch.
		w := get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")
	})

	t.Run("locate-pathname filters", func(t *testing.T) {
		// Resource that exists in the snapshot.
		w := get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&sort=-Timestamp")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
		require.Contains(t, w.Body.String(), "total")

		// importerType filter that matches nothing.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&importerType=nope")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// importerOrigin filter that matches nothing.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&importerOrigin=nope")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// importerDirectory filter that matches nothing.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&importerDirectory=nope")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// Resource that does not resolve in any snapshot -> empty result, 200.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/no/such/file")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())

		// invalid sort key -> 400.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&sort=Bogus")
		require.Equal(t, http.StatusBadRequest, w.Code, "body=%s", w.Body.String())

		// offset beyond results.
		w = get(t, mux, "/api/repository/locate-pathname?resource=/subdir/dummy.txt&offset=1000")
		require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	})

	t.Run("locate-pathname param errors", func(t *testing.T) {
		for _, q := range []string{"offset=abc", "limit=abc", "sort=NoSuchKey"} {
			t.Run(q, func(t *testing.T) {
				w := get(t, mux, "/api/repository/locate-pathname?"+q)
				require.GreaterOrEqual(t, w.Code, 400, "body=%s", w.Body.String())
			})
		}
	})
}
