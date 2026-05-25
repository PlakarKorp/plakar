package cached

import (
	"errors"
	"net"
	"testing"

	"github.com/PlakarKorp/plakar/utils"
	"github.com/vmihailenco/msgpack/v5"
)

func newPipeClient() (*Client, net.Conn) {
	clientConn, serverConn := net.Pipe()
	c := &Client{
		conn: clientConn,
		enc:  msgpack.NewEncoder(clientConn),
		dec:  msgpack.NewDecoder(clientConn),
	}
	return c, serverConn
}

func TestClientCloseClosesUnderlyingConn(t *testing.T) {
	c, server := newPipeClient()
	defer server.Close()
	if err := c.Close(); err != nil {
		t.Fatalf("Close = %v", err)
	}
	// After close, the server side should observe EOF / closed pipe on read.
	buf := make([]byte, 1)
	if _, err := server.Read(buf); err == nil {
		t.Fatal("expected read error after client closed, got nil")
	}
}

func TestHandshakeSuccess(t *testing.T) {
	c, server := newPipeClient()
	defer c.Close()
	defer server.Close()

	// Server: read our version, echo it back. ourvers and cachedvers match.
	serverDone := make(chan error, 1)
	go func() {
		dec := msgpack.NewDecoder(server)
		enc := msgpack.NewEncoder(server)
		var got []byte
		if err := dec.Decode(&got); err != nil {
			serverDone <- err
			return
		}
		serverDone <- enc.Encode(got)
	}()

	if err := c.handshake(false); err != nil {
		t.Fatalf("handshake = %v, want nil", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("server side: %v", err)
	}
}

func TestHandshakeWrongVersion(t *testing.T) {
	c, server := newPipeClient()
	defer c.Close()
	defer server.Close()

	go func() {
		dec := msgpack.NewDecoder(server)
		enc := msgpack.NewEncoder(server)
		var got []byte
		_ = dec.Decode(&got)
		_ = enc.Encode([]byte("some-other-version-99.99"))
	}()

	err := c.handshake(false)
	if err == nil {
		t.Fatal("expected wrong-version error, got nil")
	}
	if !errors.Is(err, ErrWrongVersion) {
		t.Fatalf("err = %v, want errors.Is(err, ErrWrongVersion)", err)
	}
}

func TestHandshakeWrongVersionIgnored(t *testing.T) {
	c, server := newPipeClient()
	defer c.Close()
	defer server.Close()

	go func() {
		dec := msgpack.NewDecoder(server)
		enc := msgpack.NewEncoder(server)
		var got []byte
		_ = dec.Decode(&got)
		_ = enc.Encode([]byte("some-other-version"))
	}()

	if err := c.handshake(true); err != nil {
		t.Fatalf("handshake(ignoreVersion=true) = %v, want nil", err)
	}
}

func TestHandshakeServerSeesOurVersion(t *testing.T) {
	c, server := newPipeClient()
	defer c.Close()
	defer server.Close()

	got := make(chan []byte, 1)
	go func() {
		dec := msgpack.NewDecoder(server)
		enc := msgpack.NewEncoder(server)
		var v []byte
		_ = dec.Decode(&v)
		got <- v
		_ = enc.Encode(v) // match so handshake succeeds
	}()

	if err := c.handshake(false); err != nil {
		t.Fatalf("handshake = %v", err)
	}
	saw := <-got
	if want := utils.GetVersion(); string(saw) != want {
		t.Fatalf("server saw version %q, want %q", saw, want)
	}
}

func TestRequestPktRoundTrip(t *testing.T) {
	in := RequestPkt{
		Secret:        []byte("s3cr3t"),
		StoreConfig:   map[string]string{"k": "v"},
		FireAndForget: true,
	}
	buf, err := msgpack.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out RequestPkt
	if err := msgpack.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(out.Secret) != "s3cr3t" || !out.FireAndForget || out.StoreConfig["k"] != "v" {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func TestResponsePktRoundTrip(t *testing.T) {
	in := ResponsePkt{Err: "oops", ExitCode: 7}
	buf, err := msgpack.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out ResponsePkt
	if err := msgpack.Unmarshal(buf, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Err != "oops" || out.ExitCode != 7 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}
