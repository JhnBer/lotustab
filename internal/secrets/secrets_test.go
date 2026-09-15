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

func TestGitHubTokenAccountIsNotAProfileSlot(t *testing.T) {
	if githubTokenAccount != "app:githubToken" {
		t.Fatalf("github token account = %q", githubTokenAccount)
	}
	for _, kind := range Kinds {
		if account("app", kind) == githubTokenAccount {
			t.Fatalf("profile kind %q collides with github token account", kind)
		}
	}
}
