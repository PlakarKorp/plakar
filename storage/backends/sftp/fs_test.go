package sftp

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/stretchr/testify/require"
)

func TestFsBackend(t *testing.T) {
	t.Cleanup(func() {
		os.RemoveAll("/tmp/testfs")
	})
	// create a repository
	repo := NewRepository("fs:///tmp/testfs")
	if repo == nil {
		t.Fatal("error creating repository")
	}

	location := repo.Location()
	require.Equal(t, "fs:///tmp/testfs", location)

	config := storage.NewConfiguration()
	serialized, err := config.ToBytes()
	require.NoError(t, err)

	err = repo.Create("fs:///tmp/testfs", serialized)
	require.NoError(t, err)

	_, err = repo.Open("fs:///tmp/testfs")
	require.NoError(t, err)
	//require.Equal(t, repo.Configuration().Version, versioning.FromString(storage.VERSION))

	err = repo.Close()
	require.NoError(t, err)

	// states
	mac1 := objects.MAC{0x10, 0x20}
	mac2 := objects.MAC{0x30, 0x40}
	err = repo.PutState(mac1, bytes.NewReader([]byte("test1")))
	require.NoError(t, err)
	err = repo.PutState(mac2, bytes.NewReader([]byte("test2")))
	require.NoError(t, err)

	states, err := repo.GetStates()
	require.NoError(t, err)
	expected := []objects.MAC{
		{0x10, 0x20, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		{0x30, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
	}
	require.Equal(t, expected, states)

	rd, err := repo.GetState(mac2)
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test2", buf.String())

	err = repo.DeleteState(mac1)
	require.NoError(t, err)

	states, err = repo.GetStates()
	require.NoError(t, err)
	expected = []objects.MAC{{0x30, 0x40, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}}
	require.Equal(t, expected, states)

	// packfiles
	mac3 := objects.MAC{0x50, 0x60}
	mac4 := objects.MAC{0x60, 0x70}
	err = repo.PutPackfile(mac3, bytes.NewReader([]byte("test3")))
	require.NoError(t, err)
	err = repo.PutPackfile(mac4, bytes.NewReader([]byte("test4")))
	require.NoError(t, err)

	packfiles, err := repo.GetPackfiles()
	require.NoError(t, err)
	expected = []objects.MAC{
		{0x50, 0x60, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		{0x60, 0x70, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
	}
	require.Equal(t, expected, packfiles)

	rd, err = repo.GetPackfileBlob(mac4, 0, 4)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test", buf.String())

	err = repo.DeletePackfile(mac3)
	require.NoError(t, err)

	packfiles, err = repo.GetPackfiles()
	require.NoError(t, err)
	expected = []objects.MAC{{0x60, 0x70, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}}
	require.Equal(t, expected, packfiles)

	rd, err = repo.GetPackfile(mac4)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test4", buf.String())

}
