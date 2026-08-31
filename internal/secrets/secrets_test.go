package secrets

import "testing"

func TestAccountKeepsPasswordCompatible(t *testing.T) {
	if got := account("abc", KindPassword); got != "abc" {
		t.Fatalf("password account = %q, want profile id", got)
	}
	if got := account("abc", ""); got != "abc" {
		t.Fatalf("empty kind account = %q, want profile id", got)
	}
	if got := account("abc", KindSSHPassword); got != "abc:sshPassword" {
		t.Fatalf("ssh password account = %q", got)
	}
	if got := account("abc", KindSSHPassphrase); got != "abc:sshPassphrase" {
		t.Fatalf("ssh passphrase account = %q", got)
	}
	if got := account("abc", KindSSHJumpPassword); got != "abc:sshJumpPassword" {
		t.Fatalf("jump password account = %q", got)
	}
	if got := account("abc", KindSSHJumpPassphrase); got != "abc:sshJumpPassphrase" {
		t.Fatalf("jump passphrase account = %q", got)
	}
}
