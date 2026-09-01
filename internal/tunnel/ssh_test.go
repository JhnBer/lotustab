package tunnel

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigFromParamsDisabled(t *testing.T) {
	_, ok, err := ConfigFromParams(nil, "", "")
	if err != nil || ok {
		t.Fatalf("nil params: ok=%v err=%v", ok, err)
	}
	_, ok, err = ConfigFromParams(map[string]string{ParamEnabled: "false"}, "", "")
	if err != nil || ok {
		t.Fatalf("disabled: ok=%v err=%v", ok, err)
	}
}

func TestConfigFromParamsRequiresHostUserKey(t *testing.T) {
	_, ok, err := ConfigFromParams(map[string]string{
		ParamEnabled: "true",
		ParamAuth:    AuthKey,
	}, "", "")
	if !ok || err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("expected hostname error, ok=%v err=%v", ok, err)
	}

	_, _, err = ConfigFromParams(map[string]string{
		ParamEnabled: "true",
		ParamHost:    "bastion",
		ParamAuth:    AuthKey,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "username") {
		t.Fatalf("expected username error, got %v", err)
	}

	_, _, err = ConfigFromParams(map[string]string{
		ParamEnabled: "true",
		ParamHost:    "bastion",
		ParamUser:    "ubuntu",
		ParamAuth:    AuthKey,
	}, "", "")
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("expected key path error, got %v", err)
	}
}

func TestConfigFromParamsDefaults(t *testing.T) {
	cfg, ok, err := ConfigFromParams(map[string]string{
		ParamEnabled: "true",
		ParamHost:    "bastion",
		ParamUser:    "ubuntu",
		ParamKeyPath: "~/.ssh/id_ed25519",
	}, "pw", "ph")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("port = %q, want %s", cfg.Port, DefaultPort)
	}
	if cfg.Auth != AuthKey {
		t.Fatalf("auth = %q, want %s", cfg.Auth, AuthKey)
	}
	if cfg.Password != "pw" || cfg.Passphrase != "ph" {
		t.Fatalf("secrets not copied: %+v", cfg)
	}
}

func TestNoDeadlineConnIgnoresDeadlines(t *testing.T) {
	a, b := net.Pipe()
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	c := noDeadlineConn{Conn: a}
	if err := c.SetDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := c.SetReadDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := c.SetWriteDeadline(time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestExpandPathHome(t *testing.T) {
	got, err := expandPath("~/.ssh/id_ed25519")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) || !strings.HasSuffix(got, filepath.Join(".ssh", "id_ed25519")) {
		t.Fatalf("expanded path = %q", got)
	}
	plain, err := expandPath("/tmp/key")
	if err != nil || plain != "/tmp/key" {
		t.Fatalf("plain path = %q err=%v", plain, err)
	}
}

func TestJumpConfigFromParamsDisabled(t *testing.T) {
	_, ok, err := JumpConfigFromParams(nil, "", "")
	if err != nil || ok {
		t.Fatalf("nil params: ok=%v err=%v", ok, err)
	}
	_, ok, err = JumpConfigFromParams(map[string]string{
		ParamEnabled:     "true",
		ParamJumpEnabled: "false",
	}, "", "")
	if err != nil || ok {
		t.Fatalf("jump off: ok=%v err=%v", ok, err)
	}
	_, ok, err = JumpConfigFromParams(map[string]string{
		ParamEnabled:     "false",
		ParamJumpEnabled: "true",
		ParamJumpHost:    "jump",
		ParamJumpUser:    "ubuntu",
		ParamJumpKeyPath: "~/.ssh/id_ed25519",
	}, "", "")
	if err != nil || ok {
		t.Fatalf("ssh off: ok=%v err=%v", ok, err)
	}
}

func TestJumpConfigFromParamsRequiresHostUserKey(t *testing.T) {
	base := map[string]string{
		ParamEnabled:     "true",
		ParamJumpEnabled: "true",
		ParamJumpAuth:    AuthKey,
	}
	_, ok, err := JumpConfigFromParams(base, "", "")
	if !ok || err == nil || !strings.Contains(err.Error(), "hostname") {
		t.Fatalf("expected hostname error, ok=%v err=%v", ok, err)
	}

	base[ParamJumpHost] = "jump"
	_, _, err = JumpConfigFromParams(base, "", "")
	if err == nil || !strings.Contains(err.Error(), "username") {
		t.Fatalf("expected username error, got %v", err)
	}

	base[ParamJumpUser] = "ubuntu"
	_, _, err = JumpConfigFromParams(base, "", "")
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("expected key path error, got %v", err)
	}
}

func TestJumpConfigFromParamsDefaults(t *testing.T) {
	cfg, ok, err := JumpConfigFromParams(map[string]string{
		ParamEnabled:     "true",
		ParamJumpEnabled: "true",
		ParamJumpHost:    "jump",
		ParamJumpUser:    "ubuntu",
		ParamJumpKeyPath: "~/.ssh/id_ed25519",
	}, "pw", "ph")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if cfg.Port != DefaultPort {
		t.Fatalf("port = %q, want %s", cfg.Port, DefaultPort)
	}
	if cfg.Host != "jump" || cfg.Password != "pw" || cfg.Passphrase != "ph" {
		t.Fatalf("config = %+v", cfg)
	}
}

type trackClose struct{ closed bool }

func (t *trackClose) Close() error {
	t.closed = true
	return nil
}

func TestClientCloseClosesVia(t *testing.T) {
	via := &trackClose{}
	extra := &trackClose{}
	c := &Client{extra: extra, via: via}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !via.closed || !extra.closed {
		t.Fatalf("via=%v extra=%v", via.closed, extra.closed)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenViaClosesHopOnError(t *testing.T) {
	hopExtra := &trackClose{}
	hop := &Client{extra: hopExtra}
	_, err := OpenVia(context.Background(), hop, Config{
		Host: "db.internal",
		Port: DefaultPort,
		User: "ubuntu",
		Auth: AuthPassword,
	})
	if err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected password error, got %v", err)
	}
	if !hopExtra.closed {
		t.Fatal("hop was not closed after OpenVia failure")
	}
}

func TestOpenViaNilHop(t *testing.T) {
	_, err := OpenVia(context.Background(), nil, Config{Host: "x", Port: DefaultPort, User: "u", Auth: AuthPassword, Password: "p"})
	if err == nil || !strings.Contains(err.Error(), "missing hop") {
		t.Fatalf("expected missing hop error, got %v", err)
	}
}

func TestOpenCancelsDuringHandshake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	hold := make(chan struct{})
	t.Cleanup(func() {
		close(hold)
		_ = ln.Close()
	})
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		<-hold
	}()

	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err = Open(ctx, Config{
		Host:     "127.0.0.1",
		Port:     port,
		User:     "ubuntu",
		Auth:     AuthPassword,
		Password: "secret",
	})
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline error, got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatalf("cancel took too long: %s", time.Since(start))
	}
}
