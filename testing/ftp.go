package testing

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

type MockFTPServer struct {
	Addr     string
	listener net.Listener
	Files    map[string][]byte
	Dirs     map[string]bool
	auth     map[string]string
}

func NewMockFTPServer() (*MockFTPServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %v", err)
	}

	server := &MockFTPServer{
		Addr:     listener.Addr().String(),
		listener: listener,
		Files:    make(map[string][]byte),
		Dirs:     make(map[string]bool),
		auth: map[string]string{
			"test": "test",
		},
	}

	// Start the server
	go server.serve()

	return server, nil
}

func (s *MockFTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConnection(conn)
	}
}

func (s *MockFTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	var username string
	var authenticated bool
	var dataConn net.Conn

	// Send welcome message
	conn.Write([]byte("220 Welcome to mock FTP server\r\n"))

	// Command handling
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		cmd := strings.TrimSpace(string(buf[:n]))
		switch {
		case strings.HasPrefix(cmd, "USER"):
			username = strings.TrimPrefix(cmd, "USER ")
			username = strings.TrimSpace(username)
			conn.Write([]byte("331 Please specify the password\r\n"))
		case strings.HasPrefix(cmd, "PASS"):
			password := strings.TrimPrefix(cmd, "PASS ")
			password = strings.TrimSpace(password)
			if expectedPass, exists := s.auth[username]; exists && expectedPass == password {
				authenticated = true
				conn.Write([]byte("230 Login successful\r\n"))
			} else {
				conn.Write([]byte("530 Login incorrect\r\n"))
				return // Immediately close connection on auth failure
			}
		case strings.HasPrefix(cmd, "SYST"):
			conn.Write([]byte("215 UNIX Type: L8\r\n"))
		case strings.HasPrefix(cmd, "FEAT"):
			conn.Write([]byte("211-Features:\r\n"))
			conn.Write([]byte(" PASV\r\n"))
			conn.Write([]byte(" UTF8\r\n"))
			conn.Write([]byte("211 End\r\n"))
		case strings.HasPrefix(cmd, "PWD"):
			conn.Write([]byte("257 \"/\" is current directory\r\n"))
		case strings.HasPrefix(cmd, "TYPE"):
			conn.Write([]byte("200 Type set to I\r\n"))
		case strings.HasPrefix(cmd, "PASV"):
			// Create a listener for data connection
			dataListener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}
			defer dataListener.Close()

			// Get the port number
			_, portStr, _ := net.SplitHostPort(dataListener.Addr().String())
			port, _ := strconv.Atoi(portStr)

			// Send the passive mode response with the port number
			conn.Write([]byte(fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)\r\n", port>>8, port&0xFF)))

			// Accept the data connection
			dataConn, err = dataListener.Accept()
			if err != nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}
			defer dataConn.Close()
		case strings.HasPrefix(cmd, "LIST"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			conn.Write([]byte("150 Opening data connection\r\n"))
			if dataConn != nil {
				dataConn.Write([]byte("-rw-r--r-- 1 ftp ftp 0 Jan 1 00:00 .\r\n"))
				dataConn.Close()
				dataConn = nil
				conn.Write([]byte("226 Transfer complete\r\n"))
			} else {
				conn.Write([]byte("425 Can't open data connection\r\n"))
			}
		case strings.HasPrefix(cmd, "MKD"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			dir := strings.TrimPrefix(cmd, "MKD ")
			s.Dirs[dir] = true
			conn.Write([]byte("257 \"" + dir + "\" directory created\r\n"))
		case strings.HasPrefix(cmd, "STOR"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			file := strings.TrimPrefix(cmd, "STOR ")
			conn.Write([]byte("150 Ok to send data\r\n"))

			// Read file data from data connection
			data := make([]byte, 1024)
			n, err := dataConn.Read(data)
			if err != nil && err != io.EOF {
				conn.Write([]byte("550 Error reading file\r\n"))
				continue
			}
			s.Files[file] = data[:n]
			conn.Write([]byte("226 Transfer complete\r\n"))
		case strings.HasPrefix(cmd, "QUIT"):
			conn.Write([]byte("221 Goodbye\r\n"))
			return
		default:
			conn.Write([]byte("500 Unknown command\r\n"))
		}
	}
}

func (s *MockFTPServer) Close() {
	if s.listener != nil {
		s.listener.Close()
	}
}
