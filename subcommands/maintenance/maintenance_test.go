package maintenance

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdMaintenanceDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("/"),
		// ptesting.NewMockDir("subdir"),
		//ptesting.NewMockDir("another_subdir"),
		// ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		// ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		// ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		//ptesting.NewMockFile("bar.txt", 0644, ""),
	})

	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

	subcommand := &Maintenance{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this

	output := bufOut.String()
	log.Println("the output is", output)
	// this is unstable due to randomness of backup
	require.Contains(t, output, "maintenance: Coloured 0 packfiles (0 orphaned) for deletion")
	require.Contains(t, output, "maintenance: 0 blobs and 0 packfiles were removed")
}
