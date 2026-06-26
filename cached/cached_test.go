package cached

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// ---------------------------------------------------------------------------
// FileLock
// ---------------------------------------------------------------------------

func TestLockedFileAndUnlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lock")

	lock, err := LockedFile(path)
	require.NoError(t, err)
	require.NotNil(t, lock)
	require.Equal(t, path, lock.Path)

	// the lock file should exist while held
	_, statErr := os.Stat(path)
	require.NoError(t, statErr)

	lock.Unlock()

	// Unlock removes the file
	_, statErr = os.Stat(path)
	require.True(t, os.IsNotExist(statErr))
}

func TestFileLockLockDefaultAttempts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "y.lock")
	lock := &FileLock{Path: path} // Attempts == 0 -> defaults to 1000
	require.NoError(t, lock.Lock())
	lock.Unlock()
}

func TestFileLockLockOpenError(t *testing.T) {
	// a path whose parent does not exist cannot be opened
	lock := &FileLock{Path: filepath.Join(t.TempDir(), "missing", "z.lock")}
	require.Error(t, lock.Lock())
}

func TestFlockExclusive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f.lock")
	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0600)
	require.NoError(t, err)
	defer fp.Close()

	require.NoError(t, flock(fp))
}

// ---------------------------------------------------------------------------
// Request / Response packets round-trip
// ---------------------------------------------------------------------------

func TestPacketsMsgpackRoundTrip(t *testing.T) {
	req := RequestPkt{
		Secret:        []byte("s3cr3t"),
		StoreConfig:   map[string]string{"location": "fs:///tmp/x"},
		FireAndForget: true,
	}
	data, err := msgpack.Marshal(&req)
	require.NoError(t, err)

	var got RequestPkt
	require.NoError(t, msgpack.Unmarshal(data, &got))
	require.Equal(t, req.Secret, got.Secret)
	require.Equal(t, req.StoreConfig, got.StoreConfig)
	require.True(t, got.FireAndForget)

	resp := ResponsePkt{Err: "boom", ExitCode: -1}
	data, err = msgpack.Marshal(&resp)
	require.NoError(t, err)
	var gotResp ResponsePkt
	require.NoError(t, msgpack.Unmarshal(data, &gotResp))
	require.Equal(t, resp, gotResp)
}

// ---------------------------------------------------------------------------
// Client.handshake / Close (driven over net.Pipe, no real daemon)
// ---------------------------------------------------------------------------

// fakeServer plays the server side of the handshake: it reads the client's
// version then replies with serverVersion.
func fakeServer(t *testing.T, conn net.Conn, serverVersion []byte) {
	t.Helper()
	dec := msgpack.NewDecoder(conn)
	enc := msgpack.NewEncoder(conn)

	var clientVers []byte
	require.NoError(t, dec.Decode(&clientVers))
	require.Equal(t, utils.GetVersion(), string(clientVers))
	require.NoError(t, enc.Encode(serverVersion))
}

func newPipeClient(c net.Conn) *Client {
	return &Client{
		conn: c,
		enc:  msgpack.NewEncoder(c),
		dec:  msgpack.NewDecoder(c),
	}
}

func TestClientHandshakeMatchingVersion(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()
	defer serverSide.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fakeServer(t, serverSide, []byte(utils.GetVersion()))
	}()

	c := newPipeClient(clientSide)
	require.NoError(t, c.handshake(false))
	wg.Wait()
}

func TestClientHandshakeVersionMismatch(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()
	defer serverSide.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fakeServer(t, serverSide, []byte("99.99.99-bogus"))
	}()

	c := newPipeClient(clientSide)
	err := c.handshake(false)
	require.ErrorIs(t, err, ErrWrongVersion)
	wg.Wait()
}

func TestClientHandshakeIgnoreVersion(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()
	defer serverSide.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		fakeServer(t, serverSide, []byte("99.99.99-bogus"))
	}()

	c := newPipeClient(clientSide)
	// ignoreVersion=true skips the comparison even on a mismatch
	require.NoError(t, c.handshake(true))
	wg.Wait()
}

func TestClientHandshakeEncodeError(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	serverSide.Close()
	clientSide.Close() // writing to a closed pipe fails the initial Encode

	c := newPipeClient(clientSide)
	require.Error(t, c.handshake(false))
}

func TestClientHandshakeDecodeError(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// read the client version then hang up without replying
		dec := msgpack.NewDecoder(serverSide)
		var v []byte
		_ = dec.Decode(&v)
		serverSide.Close()
	}()

	c := newPipeClient(clientSide)
	require.Error(t, c.handshake(false))
	wg.Wait()
}

func TestClientClose(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer serverSide.Close()

	c := newPipeClient(clientSide)
	require.NoError(t, c.Close())
}
