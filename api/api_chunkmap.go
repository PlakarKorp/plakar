package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PlakarKorp/plakar/chunkmap"
)

// maxChunkmapPaths bounds how many snapshot:path pairs a single request may
// compare. Each pair costs a snapshot-prefix lookup plus a filesystem walk and
// holds its chunk list in memory, so the count must be capped to keep an
// authenticated caller from amplifying one request into arbitrary work. 64 is
// generous for the real use case (one file across a retention window).
const maxChunkmapPaths = 64

// snapshotChunkmap compares one file across several snapshots and reports, per
// chunk, how many of the requested files share it. It is the API analogue of
// the `diag chunkmap` command: the snapshot:path pairs are passed as repeated
// `path` query parameters. The share ratio and its colour are left to the
// frontend: for total > 1, ratio = (share_count-1)/(total-1); for total <= 1,
// ratio is 1.0 (everything is shared with itself). total is the top-level item
// count. This mirrors chunkmap.ShareRatio.
func (ui *uiserver) snapshotChunkmap(w http.ResponseWriter, r *http.Request) error {
	raws := QueryParamToStrings(r, "path")
	if len(raws) == 0 {
		return parameterError("path", MissingArgument, ErrMissingField)
	}
	if len(raws) > maxChunkmapPaths {
		return parameterError("path", InvalidArgument,
			fmt.Errorf("too many paths, at most %d may be compared", maxChunkmapPaths))
	}

	files := make([]chunkmap.File, 0, len(raws))
	for _, raw := range raws {
		mac, path, err := resolveSnapshotPath(ui.repository, raw, "path")
		if err != nil {
			return err
		}

		snap, err := loadsnap(ui.repository, mac)
		if err != nil {
			return err
		}

		fs, err := snap.Filesystem()
		if err != nil {
			return err
		}

		entry, err := fs.GetEntry(path)
		if err != nil {
			return fmt.Errorf("chunkmap: %q: %w", raw, err)
		}

		if entry.ResolvedObject == nil {
			return parameterError("path", InvalidArgument,
				fmt.Errorf("%q has no content (directory or empty object)", raw))
		}

		files = append(files, chunkmap.File{
			Label:  raw,
			Chunks: entry.ResolvedObject.Chunks,
		})
	}

	res := chunkmap.Compute(files)
	return json.NewEncoder(w).Encode(Items[chunkmap.FileShare]{Total: res.Total, Items: res.Files})
}
