package diag

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
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

func TestExecuteCmdDiagSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "snapshot", fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))}

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

func TestExecuteCmdDiagState(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "state"}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// aaf4c5b7b91ba00f5afde31c5a9f721bc78de202a491379339e011b9172db298
	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))

	bufOut.Reset()
	args = []string{"diag", "state", strings.Trim(output, "\n")}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 655.3.6
	// Creation: 2025-03-10 21:40:24.310314347 +0000 UTC
	// State serial: 1914a7f1-5df5-4106-a5f4-930414d5bcbb
	// snapshot f5e898a19e79196642e33ceba3300f1f88a603b251e23d7a0efe731cabf1e3e9 : packfile 26194d2f6db70e21b6b002b67cc23f269e9bfc985e09a9b2822910be1d1d2338, offset 86, length 850
	// chunk 71c5b84ff64199381df2b4d8b480ec97fb4d57c7175bd21a9d166de55a4d0c45 : packfile 36cdb39fa75085ea515c0ab14967ec2675ed38c1dfbf9236a4aa6e9ccd74ea1d, offset 0, length 30
	// chunk 73c173afeab63ba01cda9bb7c79ee5296917708c2265eeb3eaa9292a26ddba6e : packfile dcae477b2d0a20b5250802b4ec5f6bb01bec08591f8c9b24831ec44834fc0ba0, offset 0, length 39
	// chunk e3e615a992db0b0931ba5d5db26a4758c9929ede8687a891e6eb7a28ecebc6a2 : packfile 65bda45599f74f9fecb29253129cf9d4b3759f3a3c75218b52c7eb30ec3ff081, offset 0, length 28
	// chunk eb6b2f5210c19246e75d75027bc731c40fe9cd043ad77acf80713c696e9e9728 : packfile 02536f992422092b2e1fd7c259cab5876cebe1b8abcc68b52a62e7e442bb35b4, offset 0, length 28
	// object 61e110eb808b6a6849a035c8228fdf18acf7ba3a36ee7113832ce745da800442 : packfile 521e651ee403e39ad3c62b8f794213122b10ee377e7e65dab58dcfc6ee919985, offset 0, length 186
	// object 7f3e1828207a96e5d69cceb611f4c1ac6ead112502f623be53f3657ca8c54529 : packfile e26b7a3c29f5e4d97c71998de4e5bed1b2f8b85d79f7fa0691839921262e9d79, offset 0, length 186
	// object cb83b178515d643d50f75a6573381b5b88ac41d5706985411d5205cc25a2830f : packfile d1d11c3082a84fb9a2a749e8f3a16ed107deac5eaa92c06676cf5345dd84fdfc, offset 0, length 186
	// object e32583acc908696e01e4284620ae55b5df8bf2cb392d2a2d0fb33688143aca17 : packfile fb4be0ea29db0e3fae660dc010c4cf7d9c8e481ac0ba21b431a9a0de1683d117, offset 0, length 186
	// file 06cccef655a87a75323d571cb3aa99e3c19cb7d1c1400d12db05d405a0537053 : packfile b62d8aa0e53362104bc3931f558c071d86d94e58bee9f41fd40a0f0967f4f75b, offset 293, length 86

	output = bufOut.String()
	require.Contains(t, output, "Version:")
	require.Contains(t, output, "State serial:")
	require.Contains(t, output, fmt.Sprintf("snapshot %s", fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))))
	require.Contains(t, output, "chunk ")
	require.Contains(t, output, "object ")
	require.Contains(t, output, "file ")
}

