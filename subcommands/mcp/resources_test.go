package mcp

import (
	"bytes"
	"encoding/hex"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestReadResource(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	data, _, err := readResource(repo, snapID, "/subdir/dummy.txt", 1<<20)
	require.NoError(t, err)
	require.Equal(t, "hello dummy", string(data))

	// unlike the plakar tool, a resource cannot say it was cut: too large is an error
	_, _, err = readResource(repo, snapID, "/subdir/dummy.txt", 4)
	require.Error(t, err)

	_, _, err = readResource(repo, snapID, "/subdir/missing.txt", 1<<20)
	require.Error(t, err)

	_, _, err = readResource(repo, snapID, "/subdir", 1<<20)
	require.Error(t, err, "a directory is not a resource")
}
