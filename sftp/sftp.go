package sftp

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	sshcfg "github.com/kevinburke/ssh_config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

func Connect(endpoint *url.URL, config map[string]string) (*sftp.Client, error) {
	// if plakar config has identity or insecure_ignore_host_key, use them
	identity := config["identity"]
	ignoreHostKey := config["insecure_ignore_host_key"] == "true"

	// resolve ~/.ssh/config and provide fallback values
	sshConfig := resolveSSHConfig(endpoint.User.Username(), endpoint.Hostname(), endpoint.Port())

	var username string
	var host string
	var port string

	// this is either the override or the default fallback
	host = sshConfig["host"]
	port = sshConfig["port"]

	// if username is set in the URL, use it otherwise:
	// - fallback to ssh_config
	// - fallback to the user environment variable
	// - fallback to the current user
	if endpoint.User.Username() != "" {
		username = endpoint.User.Username()
	} else {
		username = sshConfig["user"]
		if username == "" {
			username = os.Getenv("USER")
			if username == "" {
				u, err := user.Current()
				if err != nil {
					return nil, err
				}
				username = u.Username
			}
		}
	}

	// if identity is set in sshConfig and we didn't get one from the config, use it
	if sshConfig["identity"] != "" && identity == "" {
		identity = sshConfig["identity"]
		if strings.HasPrefix(identity, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %v", err)
			}
			identity = filepath.Join(home, identity[2:])
		}
	}

	// look for home directory to resolve known hosts...
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// ... unless one was provided in config for override
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")
	if customPath := sshConfig["known_hosts"]; customPath != "" {
		knownHostsPath = customPath
	}

	if sshConfig["strict_host_checking"] != "" && !ignoreHostKey {
		switch strings.ToLower(sshConfig["strict_host_checking"]) {
		case "no":
			ignoreHostKey = true
		case "yes":
			ignoreHostKey = false
		case "ask":
			// current prompt behavior
		default:
			// fallback to current prompt behavior
		}
	}

	hostKeyCallback, err := safeHostKeyCallback(knownHostsPath, ignoreHostKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}

	target := net.JoinHostPort(host, port)
	proxyCommand := sshConfig["proxy_command"]

	// if there's a proxy command, pipe
	var conn net.Conn
	if proxyCommand != "" {
		proxyCommand = strings.ReplaceAll(proxyCommand, "%h", host)
		proxyCommand = strings.ReplaceAll(proxyCommand, "%p", port)

		cmd := exec.Command("sh", "-c", proxyCommand)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, fmt.Errorf("proxy stdin pipe failed: %w", err)
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("proxy stdout pipe failed: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("proxy command failed to start: %w", err)
		}

		conn = &proxyConn{
			stdin:  stdin,
			stdout: stdout,
			cmd:    cmd,
		}
	} else {
		conn, err = net.Dial("tcp", target)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", target, err)
		}
	}

	// we're done !
	// connect to the server (or proxy)
	sshClientConn, chans, reqs, err := ssh.NewClientConn(conn, target, &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
				return loadSigners(identity)
			}),
		},
		HostKeyCallback: hostKeyCallback,
	})
	if err != nil {
		return nil, fmt.Errorf("ssh handshake failed: %w", err)
	}

	client := ssh.NewClient(sshClientConn, chans, reqs)

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, err
	}
	return sftpClient, nil
}

func resolveSSHConfig(username string, host string, port string) map[string]string {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	file, err := os.Open(cfgPath)
	if err != nil {
		return map[string]string{}
	}
	defer file.Close()

	cfg, err := sshcfg.Decode(file)
	if err != nil {
		return map[string]string{}
	}

	get := func(field string) string {
		val, err := cfg.Get(host, field)
		if err != nil {
			return ""
		}
		return val
	}

	return map[string]string{
		"host":                 fallback(get("HostName"), host),
		"user":                 fallback(get("User"), username),
		"port":                 fallback(fallback(get("Port"), port), "22"),
		"identity":             get("IdentityFile"),
		"known_hosts":          get("UserKnownHostsFile"),
		"strict_host_checking": get("StrictHostKeyChecking"),
		"proxy_command":        get("ProxyCommand"),
	}
}

