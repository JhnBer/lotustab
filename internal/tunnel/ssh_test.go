package tunnel

import (
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
