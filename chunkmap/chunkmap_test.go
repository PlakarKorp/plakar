package chunkmap

import (
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/stretchr/testify/require"
)

// mac builds a distinct MAC from a single byte, enough to key chunks in tests.
func mac(b byte) objects.MAC {
	var m objects.MAC
	m[0] = b
	return m
}

func chunk(b byte) objects.Chunk {
	return objects.Chunk{ContentMAC: mac(b)}
}

func TestComputeAllUnique(t *testing.T) {
	res := Compute([]File{
		{Label: "a", Chunks: []objects.Chunk{chunk(1), chunk(2)}},
		{Label: "b", Chunks: []objects.Chunk{chunk(3), chunk(4)}},
	})

	require.Equal(t, 2, res.Total)
	require.Len(t, res.Files, 2)
	for _, f := range res.Files {
		require.Equal(t, Stats{NChunks: 2, Unique: 2}, f.Stats)
		for _, c := range f.Chunks {
			require.Equal(t, 1, c.ShareCount)
		}
	}
}

func TestComputeAllFullyShared(t *testing.T) {
	res := Compute([]File{
		{Label: "a", Chunks: []objects.Chunk{chunk(1), chunk(2)}},
		{Label: "b", Chunks: []objects.Chunk{chunk(1), chunk(2)}},
	})

	require.Equal(t, 2, res.Total)
	for _, f := range res.Files {
		require.Equal(t, Stats{NChunks: 2, FullyShared: 2}, f.Stats)
		for _, c := range f.Chunks {
			require.Equal(t, 2, c.ShareCount)
		}
	}
}

func TestComputeMixed(t *testing.T) {
	// A in all 3 files, B in 2 of 3, C in 1.
	res := Compute([]File{
		{Label: "a", Chunks: []objects.Chunk{chunk('A'), chunk('B'), chunk('C')}},
		{Label: "b", Chunks: []objects.Chunk{chunk('A'), chunk('B')}},
		{Label: "c", Chunks: []objects.Chunk{chunk('A')}},
	})

	require.Equal(t, 3, res.Total)
	require.Equal(t, Stats{NChunks: 3, FullyShared: 1, PartiallyShared: 1, Unique: 1}, res.Files[0].Stats)
	require.Equal(t, Stats{NChunks: 2, FullyShared: 1, PartiallyShared: 1}, res.Files[1].Stats)
	require.Equal(t, Stats{NChunks: 1, FullyShared: 1}, res.Files[2].Stats)

	// Per-chunk share counts and index ordering.
	require.Equal(t, []int{3, 2, 1}, shareCounts(res.Files[0]))
	require.Equal(t, []int{3, 2}, shareCounts(res.Files[1]))
	require.Equal(t, []int{3}, shareCounts(res.Files[2]))
	for _, f := range res.Files {
		for i, c := range f.Chunks {
			require.Equal(t, i, c.Index)
		}
	}
}

func TestComputeSingleFileIsFullyShared(t *testing.T) {
	// With one file, every chunk is present in all (one) compared files.
	res := Compute([]File{
		{Label: "solo", Chunks: []objects.Chunk{chunk(1), chunk(2), chunk(3)}},
	})

	require.Equal(t, 1, res.Total)
	require.Equal(t, Stats{NChunks: 3, FullyShared: 3}, res.Files[0].Stats)
	for _, c := range res.Files[0].Chunks {
		require.Equal(t, 1, c.ShareCount)
	}
}

func TestComputeRepeatedMACCountedOncePerFile(t *testing.T) {
	// The same MAC appears twice within file "a" but must count as one file.
	res := Compute([]File{
		{Label: "a", Chunks: []objects.Chunk{chunk(1), chunk(1)}},
		{Label: "b", Chunks: []objects.Chunk{chunk(1)}},
	})

	require.Equal(t, 2, res.Total)
	// Share count is 2 (both files), not 3 (total chunk occurrences).
	require.Equal(t, Stats{NChunks: 2, FullyShared: 2}, res.Files[0].Stats)
	require.Equal(t, Stats{NChunks: 1, FullyShared: 1}, res.Files[1].Stats)
	for _, c := range res.Files[0].Chunks {
		require.Equal(t, 2, c.ShareCount)
	}
}

func TestComputeEmpty(t *testing.T) {
	res := Compute(nil)
	require.Equal(t, 0, res.Total)
	require.Empty(t, res.Files)
}

func TestShareRatio(t *testing.T) {
	// total <= 1 short-circuits to 1.0.
	require.Equal(t, 1.0, ShareRatio(1, 1))
	require.Equal(t, 1.0, ShareRatio(0, 0))

	// unique -> 0, all -> 1, midpoint -> 0.5.
	require.Equal(t, 0.0, ShareRatio(1, 3))
	require.Equal(t, 1.0, ShareRatio(3, 3))
	require.Equal(t, 0.5, ShareRatio(2, 3))
}

func shareCounts(f FileShare) []int {
	out := make([]int, len(f.Chunks))
	for i, c := range f.Chunks {
		out[i] = c.ShareCount
	}
	return out
}
