package ptar

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestPtarBackend(t *testing.T) {
	// init temporary directories
	tmpRepoDirRoot, err := os.MkdirTemp("", "tmp_ptar")
	require.NoError(t, err)
	tmpRepoDir := filepath.Join(tmpRepoDirRoot, "repo")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpRepoDir)
		os.RemoveAll(tmpRepoDirRoot)
	})

	ctx := appcontext.NewAppContext()

	// create a repository
	repo, err := NewStore(ctx, "ptar", map[string]string{"location": tmpRepoDir})
	if err != nil {
		t.Fatal("error creating repository", err)
	}

	location, err := repo.Location(ctx)
	require.Equal(t, tmpRepoDir, location)

	config := storage.NewConfiguration()
	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	err = repo.Create(ctx, serializedConfig)
	require.NoError(t, err)

	// packfiles
	mac3 := objects.MAC{0x50, 0x60}
	mac4 := objects.MAC{0x60, 0x70}
	_, err = repo.PutPackfile(ctx, mac3, bytes.NewReader([]byte("test3")))
	require.NoError(t, err)
	_, err = repo.PutPackfile(ctx, mac4, bytes.NewReader([]byte("test4")))
	require.NoError(t, err)

	packfiles, err := repo.GetPackfiles(ctx)
	require.NoError(t, err)
	expected := []objects.MAC{
		packfileMAC,
	}
	require.Equal(t, expected, packfiles)

	rd, err := repo.GetPackfileBlob(ctx, packfileMAC, 0, 4)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test", buf.String())

	err = repo.DeletePackfile(ctx, packfileMAC)
	require.NoError(t, err)

	packfiles, err = repo.GetPackfiles(ctx)
	require.NoError(t, err)
	require.Equal(t, []objects.MAC{packfileMAC}, packfiles)

	rd, err = repo.GetPackfile(ctx, packfileMAC)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test3", buf.String())

	// states
	mac1 := objects.MAC{0x10, 0x20}
	mac2 := objects.MAC{0x30, 0x40}
	_, err = repo.PutState(ctx, mac1, bytes.NewReader([]byte("test1")))
	require.NoError(t, err)
	_, err = repo.PutState(ctx, mac2, bytes.NewReader([]byte("test2")))
	require.NoError(t, err)

	err = repo.Close(ctx)
	require.NoError(t, err)

	_, err = repo.Open(ctx)
	require.NoError(t, err)

	states, err := repo.GetStates(ctx)
	require.NoError(t, err)
	expected = []objects.MAC{
		stateMAC,
	}
	require.Equal(t, expected, states)

	err = repo.Close(ctx)
	require.NoError(t, err)

	_, err = repo.Open(ctx)
	require.NoError(t, err)

	rd, err = repo.GetState(ctx, stateMAC)
	require.NoError(t, err)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "test4", buf.String())
}