func TestExecuteCmdDiagPackfile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "state"}
	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// aaf4c5b7b91ba00f5afde31c5a9f721bc78de202a491379339e011b9172db298
	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))

	bufOut.Reset()
	args = []string{"diag", "state", strings.Trim(output, "\n")}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 655.3.6
	// Creation: 2025-03-10 21:40:24.310314347 +0000 UTC
	// State serial: 1914a7f1-5df5-4106-a5f4-930414d5bcbb
	// snapshot f5e898a19e79196642e33ceba3300f1f88a603b251e23d7a0efe731cabf1e3e9 : packfile 26194d2f6db70e21b6b002b67cc23f269e9bfc985e09a9b2822910be1d1d2338, offset 86, length 850
	// chunk 71c5b84ff64199381df2b4d8b480ec97fb4d57c7175bd21a9d166de55a4d0c45 : packfile 36cdb39fa75085ea515c0ab14967ec2675ed38c1dfbf9236a4aa6e9ccd74ea1d, offset 0, length 30
	// chunk 73c173afeab63ba01cda9bb7c79ee5296917708c2265eeb3eaa9292a26ddba6e : packfile dcae477b2d0a20b5250802b4ec5f6bb01bec08591f8c9b24831ec44834fc0ba0, offset 0, length 39
	// chunk e3e615a992db0b0931ba5d5db26a4758c9929ede8687a891e6eb7a28ecebc6a2 : packfile 65bda45599f74f9fecb29253129cf9d4b3759f3a3c75218b52c7eb30ec3ff081, offset 0, length 28
	// chunk eb6b2f5210c19246e75d75027bc731c40fe9cd043ad77acf80713c696e9e9728 : packfile 02536f992422092b2e1fd7c259cab5876cebe1b8abcc68b52a62e7e442bb35b4, offset 0, length 28
	// object 61e110eb808b6a6849a035c8228fdf18acf7ba3a36ee7113832ce745da800442 : packfile 521e651ee403e39ad3c62b8f794213122b10ee377e7e65dab58dcfc6ee919985, offset 0, length 186
	// object 7f3e1828207a96e5d69cceb611f4c1ac6ead112502f623be53f3657ca8c54529 : packfile e26b7a3c29f5e4d97c71998de4e5bed1b2f8b85d79f7fa0691839921262e9d79, offset 0, length 186
	// object cb83b178515d643d50f75a6573381b5b88ac41d5706985411d5205cc25a2830f : packfile d1d11c3082a84fb9a2a749e8f3a16ed107deac5eaa92c06676cf5345dd84fdfc, offset 0, length 186
	// object e32583acc908696e01e4284620ae55b5df8bf2cb392d2a2d0fb33688143aca17 : packfile fb4be0ea29db0e3fae660dc010c4cf7d9c8e481ac0ba21b431a9a0de1683d117, offset 0, length 186
	// file 06cccef655a87a75323d571cb3aa99e3c19cb7d1c1400d12db05d405a0537053 : packfile b62d8aa0e53362104bc3931f558c071d86d94e58bee9f41fd40a0f0967f4f75b, offset 293, length 86

	output = bufOut.String()
	require.Contains(t, output, "Version:")
	require.Contains(t, output, "State serial:")
	require.Contains(t, output, fmt.Sprintf("snapshot %s", fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))))
	require.Contains(t, output, "chunk ")
	require.Contains(t, output, "object ")
	require.Contains(t, output, "vfs btree ")

	lines = strings.Split(strings.Trim(output, "\n"), "\n")
	var fileline string
	for _, line := range lines {
		if strings.HasPrefix(line, "vfs btree ") {
			fileline = line
			break
		}
	}

	var partFile, partPackfile []byte
	var partOffset, partLength int
	fmt.Println(fileline)
	fmt.Sscanf(fileline, "vfs btree %x : packfile %x, offset %d, length %d", &partFile, &partPackfile, &partOffset, &partLength)

	bufOut.Reset()
	args = []string{"diag", "packfile", hex.EncodeToString(partPackfile)}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 1.0.0
	// Timestamp: 2025-03-10 21:56:35.854685977 +0000 UTC
	// Index MAC: 887b4128edf0f15bcbe9061265a25252339aa36c23e7394ea628664228dbe5c4

	// blob[0]: 1448059cb6850e686ed6104bfa95c9098566563eb694afcd59643969d2343578 0 291 0 vfs entry
	// blob[1]: 5937a8cb73583a2571e9fa3baa117424d6262ee0e9b079f9a940ccb92fb8a66b 291 86 0 vfs btree
	output = bufOut.String()
	require.Contains(t, output, "Index MAC:")
	require.Contains(t, output, "blob[0]:")
	require.Contains(t, output, "blob[1]:")
}

