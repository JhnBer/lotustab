package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	ParamEnabled = "sshEnabled"
	ParamHost    = "sshHost"
	ParamPort    = "sshPort"
	ParamUser    = "sshUser"
	ParamAuth    = "sshAuth"
	ParamKeyPath = "sshKeyPath"

	ParamJumpEnabled = "sshJumpEnabled"
	ParamJumpHost    = "sshJumpHost"
	ParamJumpPort    = "sshJumpPort"
	ParamJumpUser    = "sshJumpUser"
	ParamJumpAuth    = "sshJumpAuth"
	ParamJumpKeyPath = "sshJumpKeyPath"

	AuthKey      = "key"
	AuthPassword = "password"
	AuthAgent    = "agent"

	DefaultPort = "22"

	keepAliveInterval = 30 * time.Second
	dialKeepAlive     = 30 * time.Second
)

// Config is the SSH hop used to reach a database host.
type Config struct {
	Host       string
	Port       string
	User       string
	Auth       string
	KeyPath    string
	Passphrase string
	Password   string
}

// ConfigFromParams reads SSH settings from a profile params map.
// ok is false when the tunnel is disabled.
func ConfigFromParams(params map[string]string, sshPassword, sshPassphrase string) (Config, bool, error) {
	if params == nil || params[ParamEnabled] != "true" {
		return Config{}, false, nil
	}
	return parseConfig(params, ParamHost, ParamPort, ParamUser, ParamAuth, ParamKeyPath, sshPassword, sshPassphrase, "SSH")
}

// JumpConfigFromParams reads the optional jump-host hop.
// ok is false when the main tunnel or the jump is disabled.
func JumpConfigFromParams(params map[string]string, jumpPassword, jumpPassphrase string) (Config, bool, error) {
	if params == nil || params[ParamEnabled] != "true" || params[ParamJumpEnabled] != "true" {
		return Config{}, false, nil
	}
	return parseConfig(params, ParamJumpHost, ParamJumpPort, ParamJumpUser, ParamJumpAuth, ParamJumpKeyPath, jumpPassword, jumpPassphrase, "SSH jump")
}

func parseConfig(params map[string]string, hostKey, portKey, userKey, authKey, keyPathKey, password, passphrase, label string) (Config, bool, error) {
	auth := strings.TrimSpace(params[authKey])
	if auth == "" {
		auth = AuthKey
	}
	cfg := Config{
		Host:       strings.TrimSpace(params[hostKey]),
		Port:       strings.TrimSpace(params[portKey]),
		User:       strings.TrimSpace(params[userKey]),
		Auth:       auth,
		KeyPath:    strings.TrimSpace(params[keyPathKey]),
		Passphrase: passphrase,
		Password:   password,
	}
	if cfg.Port == "" {
		cfg.Port = DefaultPort
	}
	if cfg.Host == "" {
		return Config{}, true, fmt.Errorf("%s hostname is required", label)
	}
	if _, err := strconv.ParseUint(cfg.Port, 10, 16); err != nil {
		return Config{}, true, fmt.Errorf("invalid %s port", label)
	}
	if cfg.User == "" {
		return Config{}, true, fmt.Errorf("%s username is required", label)
	}
	switch cfg.Auth {
	case AuthKey:
		if cfg.KeyPath == "" {
			return Config{}, true, fmt.Errorf("%s private key file is required", label)
		}
	case AuthPassword, AuthAgent:
	default:
		return Config{}, true, fmt.Errorf("unknown %s auth method", label)
	}
	return cfg, true, nil
}

// Client is an open SSH session that can dial through to a destination.
type Client struct {
	ssh    *ssh.Client
	extra  io.Closer
	via    io.Closer
	stop   chan struct{}
	closed sync.Once
}

func newClient(sshClient *ssh.Client, extra, via io.Closer) *Client {
	c := &Client{ssh: sshClient, extra: extra, via: via, stop: make(chan struct{})}
	go c.keepAlive()
	return c
}

// Open dials the SSH server. The caller must Close the client.
func Open(ctx context.Context, cfg Config) (*Client, error) {
	auth, extra, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	sshCfg := clientConfig(cfg.User, auth)
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	dialer := net.Dialer{Timeout: sshCfg.Timeout, KeepAlive: dialKeepAlive}
	netConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if extra != nil {
			_ = extra.Close()
		}
		return nil, fmt.Errorf("ssh: %w", err)
	}
	client, err := handshake(ctx, netConn, addr, sshCfg)
	if err != nil {
		if extra != nil {
			_ = extra.Close()
		}
		return nil, err
	}
	return newClient(client, extra, nil), nil
}