func fallback(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func loadSigners(keyPath string) ([]ssh.Signer, error) {
	var signers []ssh.Signer

	// 0. Check for explicitly configured identity file
	if keyPath != "" {
		data, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read specified key %s: %w", keyPath, err)
		}

		// Try without passphrase
		signer, err := ssh.ParsePrivateKey(data)
		if err == nil {
			return []ssh.Signer{signer}, nil
		}

		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			fmt.Printf("Enter passphrase for %s: ", keyPath)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return nil, fmt.Errorf("failed to read passphrase: %w", err)
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, passphrase)
			if err != nil {
				return nil, fmt.Errorf("invalid passphrase for %s: %w", keyPath, err)
			}
			return []ssh.Signer{signer}, nil
		}

		return nil, fmt.Errorf("failed to parse specified key %s: %w", keyPath, err)
	}

	// 1. Check SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(conn)
			if agentSigners, err := ag.Signers(); err == nil && len(agentSigners) > 0 {
				return agentSigners, nil // ✅ Use agent keys silently
			}
		}
	}

	// 2. Fallback to local keys
	keyFiles := []string{
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ed25519"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_ecdsa"),
		filepath.Join(os.Getenv("HOME"), ".ssh", "id_dsa"),
	}

	for _, file := range keyFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Try without passphrase
		signer, err := ssh.ParsePrivateKey(data)
		if err == nil {
			signers = append(signers, signer)
			continue
		}

		// Prompt if encrypted
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			fmt.Printf("Enter passphrase for %s: ", file)
			passphrase, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				continue
			}
			signer, err = ssh.ParsePrivateKeyWithPassphrase(data, passphrase)
			if err == nil {
				signers = append(signers, signer)
			}
		}
	}

	if len(signers) == 0 {
		return nil, fmt.Errorf("no usable SSH keys found")
	}

	return signers, nil
}

func sha256Fingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return "SHA256:" + base64.StdEncoding.EncodeToString(hash[:])
}

func safeHostKeyCallback(knownHostsPath string, ignore bool) (ssh.HostKeyCallback, error) {
	if ignore {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	rawCallback, err := knownhosts.New(knownHostsPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to parse known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		safeRemote := remote
		if remote == nil || !strings.Contains(remote.String(), ":") {
			safeRemote = dummyAddrWithPort("0.0.0.0:22")
		}

		if rawCallback != nil {
			err := rawCallback(hostname, safeRemote, key)
			if err == nil {
				return nil
			}
			// Handle unknown key
			var keyErr *knownhosts.KeyError
			if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
				return err
			}
		}

		// Prompt user to trust
		fingerprint := sha256Fingerprint(key)
		fmt.Printf("The authenticity of host '%s' can't be established.\n", hostname)
		fmt.Printf("%s key fingerprint is %s.\n", key.Type(), fingerprint)
		fmt.Print("Are you sure you want to continue connecting (yes/no)? ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return fmt.Errorf("failed to read user input")
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "yes" {
			return fmt.Errorf("host key not accepted")
		}

		// Add entry to known_hosts
		hostKey := hostname
		if port := getPort(remote); port != "22" {
			hostKey = fmt.Sprintf("[%s]:%s", hostname, port)
		}
		line := fmt.Sprintf("%s %s", hostKey, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))

		if err := os.MkdirAll(filepath.Dir(knownHostsPath), 0700); err != nil {
			return fmt.Errorf("could not create .ssh dir: %w", err)
		}
		f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open known_hosts file: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("failed to write known_hosts entry: %w", err)
		}

		fmt.Printf("Added %s to known hosts.\n", hostKey)
		return nil
	}, nil
}

func getPort(addr net.Addr) string {
	if addr == nil {
		return "22"
	}
	_, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "22"
	}
	return port
}

// Wraps stdin/stdout of ProxyCommand as a net.Conn
type proxyConn struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func (p *proxyConn) Read(b []byte) (int, error)         { return p.stdout.Read(b) }
func (p *proxyConn) Write(b []byte) (int, error)        { return p.stdin.Write(b) }
func (p *proxyConn) Close() error                       { return p.cmd.Process.Kill() }
func (p *proxyConn) LocalAddr() net.Addr                { return dummyAddr("local") }
func (p *proxyConn) RemoteAddr() net.Addr               { return dummyAddr("remote") }
func (p *proxyConn) SetDeadline(t time.Time) error      { return nil }
func (p *proxyConn) SetReadDeadline(t time.Time) error  { return nil }
func (p *proxyConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (d dummyAddr) Network() string { return string(d) }
func (d dummyAddr) String() string  { return string(d) }

type dummyAddrWithPort string

func (d dummyAddrWithPort) Network() string { return "tcp" }
func (d dummyAddrWithPort) String() string  { return string(d) }