func TestExecuteCmdDiagObject(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "state"}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// aaf4c5b7b91ba00f5afde31c5a9f721bc78de202a491379339e011b9172db298
	output := bufOut.String()
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	require.Equal(t, 1, len(lines))

	bufOut.Reset()
	args = []string{"diag", "state", strings.Trim(output, "\n")}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// Version: 655.3.6
	// Creation: 2025-03-10 21:40:24.310314347 +0000 UTC
	// State serial: 1914a7f1-5df5-4106-a5f4-930414d5bcbb
	// snapshot f5e898a19e79196642e33ceba3300f1f88a603b251e23d7a0efe731cabf1e3e9 : packfile 26194d2f6db70e21b6b002b67cc23f269e9bfc985e09a9b2822910be1d1d2338, offset 86, length 850
	// chunk 71c5b84ff64199381df2b4d8b480ec97fb4d57c7175bd21a9d166de55a4d0c45 : packfile 36cdb39fa75085ea515c0ab14967ec2675ed38c1dfbf9236a4aa6e9ccd74ea1d, offset 0, length 30
	// chunk 73c173afeab63ba01cda9bb7c79ee5296917708c2265eeb3eaa9292a26ddba6e : packfile dcae477b2d0a20b5250802b4ec5f6bb01bec08591f8c9b24831ec44834fc0ba0, offset 0, length 39
	// chunk e3e615a992db0b0931ba5d5db26a4758c9929ede8687a891e6eb7a28ecebc6a2 : packfile 65bda45599f74f9fecb29253129cf9d4b3759f3a3c75218b52c7eb30ec3ff081, offset 0, length 28
	// chunk eb6b2f5210c19246e75d75027bc731c40fe9cd043ad77acf80713c696e9e9728 : packfile 02536f992422092b2e1fd7c259cab5876cebe1b8abcc68b52a62e7e442bb35b4, offset 0, length 28
	// object 61e110eb808b6a6849a035c8228fdf18acf7ba3a36ee7113832ce745da800442 : packfile 521e651ee403e39ad3c62b8f794213122b10ee377e7e65dab58dcfc6ee919985, offset 0, length 186
	// object 7f3e1828207a96e5d69cceb611f4c1ac6ead112502f623be53f3657ca8c54529 : packfile e26b7a3c29f5e4d97c71998de4e5bed1b2f8b85d79f7fa0691839921262e9d79, offset 0, length 186
	// object cb83b178515d643d50f75a6573381b5b88ac41d5706985411d5205cc25a2830f : packfile d1d11c3082a84fb9a2a749e8f3a16ed107deac5eaa92c06676cf5345dd84fdfc, offset 0, length 186
	// object e32583acc908696e01e4284620ae55b5df8bf2cb392d2a2d0fb33688143aca17 : packfile fb4be0ea29db0e3fae660dc010c4cf7d9c8e481ac0ba21b431a9a0de1683d117, offset 0, length 186
	// file 06cccef655a87a75323d571cb3aa99e3c19cb7d1c1400d12db05d405a0537053 : packfile b62d8aa0e53362104bc3931f558c071d86d94e58bee9f41fd40a0f0967f4f75b, offset 293, length 86

	output = bufOut.String()
	require.Contains(t, output, "Version:")
	require.Contains(t, output, "State serial:")
	require.Contains(t, output, fmt.Sprintf("snapshot %s", fmt.Sprintf("%s", hex.EncodeToString(indexId[:]))))
	require.Contains(t, output, "chunk ")
	require.Contains(t, output, "object ")
	require.Contains(t, output, "file ")

	lines = strings.Split(strings.Trim(output, "\n"), "\n")
	var objectLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "object ") {
			objectLine = line
			break
		}
	}
	var partObject, partPackfile []byte
	var partOffset, partLength int
	fmt.Sscanf(objectLine, "object %x : packfile %x, offset %d, length %d", &partObject, &partPackfile, &partOffset, &partLength)

	bufOut.Reset()
	args = []string{"diag", "object", hex.EncodeToString(partObject)}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// object: 096d53564d0216066f0d2aa6b0f6fc159e5e78271e49f5bee0676ed5f229741e
	//  type: text/plain; charset=utf-8
	//  chunks:
	//    MAC: 096d53564d0216066f0d2aa6b0f6fc159e5e78271e49f5bee0676ed5f229741e

	output = bufOut.String()
	require.Contains(t, output, "object: ")
	require.Contains(t, output, "type: text/plain; charset=utf-8")
	require.Contains(t, output, "chunks:")
}

