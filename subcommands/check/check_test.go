package check

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap, ctx
}

func TestExecuteCmdCheckDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{}

	subcommand := &Check{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should be something like:
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/another_subdir/bar
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/another_subdir
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/subdir/dummy.txt
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/subdir/foo.txt
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/subdir/to_exclude
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482/subdir
	// 2025-02-26T20:32:53Z info: 2dd0bbc2: ✓ /tmp/tmp_to_backup2103239482
	// 2025-02-26T20:32:53Z info: check: verification of 2dd0bbc2:/ completed successfully

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 8, len(lines))

	// last line should have the summary
	lastline := lines[len(lines)-1]
	require.Contains(t, lastline, fmt.Sprintf("info: check: verification of %s:%s completed successfully", hex.EncodeToString(snap.Header.GetIndexShortID()[:]), snap.Header.GetSource(0).Importer.Directory))
}

func TestExecuteCmdCheckSpecificSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// create one snapshot
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

	subcommand := &Check{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should be something like:
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/another_subdir/bar
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/another_subdir
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/subdir/dummy.txt
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/subdir/foo.txt
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/subdir/to_exclude
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417/subdir
	// 2025-02-26T20:36:32Z info: c7b3aef6: ✓ /tmp/tmp_to_backup3511851417
	// 2025-02-26T20:36:32Z info: check: verification of c7b3aef6:/tmp/tmp_to_backup3511851417 completed successfully

	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 8, len(lines))
	// last line should have the summary
	lastline := lines[len(lines)-1]
	require.Contains(t, lastline, fmt.Sprintf("info: check: verification of %s:%s completed successfully", hex.EncodeToString(snap.Header.GetIndexShortID()[:]), snap.Header.GetSource(0).Importer.Directory))
}
