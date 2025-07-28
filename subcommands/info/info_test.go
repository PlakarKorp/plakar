package info

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	_ "github.com/PlakarKorp/plakar/connectors/fs/exporter"
	"github.com/PlakarKorp/plakar/subcommands"
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

func TestExecuteCmdInfoDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()
	args := []string{"info"}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 1.0.0
	// Timestamp: 2025-03-05 21:48:39.742132699 +0000 UTC
	// RepositoryID: 79650133-b57c-46a9-aff9-7dcaf7829033
	// Packfile:
	// - MaxSize: 21 MB (20971520 bytes)
	// Chunking:
	// - Algorithm: FASTCDC
	// - MinSize: 66 kB (65536 bytes)
	// - NormalSize: 1.0 MB (1048576 bytes)
	// - MaxSize: 4.2 MB (4194304 bytes)
	// Hashing:
	// - Algorithm: BLAKE3
	// - Bits: 256
	// Compression:
	// - Algorithm: LZ4
	// - Level: 131072
	// Encryption:
	// - SubkeyAlgorithm: AES256-KW
	// - DataAlgorithm: AES256-GCM-SIV
	// - ChunkSize: 65536
	// - Canary:
	// - KDF: ARGON2ID
	// - Salt: 1f9d5fbf813e81066d863c77d2093612
	// - SaltSize: 16
	// - KeyLen: 32
	// - Time: 4
	// - Memory: 262144
	// - Thread: 1
	// Snapshots: 1
	// Size: 49 B (49 bytes)

	output := bufOut.String()
	require.Contains(t, output, "Snapshots: 1")
}

func TestExecuteCmdInfoSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"info", hex.EncodeToString(indexId[:])}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 1.0.0
	// SnapshotID: 9ed843b00c92b6a69eb68b1f691032587a31e102fab7c82b64ad2589585883f1
	// Timestamp: 2025-03-05 21:49:22.875334881 +0000 UTC
	// Duration: 5.000531ms
	// Name: test_backup
	// Environment: default
	// Perimeter: default
	// Category: default
	// VFS: {649537a22c67f1b29bb4f59632e104653c3a01088605ad7fc25af5fc96b6e7cb d358b0f4fcab617140ff49cd0bec5f8ff3c35deca65f4754a320245f50238e00 d358b0f4fcab617140ff49cd0bec5f8ff3c35deca65f4754a320245f50238e00}
	// Importer:
	// - Type: fs
	// - Origin: grumpf
	// - Directory: /tmp/tmp_to_backup3285582724
	// Context:
	// - MachineID:
	// - Hostname:
	// - Username:
	// - OperatingSystem:
	// - Architecture:
	// - NumCPU: 16
	// - GOMAXPROCS:
	// - ProcessID: 0
	// - Client:
	// - CommandLine:
	// Summary:
	// - Directories: 0
	// - Files: 4
	// - Symlinks: 0
	// - Devices: 0
	// - Pipes: 0
	// - Sockets: 0
	// - Setuid: 0
	// - Setgid: 0
	// - Sticky: 0
	// - Objects: 4
	// - Chunks: 4
	// - MinSize: 0 B (0 bytes)
	// - MaxSize: 20 B (20 bytes)
	// - Size: 49 B (49 bytes)
	// - MinModTime: 1970-01-01 00:00:00 +0000 UTC
	// - MaxModTime: 2025-03-05 21:49:22 +0000 UTC
	// - MinEntropy: 0.000000
	// - MaxEntropy: 3.921928
	// - HiEntropy: 0
	// - LoEntropy: 0
	// - MIMEAudio: 0
	// - MIMEVideo: 0
	// - MIMEImage: 0
	// - MIMEText: 4
	// - MIMEApplication: 0
	// - MIMEOther: 0
	// - Errors: 0

	output := bufOut.String()
	require.Contains(t, output, "Name: test_backup")
	//require.Contains(t, output, "Files: 4")
	require.Contains(t, output, fmt.Sprintf("Directory: %s", snap.Header.GetSource(0).Importer.Directory))
	require.Contains(t, output, fmt.Sprintf("SnapshotID: %s", hex.EncodeToString(indexId[:])))
}
