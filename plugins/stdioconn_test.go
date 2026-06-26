package plugins

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStdioConnReadWrite(t *testing.T) {
	// Build a connection whose "stdin" (read side) and "stdout" (write side)
	// are wired through OS pipes so we can exercise Read/Write end to end.
	inR, inW, err := os.Pipe()
	require.NoError(t, err)
	outR, outW, err := os.Pipe()
	require.NoError(t, err)

	conn := NewStdioConn(inR, outW, nil)

	// addresses
	require.Equal(t, stdioaddr, conn.LocalAddr())
	require.Equal(t, stdioaddr, conn.RemoteAddr())
	require.Equal(t, "stdio", conn.LocalAddr().String())

	// Write goes to outW; read it back from outR.
	go func() {
		conn.Write([]byte("ping"))
	}()
	buf := make([]byte, 4)
	n, err := outR.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "ping", string(buf[:n]))

	// Feed inW; Read on conn reads from inR.
	go func() {
		inW.Write([]byte("pong"))
	}()
	n, err = conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "pong", string(buf[:n]))

	outR.Close()
	inW.Close()
	require.NoError(t, conn.Close())
}

func TestStdioConnDeadlines(t *testing.T) {
	inR, _, err := os.Pipe()
	require.NoError(t, err)
	_, outW, err := os.Pipe()
	require.NoError(t, err)

	conn := NewStdioConn(inR, outW, nil)
	t.Cleanup(func() { conn.Close() })

	deadline := time.Now().Add(time.Hour)
	require.NoError(t, conn.SetReadDeadline(deadline))
	require.NoError(t, conn.SetWriteDeadline(deadline))
	require.NoError(t, conn.SetDeadline(deadline))

	// implements net.Conn
	var _ net.Conn = conn
}

func TestSpawnNonexistentBinary(t *testing.T) {
	_, err := spawn(t.Context(), "/nonexistent/plugin/binary", nil)
	require.Error(t, err)
}

func TestConnectPluginNonexistentBinary(t *testing.T) {
	_, err := connectPlugin(t.Context(), "/nonexistent/plugin/binary", nil)
	require.Error(t, err)
}
