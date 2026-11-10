package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStagePendingCopiesFileAndDirectory(t *testing.T) {
	src := t.TempDir()
	appDir := filepath.Join(src, "LotusTab.app")
	binDir := filepath.Join(appDir, "Contents", "MacOS")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "LotusTab"), []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("LotusTab", filepath.Join(binDir, "LotusTab.link")); err != nil {
		t.Fatal(err)
	}

	root := filepath.Join(t.TempDir(), "pending-update")
	if err := stagePending(root, "1.2.3", appDir); err != nil {
		t.Fatal(err)
	}
	man, err := loadPending(root)
	if err != nil {
		t.Fatal(err)
	}
	if man.Version != "1.2.3" {
		t.Fatalf("version: got %q", man.Version)
	}
	if !withinDir(root, man.Artifact) {
		t.Fatalf("artifact %q is outside %q", man.Artifact, root)
	}
	got, err := os.ReadFile(filepath.Join(man.Artifact, "Contents", "MacOS", "LotusTab"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("binary: got %q", got)
	}
	link, err := os.Readlink(filepath.Join(man.Artifact, "Contents", "MacOS", "LotusTab.link"))
	if err != nil {
		t.Fatal(err)
	}
	if link != "LotusTab" {
		t.Fatalf("symlink: got %q", link)
	}
}

func TestApplyPendingNewerVersionSpawnsHelper(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "LotusTab.app")
	if err := os.Mkdir(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, pendingManifest{Version: "1.2.3", Artifact: artifact}); err != nil {
		t.Fatal(err)
	}
	self := "/Applications/LotusTab.app/Contents/MacOS/LotusTab"
	prev := sameDeviceFn
	sameDeviceFn = func(string, string) (bool, error) { return true, nil }
	t.Cleanup(func() { sameDeviceFn = prev })
	var spawned *exec.Cmd
	exit, err := applyPending(root, "1.2.0", self, func(cmd *exec.Cmd) error {
		spawned = cmd
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !exit {
		t.Fatal("expected the parent path to exit")
	}
	if spawned == nil {
		t.Fatal("helper was not spawned")
	}
	if !commandDetached(spawned) {
		t.Fatal("helper was not detached")
	}
	if spawned.Stdin != nil || spawned.Stdout != nil || spawned.Stderr != nil {
		t.Fatal("helper stdio is attached")
	}
	assertEnv := func(key, want string) {
		t.Helper()
		if got := envValue(spawned, key); got != want {
			t.Fatalf("%s: got %q, want %q", key, got, want)
		}
	}
	assertEnv(envHelperMode, "1")
	assertEnv(envHelperTarget, bundleTarget(self))
	assertEnv(envHelperNew, artifact)
	assertEnv(envHelperPID, strconv.Itoa(os.Getpid()))
	if !strings.Contains(envValue(spawned, envHelperLog), "wails-update-") {
		t.Fatalf("log path: %q", envValue(spawned, envHelperLog))
	}
	if _, err := os.Stat(filepath.Join(root, manifestName)); err != nil {
		t.Fatal("manifest should stay until the new version starts")
	}
}

func TestApplyPendingClearsStaleOrBrokenManifest(t *testing.T) {
	t.Run("broken", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, manifestName), []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		assertPendingCleared(t, root, "1.0.0")
	})
	t.Run("not newer", func(t *testing.T) {
		root := t.TempDir()
		artifact := filepath.Join(root, "LotusTab.app")
		if err := os.WriteFile(artifact, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := writeManifest(root, pendingManifest{Version: "1.0.0", Artifact: artifact}); err != nil {
			t.Fatal(err)
		}
		assertPendingCleared(t, root, "1.0.0")
	})
	t.Run("missing artifact", func(t *testing.T) {
		root := t.TempDir()
		if err := writeManifest(root, pendingManifest{Version: "9.0.0", Artifact: filepath.Join(root, "gone")}); err != nil {
			t.Fatal(err)
		}
		assertPendingCleared(t, root, "1.0.0")
	})
	t.Run("outside root", func(t *testing.T) {
		root := t.TempDir()
		outside := filepath.Join(t.TempDir(), "other")
		if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := writeManifest(root, pendingManifest{Version: "9.0.0", Artifact: outside}); err != nil {
			t.Fatal(err)
		}
		assertPendingCleared(t, root, "1.0.0")
		if _, err := os.Stat(outside); err != nil {
			t.Fatal("outside file should be left in place")
		}
	})
}

func TestApplyPendingRejectsSymlinkOutsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "payload")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, pendingManifest{Version: "9.0.0", Artifact: link}); err != nil {
		t.Fatal(err)
	}
	assertPendingCleared(t, root, "1.0.0")
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
}

func TestApplyPendingSpawnErrorKeepsManifest(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "build")
	if err := os.WriteFile(artifact, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeManifest(root, pendingManifest{Version: "2.0.0", Artifact: artifact}); err != nil {
		t.Fatal(err)
	}
	exit, err := applyPending(root, "1.0.0", artifact, func(*exec.Cmd) error {
		return os.ErrPermission
	})
	if err == nil {
		t.Fatal("expected spawn error")
	}
	if exit {
		t.Fatal("spawn failure should keep the current process")
	}
	if _, err := loadPending(root); err != nil {
		t.Fatal(err)
	}
}

func TestPayloadForSwapCopiesOntoTargetDevice(t *testing.T) {
	root := t.TempDir()
	artifact := filepath.Join(root, "LotusTab.app")
	if err := os.Mkdir(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifact, "Contents"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(t.TempDir(), "Applications")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "LotusTab.app")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	prev := sameDeviceFn
	sameDeviceFn = func(string, string) (bool, error) { return false, nil }
	t.Cleanup(func() { sameDeviceFn = prev })

	got, err := payloadForSwap(artifact, target)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(targetDir, "LotusTab.app.pending")
	if got != want {
		t.Fatalf("payload: got %q, want %q", got, want)
	}
	body, err := os.ReadFile(filepath.Join(got, "Contents"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "bin" {
		t.Fatalf("copied payload: %q", body)
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatal("source artifact should stay in the pending directory")
	}
}

func TestRelaunchDoesNotUseHelperMode(t *testing.T) {
	var spawned *exec.Cmd
	quit := false
	prevStart := startCommand
	prevQuit := quitApp
	startCommand = func(cmd *exec.Cmd) error {
		spawned = cmd
		return nil
	}
	quitApp = func() { quit = true }
	t.Cleanup(func() {
		startCommand = prevStart
		quitApp = prevQuit
	})

	if err := Relaunch(); err != nil {
		t.Fatal(err)
	}
	if spawned == nil {
		t.Fatal("relaunch did not start a process")
	}
	if !quit {
		t.Fatal("relaunch did not quit the current app")
	}
	if !commandDetached(spawned) {
		t.Fatal("relaunch was not detached")
	}
	if envValue(spawned, envHelperMode) != "" || envValue(spawned, envHelperTarget) != "" || envValue(spawned, envHelperNew) != "" {
		t.Fatalf("relaunch entered helper mode: %v", spawned.Env)
	}
	if envValue(spawned, envWaitPID) != strconv.Itoa(os.Getpid()) {
		t.Fatalf("wait pid: %q", envValue(spawned, envWaitPID))
	}
}

func TestWaitForRelaunchParentClearsEnv(t *testing.T) {
	pid := os.Getpid() + 100000
	if processAlive(pid) {
		t.Fatal("expected an unused pid")
	}
	prevTimeout := relaunchWaitTimeout
	prevPoll := relaunchPoll
	relaunchWaitTimeout = 200 * time.Millisecond
	relaunchPoll = time.Millisecond
	t.Cleanup(func() {
		relaunchWaitTimeout = prevTimeout
		relaunchPoll = prevPoll
	})
	t.Setenv(envWaitPID, strconv.Itoa(pid))
	waitForRelaunchParent()
	if os.Getenv(envWaitPID) != "" {
		t.Fatal("wait pid env still set")
	}
}

func TestDarwinBundleTarget(t *testing.T) {
	exe := string(os.PathSeparator) + filepath.Join("Applications", "LotusTab.app", "Contents", "MacOS", "LotusTab")
	want := string(os.PathSeparator) + filepath.Join("Applications", "LotusTab.app")
	if got := darwinBundle(exe); got != want {
		t.Fatalf("bundle: got %q, want %q", got, want)
	}
	plain := string(os.PathSeparator) + filepath.Join("usr", "local", "bin", "lotustab")
	if got := darwinBundle(plain); got != plain {
		t.Fatalf("plain binary: got %q", got)
	}
}

func assertPendingCleared(t *testing.T, root, current string) {
	t.Helper()
	exit, err := applyPending(root, current, "/usr/local/bin/lotustab", func(*exec.Cmd) error {
		t.Fatal("helper should not spawn")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if exit {
		t.Fatal("stale pending should not exit")
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("pending dir still present: %v", err)
	}
}

func envValue(cmd *exec.Cmd, key string) string {
	prefix := key + "="
	for _, entry := range cmd.Env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}
