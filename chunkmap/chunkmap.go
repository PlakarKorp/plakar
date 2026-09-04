// Package chunkmap computes how the chunks of a file are shared across several
// snapshots: for each chunk, how many of the compared files contain it. It is
// the shared core behind the `diag chunkmap` command and the chunkmap API
// endpoint, so both agree on what "fully shared", "partially shared" and
// "unique" mean for the same repository.
package chunkmap

import "github.com/PlakarKorp/kloset/objects"

// File is one labeled file's chunk sequence, the input to Compute.
type File struct {
	Label  string
	Chunks []objects.Chunk
}

// ChunkShare reports, for a single chunk, how many of the compared files share
// it. Index is the chunk's position within its file; ShareCount counts each
// file once (a chunk repeated within one file still counts as that one file).
type ChunkShare struct {
	Index      int         `json:"index"`
	ShareCount int         `json:"share_count"`
	ContentMAC objects.MAC `json:"content_mac"`
}

// Stats summarizes one file's chunks by sharing bucket.
type Stats struct {
	NChunks         int `json:"n_chunks"`
	FullyShared     int `json:"fully_shared"`
	PartiallyShared int `json:"partially_shared"`
	Unique          int `json:"unique"`
}

// FileShare is the per-file result: its label, bucket counts, and per-chunk
// share data in original order.
type FileShare struct {
	Label  string       `json:"label"`
	Stats  Stats        `json:"stats"`
	Chunks []ChunkShare `json:"chunks"`
}

// Result is the outcome of comparing Total files.
type Result struct {
	Total int
	Files []FileShare
}

// Compute counts, for every distinct chunk MAC, how many of the given files
// contain it (once per file), then classifies each file's chunks as fully
// shared (present in all files), unique (present in exactly one), or partially
// shared. With a single file every chunk is reported as fully shared, since it
// is present in all (one) of the compared files.
func Compute(files []File) Result {
	total := len(files)

	shareCount := make(map[objects.MAC]int)
	for _, f := range files {
		seen := make(map[objects.MAC]struct{})
		for _, chunk := range f.Chunks {
			if _, ok := seen[chunk.ContentMAC]; !ok {
				seen[chunk.ContentMAC] = struct{}{}
				shareCount[chunk.ContentMAC]++
			}
		}
	}

	res := Result{Total: total, Files: make([]FileShare, 0, total)}
	for _, f := range files {
		fs := FileShare{
			Label:  f.Label,
			Chunks: make([]ChunkShare, len(f.Chunks)),
		}
		fs.Stats.NChunks = len(f.Chunks)
		for i, chunk := range f.Chunks {
			sc := shareCount[chunk.ContentMAC]
			// Order matters: when total == 1, `case total` wins over `case 1`,
			// so a lone file's chunks are fully shared, not unique.
			switch sc {
			case total:
				fs.Stats.FullyShared++
			case 1:
				fs.Stats.Unique++
			default:
				fs.Stats.PartiallyShared++
			}
			fs.Chunks[i] = ChunkShare{
				Index:      i,
				ShareCount: sc,
				ContentMAC: chunk.ContentMAC,
			}
		}
		res.Files = append(res.Files, fs)
	}
	return res
}

// ShareRatio maps a chunk's share count to 0.0 (unique to its file) .. 1.0
// (present in all files). shareCount includes the file itself; total is the
// number of files compared.
func ShareRatio(shareCount, total int) float64 {
	if total <= 1 {
		return 1.0
	}
	return float64(shareCount-1) / float64(total-1)
}
