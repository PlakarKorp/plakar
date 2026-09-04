package diag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/chunkmap"
	"github.com/stretchr/testify/require"
)

func macFromByte(b byte) objects.MAC {
	var m objects.MAC
	m[0] = b
	return m
}

// TestWriteChunkmapHTML pins the HTML rendering so the shared-package refactor
// (and future edits) keep the byte format the `--html` output had: the template
// resolves $.Total to the file count and macHex to the chunk's ContentMAC.
func TestWriteChunkmapHTML(t *testing.T) {
	shared := objects.Chunk{ContentMAC: macFromByte(0xAB)} // in both files
	uniqA := objects.Chunk{ContentMAC: macFromByte(0x01)}  // only in file a
	uniqB := objects.Chunk{ContentMAC: macFromByte(0x02)}  // only in file b

	res := chunkmap.Compute([]chunkmap.File{
		{Label: "aaaa:plakar", Chunks: []objects.Chunk{shared, uniqA}},
		{Label: "bbbb:plakar", Chunks: []objects.Chunk{shared, uniqB}},
	})

	path := filepath.Join(t.TempDir(), "out.html")
	code, err := writeChunkmapHTML(path, res)
	require.NoError(t, err)
	require.Equal(t, 0, code)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	html := string(data)

	sharedHex := "ab" + strings.Repeat("0", 62)
	uniqAHex := "01" + strings.Repeat("0", 62)

	// $.Total resolves to the file count (2); tooltip and macHex are correct.
	require.Contains(t, html, "chunk 0: 2/2 files, "+sharedHex)
	require.Contains(t, html, "chunk 1: 1/2 files, "+uniqAHex)
	// Colour: fully shared -> ratio 1.0 -> hsl 120.0; unique -> ratio 0.0 -> 0.0.
	require.Contains(t, html, "hsl(120.0,70%,45%)")
	require.Contains(t, html, "hsl(0.0,70%,45%)")
	// Per-file summary line and label.
	require.Contains(t, html, "aaaa:plakar")
	require.Contains(t, html, "2 chunks &mdash; 1 in all files, 0 partial, 1 unique")
}
