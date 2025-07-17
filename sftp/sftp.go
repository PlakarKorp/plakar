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

type config struct {
	user               string
	host               string
	port               string
	identity           string
	knownHosts         string
	strictHostChecking bool
	proxyCmd           string
}

func Connect(endpoint *url.URL, params map[string]string) (*sftp.Client, error) {
	var args []string

	// Due to the agent, we can't have anything interactive right now (password/known host check etc)
	// so disable them to fail early and in a meaningful way.
	args = append(args, "-o", "BatchMode=yes")

	if params["insecure_ignore_host_key"] == "true" {
		args = append(args, "-o", "StrictHostKeyChecking=no")
	}

	if id := params["identity"]; id != "" {
		args = append(args, "-i", id)
	}

	if endpoint.User != nil && params["username"] != "" {
		return nil, fmt.Errorf("can not use user@host foo syntax and username parameter.")
	} else if endpoint.User != nil {
		args = append(args, "-l", endpoint.User.Username())
	} else if params["username"] != "" {
		args = append(args, "-l", params["username"])
	}

	if endpoint.Port() != "" {
		args = append(args, "-p", endpoint.Port())
	}

	args = append(args, endpoint.Hostname())

	// This one must be after the host, tell the ssh command to load the sftp subsystem
	args = append(args, "-s", "sftp")

	cmd := exec.Command("ssh", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	var sshErr error
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "Warning:") {
				continue
			}

			sshErr = fmt.Errorf("ssh command error: %q", sc.Text())
		}
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go func() {
		cmd.Wait()
	}()

	client, err := sftp.NewClientPipe(stdout, stdin)
	if err != nil {
		if sshErr != nil {
			return nil, sshErr
		} else {
			return nil, err
		}
	}

	return client, nil
}

func OldConnect(endpoint *url.URL, params map[string]string) (*sftp.Client, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	config, err := parseconfig(home, params, endpoint)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := safeHostKeyCallback(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key callback: %w", err)
	}

	target := net.JoinHostPort(config.host, config.port)
	proxyCommand := config.proxyCmd

	// if there's a proxy command, pipe
	var conn net.Conn
	if proxyCommand != "" {
		proxyCommand = strings.ReplaceAll(proxyCommand, "%h", config.host)
		proxyCommand = strings.ReplaceAll(proxyCommand, "%p", config.port)

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
		User: config.user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
				return loadSigners(config.identity)
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

func parseconfig(home string, params map[string]string, endpoint *url.URL) (*config, error) {
	username := endpoint.User.Username()
	if username == "" {
		username = os.Getenv("USER")
		if username == "" {
			u, err := user.Current()
			if err != nil {
				return nil, fmt.Errorf("can't get current user: %v", err)
			}
			username = u.Username
		}
	}

	cfgPath := filepath.Join(home, ".ssh", "config")
	file, err := os.Open(cfgPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to open %s: %v", cfgPath, err)
	}

	var cfg *sshcfg.Config
	if file != nil {
		cfg, err = sshcfg.Decode(file)
		file.Close()
		if err != nil {
			return nil, err
		}
	}

	host := endpoint.Hostname()
	get := func(field string, def ...string) string {
		var ret string
		if cfg != nil {
			ret, _ = cfg.Get(host, field)
		}
		if ret == "" {
			for _, d := range def {
				if d != "" {
					return d
				}
			}
		}
		return ret
	}

	strict := params["insecure_ignore_host_key"] == "true"
	if !strict {
		strict = get("StrictHostKeyChecking") == "yes"
	}

	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	conf := &config{
		user:               get("User", username),
		host:               get("HostName", host),
		port:               get("Port", endpoint.Port(), "22"),
		identity:           get("IdentityFile", params["identity"]),
		knownHosts:         get("UserKnownHostsFile", knownHosts),
		strictHostChecking: strict,
		proxyCmd:           get("ProxyCommand"),
	}

	if strings.HasPrefix(conf.identity, "~/") {
		conf.identity = filepath.Join(home, conf.identity[2:])
	}

	return conf, nil
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
				return agentSigners, nil
			}
		}
	}

	// 2. Fallback to local keys
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	keyFiles := []string{
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
		filepath.Join(home, ".ssh", "id_dsa"),
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

func safeHostKeyCallback(config *config) (ssh.HostKeyCallback, error) {
	if config.strictHostChecking {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	rawCallback, err := knownhosts.New(config.knownHosts)
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

		if err := os.MkdirAll(filepath.Dir(config.knownHosts), 0700); err != nil {
			return fmt.Errorf("could not create .ssh dir: %w", err)
		}
		f, err := os.OpenFile(config.knownHosts, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
