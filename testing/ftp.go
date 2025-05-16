package testing

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MockFTPServer struct {
	Addr     string
	listener net.Listener
	Files    map[string][]byte
	Dirs     map[string]bool
	auth     map[string]string
	mu       sync.RWMutex // Protect concurrent access to Files and Dirs
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
	var dataListener net.Listener

	// Send welcome message
	conn.Write([]byte("220 Welcome to mock FTP server\r\n"))

	// Command handling
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if dataListener != nil {
				dataListener.Close()
			}
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
				return
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
			// Close any existing data listener
			if dataListener != nil {
				dataListener.Close()
			}

			// Create a listener for data connection
			var err error
			dataListener, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}

			// Get the port number
			_, portStr, _ := net.SplitHostPort(dataListener.Addr().String())
			port, _ := strconv.Atoi(portStr)

			// Send the passive mode response with the port number
			conn.Write([]byte(fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)\r\n", port>>8, port&0xFF)))

			// Accept the data connection
			dataConn, err = dataListener.Accept()
			if err != nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				dataListener.Close()
				dataListener = nil
				continue
			}
		case strings.HasPrefix(cmd, "LIST"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}

			// Parse the path argument if present
			parts := strings.Fields(cmd)
			listPath := "/"
			if len(parts) > 1 {
				listPath = parts[1]
			}
			if listPath == "" {
				listPath = "/"
			}

			if dataConn == nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}

			conn.Write([]byte("150 Opening data connection\r\n"))

			s.mu.RLock()
			normPath := strings.TrimPrefix(listPath, "/")
			if listPath == "/" {
				// Root directory listing: return . and .. plus all files and dirs in root
				entry := "drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 .\r\n"
				dataConn.Write([]byte(entry))
				entry = "drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 ..\r\n"
				dataConn.Write([]byte(entry))

				// List directories in root
				for dir := range s.Dirs {
					if parentDir(dir) == "/" && dir != "/" {
						entry := fmt.Sprintf("drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 %s\r\n", basename(dir))
						dataConn.Write([]byte(entry))
					}
				}

				// List files in root
				for file := range s.Files {
					if parentDir(file) == "/" {
						content := s.Files[file]
						entry := fmt.Sprintf("-rw-r--r--  1 ftp ftp %8d Jan 01 00:00 %s\r\n", len(content), basename(file))
						dataConn.Write([]byte(entry))
					}
				}
			} else if _, ok := s.Files[normPath]; ok {
				content := s.Files[normPath]
				entry := fmt.Sprintf("-rw-r--r--  1 ftp ftp %8d Jan 01 00:00 %s\r\n", len(content), basename(normPath))
				dataConn.Write([]byte(entry))
			} else {
				// Directory listing
				entry := "drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 .\r\n"
				dataConn.Write([]byte(entry))
				entry = "drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 ..\r\n"
				dataConn.Write([]byte(entry))

				// List directories
				for dir := range s.Dirs {
					if parentDir(dir) == listPath && dir != listPath {
						entry := fmt.Sprintf("drwxr-xr-x  2 ftp ftp     4096 Jan 01 00:00 %s\r\n", basename(dir))
						dataConn.Write([]byte(entry))
					}
				}

				// List files
				for file := range s.Files {
					if parentDir(file) == listPath {
						content := s.Files[file]
						entry := fmt.Sprintf("-rw-r--r--  1 ftp ftp %8d Jan 01 00:00 %s\r\n", len(content), basename(file))
						dataConn.Write([]byte(entry))
					}
				}
			}
			s.mu.RUnlock()

			// Give the client time to read the data
			time.Sleep(100 * time.Millisecond)
			dataConn.Close()
			dataConn = nil
			if dataListener != nil {
				dataListener.Close()
				dataListener = nil
			}
			conn.Write([]byte("226 Transfer complete\r\n"))

		case strings.HasPrefix(cmd, "MKD"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			dir := strings.TrimPrefix(cmd, "MKD ")
			s.mu.Lock()
			s.Dirs[dir] = true
			s.mu.Unlock()
			conn.Write([]byte("257 \"" + dir + "\" directory created\r\n"))

		case strings.HasPrefix(cmd, "STOR"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			file := strings.TrimPrefix(cmd, "STOR ")
			if dataConn == nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}
			conn.Write([]byte("150 Ok to send data\r\n"))

			// Read file data from data connection
			data := make([]byte, 1024)
			n, err := dataConn.Read(data)
			if err != nil && err != io.EOF {
				conn.Write([]byte("550 Error reading file\r\n"))
				continue
			}
			s.mu.Lock()
			s.Files[file] = data[:n]
			s.mu.Unlock()
			dataConn.Close()
			dataConn = nil
			if dataListener != nil {
				dataListener.Close()
				dataListener = nil
			}
			conn.Write([]byte("226 Transfer complete\r\n"))

		case strings.HasPrefix(cmd, "QUIT"):
			conn.Write([]byte("221 Goodbye\r\n"))
			if dataListener != nil {
				dataListener.Close()
			}
			return

		case strings.HasPrefix(cmd, "STAT"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			path := strings.TrimSpace(strings.TrimPrefix(cmd, "STAT"))
			if path == "" {
				path = "/"
			}
			s.mu.RLock()
			if _, ok := s.Dirs[path]; ok {
				conn.Write([]byte("213 Directory status\r\n"))
			} else if _, ok := s.Files[path]; ok {
				conn.Write([]byte("213 File status\r\n"))
			} else {
				conn.Write([]byte("550 File not found\r\n"))
			}
			s.mu.RUnlock()

		case strings.HasPrefix(cmd, "RETR"):
			if !authenticated {
				conn.Write([]byte("530 Please login with USER and PASS\r\n"))
				continue
			}
			file := strings.TrimSpace(strings.TrimPrefix(cmd, "RETR"))
			s.mu.RLock()
			content, ok := s.Files[file]
			s.mu.RUnlock()
			if !ok {
				conn.Write([]byte("550 File not found\r\n"))
				continue
			}
			if dataConn == nil {
				conn.Write([]byte("425 Can't open data connection\r\n"))
				continue
			}
			conn.Write([]byte("150 Opening data connection\r\n"))
			dataConn.Write(content)
			time.Sleep(100 * time.Millisecond)
			dataConn.Close()
			dataConn = nil
			if dataListener != nil {
				dataListener.Close()
				dataListener = nil
			}
			conn.Write([]byte("226 Transfer complete\r\n"))

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

func (s *MockFTPServer) SetAuth(user, pass string) {
	s.mu.Lock()
	s.auth[user] = pass
	s.mu.Unlock()
}

func parentDir(path string) string {
	if path == "/" {
		return ""
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) <= 1 {
		return "/"
	}
	return "/" + strings.Join(parts[:len(parts)-1], "/")
}

func basename(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return parts[len(parts)-1]
}
