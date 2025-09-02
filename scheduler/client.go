package scheduler

import (
	"errors"
	"fmt"
	"net"
	"slices"

	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

type Request struct {
	Type    string
	Payload []byte
}

type Response struct {
	ExitCode int
	Err      string
}

type Client struct {
	conn net.Conn
	enc  *msgpack.Encoder
	dec  *msgpack.Decoder
}

var (
	ErrWrongVersion = errors.New("scheduler is running with a different version of plakar")
)

func NewClient(socketPath string, ignoreVersion bool) (*Client, error) {
	var (
		conn net.Conn
		err  error
	)

	conn, err = net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
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

func (c *Client) Stop() (int, error) {
	var request Request
	request.Type = "stop"
	if err := c.enc.Encode(request); err != nil {
		return 1, fmt.Errorf("failed to send packet: %w", err)
	}

	var response Response
	if err := c.dec.Decode(&response); err != nil {
		return 1, fmt.Errorf("failed to decode response: %w", err)
	}

	var err error
	if response.Err != "" {
		err = fmt.Errorf("scheduler error: %s", response.Err)
	}
	return response.ExitCode, err
}

func (c *Client) Close() error {
	return c.conn.Close()
}
