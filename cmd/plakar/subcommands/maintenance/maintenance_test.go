package maintenance

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/PlakarKorp/plakar/snapshot"
	_ "github.com/PlakarKorp/plakar/snapshot/exporter/fs"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) *snapshot.Snapshot {
	return ptesting.GenerateSnapshot(t, bufOut, bufErr, nil, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
}

func TestExecuteCmdMaintenanceDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	snap := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	ctx := snap.AppContext()
	ctx.MaxConcurrency = 1

	repo := snap.Repository()
	// override the homedir to avoid having test overwriting existing home configuration
	ctx.HomeDir = repo.Location()
	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

	subcommand, err := parse_cmd_maintenance(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)
	require.Equal(t, "maintenance", subcommand.(*Maintenance).Name())

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this

	output := bufOut.String()
	// this is unstable due to randomness of backup
	require.Contains(t, output, "maintenance: Coloured 0 packfiles (0 orphaned) for deletion")
	require.Contains(t, output, "maintenance: 0 blobs and 0 packfiles were removed")
}