func TestExecuteCmdDiagVFS(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "vfs", fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId[:]))}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// [FileEntry]
	// Version: 65536
	// ParentPath: /tmp/tmp_to_backup3145808633/subdir
	// Name: dummy.txt
	// Size: 11 B (11 bytes)
	// Permissions: -rw-r--r--
	// ModTime: 2025-03-10 22:19:19.177534173 +0000 UTC
	// DeviceID: 64768
	// InodeID: 22415073
	// UserID: 1000
	// GroupID: 1000
	// Username: sayoun
	// Groupname: sayoun
	// NumLinks: 1
	// ExtendedAttributes: []
	// FileAttributes: 0
	// Classification:
	// CustomMetadata: []
	// Tags: []
	// ExtendedAttributes: []

	output := bufOut.String()
	require.Contains(t, output, "[FileEntry]")
	require.Contains(t, output, "ParentPath: /subdir")
	require.Contains(t, output, "Name: dummy.txt")

	bufOut.Reset()
	args = []string{"diag", "vfs", fmt.Sprintf("%s:/subdir", hex.EncodeToString(indexId[:]))}

	subcommand, _, args = subcommands.Lookup(args)
	err = subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err = subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// [DirEntry]
	// Version: 65536
	// ParentPath: /tmp/tmp_to_backup3145808633
	// Name: subdir
	// Size: 4.1 kB (4096 bytes)
	// Permissions: drwxr-xr-x
	// ModTime: 2025-03-10 22:19:19.177534173 +0000 UTC
	// DeviceID: 64768
	// InodeID: 22415071
	// UserID: 1000
	// GroupID: 1000
	// Username: sayoun
	// Groupname: sayoun
	// NumLinks: 2
	// ExtendedAttributes: []
	// FileAttributes: 0
	// Classification:
	// CustomMetadata: []
	// Tags: []
	// ExtendedAttributes: []
	// Below.Directories: 0
	// Below.Files: 0
	// Below.Symlinks: 0
	// Below.Devices: 0
	// Below.Pipes: 0
	// Below.Sockets: 0
	// Below.Setuid: 0
	// Below.Setgid: 0
	// Below.Sticky: 0
	// Below.Objects: 0
	// Below.Chunks: 0
	// Below.MinSize: 0 B (0 bytes)
	// Below.MaxSize: 0 B (0 bytes)
	// Below.Size: 0 B (0 bytes)
	// Below.MinModTime: 1970-01-01 00:00:00 +0000 UTC
	// Below.MaxModTime: 1970-01-01 00:00:00 +0000 UTC
	// Below.MinEntropy: 0.000000
	// Below.MaxEntropy: 0.000000
	// Below.HiEntropy: 0
	// Below.LoEntropy: 0
	// Below.MIMEAudio: 0
	// Below.MIMEVideo: 0
	// Below.MIMEImage: 0
	// Below.MIMEText: 0
	// Below.MIMEApplication: 0
	// Below.MIMEOther: 0
	// Below.Errors: 0
	// Directory.Directories: 0
	// Directory.Files: 3
	// Directory.Symlinks: 0
	// Directory.Devices: 0
	// Directory.Pipes: 0
	// Directory.Sockets: 0
	// Directory.Setuid: 0
	// Directory.Setgid: 0
	// Directory.Sticky: 0
	// Directory.Objects: 3
	// Directory.Chunks: 3
	// Directory.MinSize: 9 B (9 bytes)
	// Directory.MaxSize: 20 B (20 bytes)
	// Directory.Size: 40 B (40 bytes)
	// Directory.MinModTime: 2025-03-10 22:19:19 +0000 UTC
	// Directory.MaxModTime: 2025-03-10 22:19:19 +0000 UTC
	// Directory.MinEntropy: 2.419382
	// Directory.MaxEntropy: 3.921928
	// Directory.AvgEntropy: 3.145702
	// Directory.HiEntropy: 0
	// Directory.LoEntropy: 0
	// Directory.MIMEAudio: 0
	// Directory.MIMEVideo: 0
	// Directory.MIMEImage: 0
	// Directory.MIMEText: 3
	// Directory.MIMEApplication: 0
	// Directory.MIMEOther: 0
	// Directory.Errors: 0
	// Directory.Children: 3
	// Child[0].FileInfo.Name(): dummy.txt
	// Child[0].FileInfo.Size(): 11
	// Child[0].FileInfo.Mode(): -rw-r--r--
	// Child[0].FileInfo.Dev(): 64768
	// Child[0].FileInfo.Ino(): 22415073
	// Child[0].FileInfo.Uid(): 1000
	// Child[0].FileInfo.Gid(): 1000
	// Child[0].FileInfo.Username(): sayoun
	// Child[0].FileInfo.Groupname(): sayoun
	// Child[0].FileInfo.Nlink(): 1
	// Child[0].ExtendedAttributes(): []
	// Child[1].FileInfo.Name(): foo.txt
	// Child[1].FileInfo.Size(): 9
	// Child[1].FileInfo.Mode(): -rw-r--r--
	// Child[1].FileInfo.Dev(): 64768
	// Child[1].FileInfo.Ino(): 22415074
	// Child[1].FileInfo.Uid(): 1000
	// Child[1].FileInfo.Gid(): 1000
	// Child[1].FileInfo.Username(): sayoun
	// Child[1].FileInfo.Groupname(): sayoun
	// Child[1].FileInfo.Nlink(): 1
	// Child[1].ExtendedAttributes(): []
	// Child[2].FileInfo.Name(): to_exclude
	// Child[2].FileInfo.Size(): 20
	// Child[2].FileInfo.Mode(): -rw-r--r--
	// Child[2].FileInfo.Dev(): 64768
	// Child[2].FileInfo.Ino(): 22415075
	// Child[2].FileInfo.Uid(): 1000
	// Child[2].FileInfo.Gid(): 1000
	// Child[2].FileInfo.Username(): sayoun
	// Child[2].FileInfo.Groupname(): sayoun
	// Child[2].FileInfo.Nlink(): 1
	// Child[2].ExtendedAttributes(): []

	output = bufOut.String()
	require.Contains(t, output, "[DirEntry]")
	require.Contains(t, output, "ParentPath: /")
	require.Contains(t, output, "Name: subdir")
	require.Contains(t, output, "Directory.Files: 3")
}

