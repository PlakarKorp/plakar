package null

import (
	"bytes"
	"io"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/stretchr/testify/require"
)

func TestNullBackend(t *testing.T) {
	ctx := appcontext.NewAppContext()

	// create a repository
	repo, err := NewStore(ctx, map[string]string{"location": "/test/location"})
	if err != nil {
		t.Fatal("error creating repository", err)
	}

	location, err := repo.Location(ctx)
	require.NoError(t, err)
	require.Equal(t, "/test/location", location)

	config := storage.NewConfiguration()
	serializedConfig, err := config.ToBytes()
	require.NoError(t, err)

	err = repo.Create(ctx, serializedConfig)
	require.NoError(t, err)

	_, err = repo.Open(ctx)
	require.NoError(t, err)
	// only test one field
	//require.Equal(t, repo.Configuration().Version, versioning.FromString(storage.VERSION))

	err = repo.Close(ctx)
	require.NoError(t, err)

	mac := objects.MAC{0x10}

	// states
	macs, err := repo.GetStates(ctx)
	require.NoError(t, err)
	require.Equal(t, macs, []objects.MAC{})

	_, err = repo.PutState(ctx, mac, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	rd, err := repo.GetState(ctx, mac)
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	err = repo.DeleteState(ctx, mac)
	require.NoError(t, err)

	// packfiles
	macs, err = repo.GetPackfiles(ctx)
	require.NoError(t, err)
	require.Equal(t, macs, []objects.MAC{})

	_, err = repo.PutPackfile(ctx, mac, bytes.NewReader([]byte("test")))
	require.NoError(t, err)

	rd, err = repo.GetPackfile(ctx, mac)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	rd, err = repo.GetPackfileBlob(ctx, mac, 0, 0)
	buf = new(bytes.Buffer)
	_, err = io.Copy(buf, rd)
	require.NoError(t, err)
	require.Equal(t, "", buf.String())

	err = repo.DeletePackfile(ctx, mac)
	require.NoError(t, err)
}
