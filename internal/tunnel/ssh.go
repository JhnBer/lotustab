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

	AuthKey      = "key"
	AuthPassword = "password"
	AuthAgent    = "agent"

	DefaultPort = "22"
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
	auth := strings.TrimSpace(params[ParamAuth])
	if auth == "" {
		auth = AuthKey
	}
	cfg := Config{
		Host:       strings.TrimSpace(params[ParamHost]),
		Port:       strings.TrimSpace(params[ParamPort]),
		User:       strings.TrimSpace(params[ParamUser]),
		Auth:       auth,
		KeyPath:    strings.TrimSpace(params[ParamKeyPath]),
		Passphrase: sshPassphrase,
		Password:   sshPassword,
	}
	if cfg.Port == "" {
		cfg.Port = DefaultPort
	}
	if cfg.Host == "" {
		return Config{}, true, fmt.Errorf("SSH hostname is required")
	}
	if _, err := strconv.ParseUint(cfg.Port, 10, 16); err != nil {
		return Config{}, true, fmt.Errorf("invalid SSH port")
	}
	if cfg.User == "" {
		return Config{}, true, fmt.Errorf("SSH username is required")
	}
	switch cfg.Auth {
	case AuthKey:
		if cfg.KeyPath == "" {
			return Config{}, true, fmt.Errorf("SSH private key file is required")
		}
	case AuthPassword, AuthAgent:
	default:
		return Config{}, true, fmt.Errorf("unknown SSH auth method")
	}
	return cfg, true, nil
}

// Client is an open SSH session that can dial through to a destination.
type Client struct {
	ssh   *ssh.Client
	extra io.Closer
}

// Open dials the SSH server. The caller must Close the client.
func Open(cfg Config) (*Client, error) {
	auth, extra, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	sshCfg := &ssh.ClientConfig{
		User: cfg.User,
		Auth: auth,
		// TODO(hostkey): v1 skips SSH host-key verification. Add known_hosts / TOFU.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	conn, err := ssh.Dial("tcp", net.JoinHostPort(cfg.Host, cfg.Port), sshCfg)
	if err != nil {
		if extra != nil {
			_ = extra.Close()
		}
		return nil, fmt.Errorf("ssh: %w", err)
	}
	return &Client{ssh: conn, extra: extra}, nil
}

// Dial opens a connection to addr through the SSH server.
func (c *Client) Dial(ctx context.Context, network, addr string) (net.Conn, error) {
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

// Close ends the SSH session.
func (c *Client) Close() error {
	err := c.ssh.Close()
	if c.extra != nil {
		_ = c.extra.Close()
	}
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
