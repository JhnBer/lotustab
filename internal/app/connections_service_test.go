package app

import (
	"strings"
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/postgres"
	"lotustab/internal/secrets"
	"lotustab/internal/tunnel"
)

func testConnectionsService(t *testing.T) *ConnectionsService {
	t.Helper()
	reg := driver.NewRegistry()
	if err := reg.Register(postgres.New()); err != nil {
		t.Fatal(err)
	}
	return NewConnectionsService(reg, nil, nil, nil)
}

func TestPrepareStripsSecrets(t *testing.T) {
	s := testConnectionsService(t)
	_, profile, secretVals, err := s.prepare(SaveProfileInput{
		Name:     "prod",
		DriverID: "postgres",
		Params: map[string]string{
			"host":          "db.internal",
			"port":          "5432",
			"user":          "app",
			"password":      "db-secret",
			"sshEnabled":    "true",
			"sshHost":       "bastion",
			"sshPort":       "22",
			"sshUser":       "ubuntu",
			"sshAuth":       tunnel.AuthKey,
			"sshKeyPath":    "~/.ssh/id_ed25519",
			"sshPassword":   "ssh-secret",
			"sshPassphrase": "key-ph",
		},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range secrets.Kinds {
		if _, ok := profile.Params[kind]; ok {
			t.Fatalf("secret %s leaked into params", kind)
		}
	}
	if secretVals[secrets.KindPassword] != "db-secret" {
		t.Fatalf("password = %q", secretVals[secrets.KindPassword])
	}
	if secretVals[secrets.KindSSHPassword] != "ssh-secret" {
		t.Fatalf("ssh password = %q", secretVals[secrets.KindSSHPassword])
	}
	if secretVals[secrets.KindSSHPassphrase] != "key-ph" {
		t.Fatalf("passphrase = %q", secretVals[secrets.KindSSHPassphrase])
	}
	if profile.Params[tunnel.ParamHost] != "bastion" {
		t.Fatalf("ssh host not persisted: %v", profile.Params)
	}
}

func TestPrepareRequiresSSHPasswordOnCreate(t *testing.T) {
	s := testConnectionsService(t)
	_, _, _, err := s.prepare(SaveProfileInput{
		Name:     "prod",
		DriverID: "postgres",
		Params: map[string]string{
			"host":       "db.internal",
			"port":       "5432",
			"user":       "app",
			"sshEnabled": "true",
			"sshHost":    "bastion",
			"sshUser":    "ubuntu",
			"sshAuth":    tunnel.AuthPassword,
		},
	}, true)
	if err == nil || !strings.Contains(err.Error(), "SSH password") {
		t.Fatalf("expected SSH password error, got %v", err)
	}
}

func TestPrepareAllowsEmptySSHPasswordOnEdit(t *testing.T) {
	s := testConnectionsService(t)
	_, _, _, err := s.prepare(SaveProfileInput{
		ID:       "existing-id",
		Name:     "prod",
		DriverID: "postgres",
		Params: map[string]string{
			"host":       "db.internal",
			"port":       "5432",
			"user":       "app",
			"sshEnabled": "true",
			"sshHost":    "bastion",
			"sshUser":    "ubuntu",
			"sshAuth":    tunnel.AuthPassword,
		},
	}, true)
	if err != nil {
		t.Fatal(err)
	}
}