func TestExecuteCmdDiagXattr(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "xattr", fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId[:]))}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this

	output := bufOut.String()
	require.Equal(t, "", strings.Trim(output, "\n"))
}

func TestExecuteCmdDiagContentType(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "contenttype", fmt.Sprintf("%s:subdir/dummy.txt", hex.EncodeToString(indexId[:]))}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this

	output := bufOut.String()
	require.Equal(t, "", strings.Trim(output, "\n"))
}

func TestExecuteCmdDiagLocks(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{"diag", "locks"}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this

	output := bufOut.String()
	require.Equal(t, "", strings.Trim(output, "\n"))
}

func TestExecuteCmdDiagSearch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"diag", "search", fmt.Sprintf("%s:subdir/", hex.EncodeToString(indexId[:]))}

	subcommand, _, args := subcommands.Lookup(args)
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// f3b3c31e:/tmp/tmp_to_backup3206526426/another_subdir/bar
	// f3b3c31e:/tmp/tmp_to_backup3206526426/subdir/dummy.txt
	// f3b3c31e:/tmp/tmp_to_backup3206526426/subdir/foo.txt
	// f3b3c31e:/tmp/tmp_to_backup3206526426/subdir/to_exclude

	output := bufOut.String()
	require.Contains(t, output, "subdir/dummy.txt")
}
