package cached

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type PacketType uint8

const (
	RebuildState PacketType = iota
	SnapshotCache
)

type RequestPkt struct {
	Type PacketType

	Secret      []byte
	RepoID      uuid.UUID
	StoreConfig map[string]string

	// Push the request but don't wait for the actual execution.
	FireAndForget bool

	// If empty do a full rebuild otherwise ingest that file from disk.
	StateID objects.MAC

	// Valid only for SnapshotCache type requests
	SnapshotID objects.MAC
}

type ResponsePkt struct {
	Err      string
	ExitCode int
}

type Client struct {
	conn net.Conn
	enc  *msgpack.Encoder
	dec  *msgpack.Decoder
}

var (
	ErrWrongVersion = errors.New("cached is running with a different version of plakar")
)

func cachedRequest(ctx *appcontext.AppContext, req *RequestPkt) (int, error) {
	client, err := newClient(filepath.Join(ctx.CacheDir, "cached.sock"), false)
	if err != nil {
		return 1, err
	}
	defer client.Close()

	go func() {
		<-ctx.Done()
		client.Close()
	}()

	if err := client.enc.Encode(req); err != nil {
		return 1, err
	}

	response := &ResponsePkt{}
	for {
		if err := client.dec.Decode(response); err != nil {
			if err == io.EOF {
				break
			}
			if err := ctx.Err(); err != nil {
				return 1, err
			}
			return 1, fmt.Errorf("failed to decode response: %w", err)
		}

		var err error
		if response.Err != "" {
			err = fmt.Errorf("%s", response.Err)
		}

		return response.ExitCode, err
	}

	return 0, nil
}

func newClient(socketPath string, ignoreVersion bool) (*Client, error) {
	var lockfile *os.File
	var spawned bool

	defer func() {
		if lockfile != nil {
			lockfile.Close()
			os.Remove(lockfile.Name())
		}
	}()

	var (
		attempt int
		conn    net.Conn
		err     error
	)

	for {
		conn, err = net.Dial("unix", socketPath)
		if err == nil {
			// connected successfully!
			break
		}

		attempt++
		if attempt > 1000 {
			return nil, fmt.Errorf("failed to run cached")
		}

		if lockfile == nil {
			lockfile, err = os.OpenFile(socketPath+".lock",
				os.O_WRONLY|os.O_CREATE, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to create lockfile: %w", err)
			}

			err = flock(lockfile)
			if err != nil {
				return nil, fmt.Errorf("failed to take the lock: %w", err)
			}

			// Always retry at least once, even if we got
			// the lock, because another client could have
			// taken the lock, started the server and
			// released the lock between our net.Dial and
			// unix.Flock.

			continue
		}

		if !spawned {
			me, err := os.Executable()
			if err != nil {
				return nil, fmt.Errorf("failed to get executable: %w", err)
			}

			plakar := exec.Command(me, "cached")

			// Cached is daemonized, so we can, and need to wait for the return
			// of the direct child to avoid zombies.
			// The grand children will get reparented to PID 0 as a daemon and
			// will be reaped by PID 0 avoiding zonmbies.
			if err := plakar.Run(); err != nil {
				return nil, fmt.Errorf("failed to start cached: %w", err)
			}
			spawned = true
		}

		time.Sleep(5 * time.Millisecond)
	}

	encoder := msgpack.NewEncoder(conn)
	decoder := msgpack.NewDecoder(conn)

	c := &Client{
		conn: conn,
		enc:  encoder,
		dec:  decoder,
	}

	if err := c.handshake(ignoreVersion); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) handshake(ignoreVersion bool) error {
	ourvers := []byte(utils.GetVersion())

	if err := c.enc.Encode(ourvers); err != nil {
		return err
	}

	var cachedvers []byte
	if err := c.dec.Decode(&cachedvers); err != nil {
		return err
	}

	if !ignoreVersion && !slices.Equal(ourvers, cachedvers) {
		return fmt.Errorf("%w (%v)", ErrWrongVersion, string(cachedvers))
	}

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
