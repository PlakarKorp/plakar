package agent

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

type Packet struct {
	Type     string
	Data     []byte
	ExitCode int
	Eof      bool
	Err      string
}

type Client struct {
	conn net.Conn
	enc  *msgpack.Encoder
	dec  *msgpack.Decoder
}

var (
	ErrWrongVersion = errors.New("agent is running with a different version of plakar")
)

func ExecuteRPC(ctx *appcontext.AppContext, name []string, cmd subcommands.Subcommand, storeConfig map[string]string) (int, error) {
	client, err := NewClient(filepath.Join(ctx.CacheDir, "agent.sock"), cmd.GetFlags()&subcommands.IgnoreVersion != 0)
	if err != nil {
		return 1, err
	}
	defer client.Close()

	go func() {
		<-ctx.Done()
		client.Close()
	}()

	if status, err := client.SendCommand(ctx, name, cmd, storeConfig); err != nil {
		return status, err
	}
	return 0, nil
}

func NewClient(socketPath string, ignoreVersion bool) (*Client, error) {
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
			return nil, fmt.Errorf("failed to run the agent")
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

			plakar := exec.Command(me, "agent", "start")
			if err := plakar.Start(); err != nil {
				return nil, fmt.Errorf("failed to start the agent: %w", err)
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

	var agentvers []byte
	if err := c.dec.Decode(&agentvers); err != nil {
		return err
	}

	if !ignoreVersion && !slices.Equal(ourvers, agentvers) {
		return fmt.Errorf("%w (%v)", ErrWrongVersion, string(agentvers))
	}

	return nil
}

func (c *Client) SendCommand(ctx *appcontext.AppContext, name []string, cmd subcommands.Subcommand, storeConfig map[string]string) (int, error) {
	if cmd.GetFlags()&subcommands.AgentSupport == 0 {
		return 1, fmt.Errorf("command %v doesn't support execution through agent", strings.Join(name, " "))
	}

	cmd.SetLogInfo(ctx.GetLogger().EnabledInfo)
	cmd.SetLogTraces(ctx.GetLogger().EnabledTracing)

	if err := subcommands.EncodeRPC(c.enc, name, cmd, storeConfig); err != nil {
		return 1, err
	}

	var response Packet
	for {
		if err := c.dec.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			if err := ctx.Err(); err != nil {
				return 1, err
			}
			return 1, fmt.Errorf("failed to decode response: %w", err)
		}
		switch response.Type {
		case "stdin":
			var buf [8192]byte
			n, err := os.Stdin.Read(buf[:])
			pkt := &Packet{
				Type: "stdin",
				Data: buf[:n],
			}
			if err != nil {
				pkt.Eof = err == io.EOF
				pkt.Err = err.Error()
			}
			err = c.enc.Encode(pkt)
			if err != nil {
				return 1, fmt.Errorf("failed to send stdin: %w", err)
			}
		case "stdout":
			fmt.Printf("%s", string(response.Data))
		case "stderr":
			fmt.Fprintf(os.Stderr, "%s", string(response.Data))
		case "exit":
			var err error
			if response.Err != "" {
				err = fmt.Errorf("%s", response.Err)
			}
			return response.ExitCode, err
		}
	}
	return 0, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
