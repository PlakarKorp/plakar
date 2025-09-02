package plugins

import (
	"net"
	"os"
	"os/exec"
	"time"
)

type StdioConn struct {
	stdin  *os.File
	stdout *os.File
	cmd    *exec.Cmd
}

var stdioaddr = &net.UnixAddr{Name: "stdio", Net: "unix"}

func NewStdioConn(stdin, stdout *os.File, cmd *exec.Cmd) net.Conn {
	return &StdioConn{stdin, stdout, cmd}
}

func (c *StdioConn) Read(b []byte) (int, error)  { return c.stdin.Read(b) }
func (c *StdioConn) Write(b []byte) (int, error) { return c.stdout.Write(b) }
func (c *StdioConn) LocalAddr() net.Addr         { return stdioaddr }
func (c *StdioConn) RemoteAddr() net.Addr        { return stdioaddr }

func (c *StdioConn) Close() (ret error) {
	if err := c.stdin.Close(); err != nil {
		ret = err
	}
	if err := c.stdout.Close(); err != nil {
		ret = err
	}
	if c.cmd != nil {
		if err := c.cmd.Wait(); err != nil {
			ret = err
		}
	}
	return
}

func (c *StdioConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *StdioConn) SetReadDeadline(t time.Time) error  { return c.stdin.SetReadDeadline(t) }
func (c *StdioConn) SetWriteDeadline(t time.Time) error { return c.stdout.SetWriteDeadline(t) }
