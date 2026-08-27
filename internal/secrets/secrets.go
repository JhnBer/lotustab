package secrets

import (
	"fmt"

	"github.com/zalando/go-keyring"
)

const service = "lotustab"

// Store reads and writes connection passwords in the OS keychain.
type Store struct{}

// New returns a keychain-backed secret store.
func New() *Store {
	return &Store{}
}

// Get returns the password for a profile. Missing entries yield an empty string.
func (s *Store) Get(profileID string) (string, error) {
	if profileID == "" {
		return "", nil
	}
	pw, err := keyring.Get(service, profileID)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", nil
		}
		return "", fmt.Errorf("keychain get: %w", err)
	}
	return pw, nil
}

// Set stores a password. An empty password is ignored so edits can leave it unchanged.
func (s *Store) Set(profileID, password string) error {
	if profileID == "" || password == "" {
		return nil
	}
	if err := keyring.Set(service, profileID, password); err != nil {
		return fmt.Errorf("keychain set: %w", err)
	}
	return nil
}

// Delete removes a stored password. Missing entries are not an error.
func (s *Store) Delete(profileID string) error {
	if profileID == "" {
		return nil
	}
	if err := keyring.Delete(service, profileID); err != nil && err != keyring.ErrNotFound {
		return fmt.Errorf("keychain delete: %w", err)
	}
	return nil
}