// OpenVia dials cfg through an already-open hop. It takes ownership of hop:
// on success hop is closed by Client.Close; on failure hop is closed immediately.
func OpenVia(ctx context.Context, hop *Client, cfg Config) (*Client, error) {
	if hop == nil {
		return nil, fmt.Errorf("ssh jump: missing hop")
	}
	auth, extra, err := authMethods(cfg)
	if err != nil {
		_ = hop.Close()
		return nil, err
	}
	if hop.ssh == nil {
		if extra != nil {
			_ = extra.Close()
		}
		_ = hop.Close()
		return nil, fmt.Errorf("ssh jump: missing hop")
	}
	sshCfg := clientConfig(cfg.User, auth)
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
	}
	netConn, err := hop.ssh.DialContext(ctx, "tcp", addr)
	if err != nil {
		if extra != nil {
			_ = extra.Close()
		}
		_ = hop.Close()
		return nil, fmt.Errorf("ssh jump: %w", err)
	}
	client, err := handshake(ctx, netConn, addr, sshCfg)
	if err != nil {
		if extra != nil {
			_ = extra.Close()
		}
		_ = hop.Close()
		return nil, err
	}
	return newClient(client, extra, hop), nil
}

func handshake(ctx context.Context, netConn net.Conn, addr string, sshCfg *ssh.ClientConfig) (*ssh.Client, error) {
	type outcome struct {
		conn  ssh.Conn
		chans <-chan ssh.NewChannel
		reqs  <-chan *ssh.Request
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		conn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshCfg)
		done <- outcome{conn, chans, reqs, err}
	}()
	select {
	case <-ctx.Done():
		_ = netConn.Close()
		<-done
		return nil, fmt.Errorf("ssh: %w", ctx.Err())
	case out := <-done:
		if out.err != nil {
			_ = netConn.Close()
			return nil, fmt.Errorf("ssh: %w", out.err)
		}
		return ssh.NewClient(out.conn, out.chans, out.reqs), nil
	}
}

func clientConfig(user string, auth []ssh.AuthMethod) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: user,
		Auth: auth,
		// TODO(hostkey): v1 skips SSH host-key verification. Add known_hosts / TOFU.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
}

// Dial opens a connection to addr through the SSH server.
func (c *Client) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if c == nil || c.ssh == nil {
		return nil, fmt.Errorf("ssh: client is closed")
	}
	conn, err := c.ssh.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	// SSH channels do not implement SetDeadline. MySQL treats that error as
	// "invalid connection" if ReadTimeout/WriteTimeout are set.
	return noDeadlineConn{Conn: conn}, nil
}

type noDeadlineConn struct{ net.Conn }

func (noDeadlineConn) SetDeadline(time.Time) error      { return nil }
func (noDeadlineConn) SetReadDeadline(time.Time) error  { return nil }
func (noDeadlineConn) SetWriteDeadline(time.Time) error { return nil }

func (c *Client) keepAlive() {
	if c == nil || c.ssh == nil || c.stop == nil {
		return
	}
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			_, _, err := c.ssh.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				return
			}
		}
	}
}

// Close ends the SSH session, then any parent jump hop.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	var err error
	c.closed.Do(func() {
		if c.stop != nil {
			close(c.stop)
		}
		if c.ssh != nil {
			err = c.ssh.Close()
		}
		if c.extra != nil {
			_ = c.extra.Close()
		}
		if c.via != nil {
			if e := c.via.Close(); err == nil {
				err = e
			}
		}
	})
	return err
}

func authMethods(cfg Config) ([]ssh.AuthMethod, io.Closer, error) {
	switch cfg.Auth {
	case AuthKey:
		signer, err := loadSigner(cfg.KeyPath, cfg.Passphrase)
		if err != nil {
			return nil, nil, err
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil, nil
	case AuthPassword:
		if cfg.Password == "" {
			return nil, nil, fmt.Errorf("SSH password is required")
		}
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil, nil
	case AuthAgent:
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, nil, fmt.Errorf("SSH agent is not available (SSH_AUTH_SOCK is empty)")
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH agent: %w", err)
		}
		ag := agent.NewClient(conn)
		return []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}, conn, nil
	default:
		return nil, nil, fmt.Errorf("unknown SSH auth method")
	}
}

func loadSigner(path, passphrase string) (ssh.Signer, error) {
	expanded, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	pem, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("SSH private key: %w", err)
	}
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(pem, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("SSH private key: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, fmt.Errorf("SSH private key: %w", err)
	}
	return signer, nil
}

func expandPath(p string) (string, error) {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("SSH private key: %w", err)
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Join(home, p[2:]), nil
	}
	return p, nil
}
