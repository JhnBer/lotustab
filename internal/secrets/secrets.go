package secrets

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "lotustab"

const (
	KindPassword          = "password"
	KindSSHPassword       = "sshPassword"
	KindSSHPassphrase     = "sshPassphrase"
	KindSSHJumpPassword   = "sshJumpPassword"
	KindSSHJumpPassphrase = "sshJumpPassphrase"
)

// Kinds is every secret slot stored per profile. Delete removes all of them.
var Kinds = []string{KindPassword, KindSSHPassword, KindSSHPassphrase, KindSSHJumpPassword, KindSSHJumpPassphrase}

// Store reads and writes connection secrets in the OS keychain.
type Store struct{}

// New returns a keychain-backed secret store.
func New() *Store {
	return &Store{}
}

func account(profileID, kind string) string {
	if kind == "" || kind == KindPassword {
		return profileID
	}
	return profileID + ":" + kind
}

// Get returns a secret for a profile. Missing entries yield an empty string.
func (s *Store) Get(profileID, kind string) (string, error) {
	if profileID == "" {
		return "", nil
	}
	pw, err := keyring.Get(service, account(profileID, kind))
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("keychain get: %w", err)
	}
	return pw, nil
}

// Set stores a secret. An empty value is ignored so edits can leave it unchanged.
func (s *Store) Set(profileID, kind, value string) error {
	if profileID == "" || value == "" {
		return nil
	}
	if err := keyring.Set(service, account(profileID, kind), value); err != nil {
		return fmt.Errorf("keychain set: %w", err)
	}
	return nil
}

// Delete removes every stored secret for a profile. Missing entries are not an error.
func (s *Store) Delete(profileID string) error {
	if profileID == "" {
		return nil
	}
	var first error
	for _, kind := range Kinds {
		if err := keyring.Delete(service, account(profileID, kind)); err != nil && err != keyring.ErrNotFound {
			if first == nil {
				first = fmt.Errorf("keychain delete: %w", err)
			}
		}
	}
	return first
}
