//go:build linux

package mount

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestExecuteCmdMountDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	defer snap.Close()

	tmpMountPoint, err := os.MkdirTemp("", "tmp_mount_point")
	require.NoError(t, err)
	defer os.RemoveAll(tmpMountPoint)

	args := []string{tmpMountPoint}

	subcommand := &Mount{}
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	go func() {
		status, err := subcommand.Execute(ctx, repo)
		require.NoError(t, err)
		require.Equal(t, 0, status)
	}()

	time.Sleep(300 * time.Millisecond)

	file, err := os.Stat(tmpMountPoint)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "drwx------", file.Mode().String())

	// output should look like this
	// 2025-03-19T23:04:15Z info: mounted repository /tmp/tmp_repo2787767309/repo at /tmp/tmp_mount_point2239236580
	output := bufOut.String()
	location, err := repo.Location()
	require.NoError(t, err)
	require.Contains(t, output, fmt.Sprintf("mounted repository %s at %s", location, tmpMountPoint))

	indexId := snap.Header.GetIndexID()
	snapshotPath := fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))
	backupDir := snap.Header.GetSource(0).Importer.Directory

	dummyMountedPath := fmt.Sprintf("%s/%s/%s/subdir/dummy.txt", tmpMountPoint, snapshotPath, backupDir)
	file, err = os.Stat(dummyMountedPath)
	require.NoError(t, err)
	require.NotNil(t, file)

	dummyFile, err := os.Open(dummyMountedPath)
	require.NoError(t, err)
	defer dummyFile.Close()
	content, err := io.ReadAll(dummyFile)
	require.NoError(t, err)
	require.Equal(t, "hello dummy", string(content))
}
