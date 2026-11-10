package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Wails updater helper protocol. The names are not exported by the updater
// package; Restart sets the same variables before it swaps the bundle.
const (
	envHelperMode   = "WAILS_UPDATER_HELPER"
	envHelperTarget = "WAILS_UPDATER_HELPER_TARGET"
	envHelperNew    = "WAILS_UPDATER_HELPER_NEW"
	envHelperPID    = "WAILS_UPDATER_HELPER_PID"
	envHelperLog    = "WAILS_UPDATER_HELPER_LOG"
	envWaitPID      = "LOTUSTAB_WAIT_PID"

	appConfigDir   = "lotustab"
	pendingDirName = "pending-update"
	manifestName   = "pending.json"
)

var (
	errBrokenPending = errors.New("pending update manifest is invalid")

	relaunchWaitTimeout = 30 * time.Second
	relaunchPoll        = 100 * time.Millisecond

	startCommand = func(cmd *exec.Cmd) error { return cmd.Start() }
	quitApp      = func() {
		if appl := application.Get(); appl != nil {
			appl.Quit()
		}
	}
	sameDeviceFn = sameDevice
)

type pendingManifest struct {
	Version  string `json:"version"`
	Artifact string `json:"artifact"`
}

// ApplyPendingUpdate starts the Wails updater helper when a newer staged
// build is waiting. It reports whether the caller should exit so the helper
// can swap the bundle. A missing or stale manifest is removed and startup
// continues.
func ApplyPendingUpdate(currentVersion string) bool {
	waitForRelaunchParent()
	root, err := pendingRoot()
	if err != nil {
		log.Printf("updater: %v", err)
		return false
	}
	self, err := os.Executable()
	if err != nil {
		log.Printf("updater: %v", err)
		return false
	}
	exit, err := applyPending(root, currentVersion, self, startCommand)
	if err != nil {
		log.Printf("updater: %v", err)
	}
	return exit
}

// Relaunch starts this executable again and asks the running app to quit.
// The new process applies a stored update the same way a later manual open does.
func Relaunch() error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("relaunch: %w", err)
	}
	cmd := detachedCommand(self)
	cmd.Env = append(withoutUpdaterEnv(os.Environ()), envWaitPID+"="+strconv.Itoa(os.Getpid()))
	if err := startCommand(cmd); err != nil {
		return fmt.Errorf("relaunch: %w", err)
	}
	quitApp()
	return nil
}

func waitForRelaunchParent() {
	raw := os.Getenv(envWaitPID)
	if raw == "" {
		return
	}
	_ = os.Unsetenv(envWaitPID)
	pid, err := strconv.Atoi(raw)
	if err != nil || pid <= 0 || pid == os.Getpid() {
		return
	}
	deadline := time.Now().Add(relaunchWaitTimeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(relaunchPoll)
	}
	log.Printf("updater: previous process %d still running; applying update anyway", pid)
}

func pendingRoot() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, appConfigDir, pendingDirName), nil
}

func stagePending(root, version, src string) error {
	version = strings.TrimSpace(version)
	src = filepath.Clean(src)
	base := filepath.Base(src)
	if version == "" || base == "." || base == ".." || base == string(os.PathSeparator) {
		return fmt.Errorf("pending update: missing version or artifact")
	}
	if _, err := os.Lstat(src); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	tmp, err := os.MkdirTemp(root, ".copy-")
	if err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	defer os.RemoveAll(tmp)
	copied := filepath.Join(tmp, base)
	if err := copyAny(src, copied); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	final := filepath.Join(root, base)
	if err := os.RemoveAll(final); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	if err := os.Rename(copied, final); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	if err := writeManifest(root, pendingManifest{Version: version, Artifact: final}); err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == manifestName || name == base {
			continue
		}
		_ = os.RemoveAll(filepath.Join(root, name))
	}
	return nil
}

func applyPending(root, current, self string, spawn func(*exec.Cmd) error) (bool, error) {
	man, err := loadPending(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if errors.Is(err, errBrokenPending) {
			_ = os.RemoveAll(root)
			return false, nil
		}
		return false, err
	}
	if !releaseIsNewer(man.Version, current) || !withinDir(root, man.Artifact) {
		_ = os.RemoveAll(root)
		return false, nil
	}
	if _, err := os.Lstat(man.Artifact); err != nil {
		_ = os.RemoveAll(root)
		return false, nil
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false, fmt.Errorf("pending update: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(man.Artifact)
	if err != nil || !withinDir(resolvedRoot, resolved) {
		_ = os.RemoveAll(root)
		return false, nil
	}
	target := bundleTarget(self)
	payload, err := payloadForSwap(man.Artifact, target)
	if err != nil {
		return false, err
	}
	cmd := helperCommand(self, target, payload, os.Getpid())
	if err := spawn(cmd); err != nil {
		return false, fmt.Errorf("start update helper: %w", err)
	}
	return true, nil
}

func loadPending(root string) (pendingManifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, manifestName))
	if err != nil {
		return pendingManifest{}, err
	}
	var man pendingManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return pendingManifest{}, errBrokenPending
	}
	man.Version = strings.TrimSpace(man.Version)
	man.Artifact = strings.TrimSpace(man.Artifact)
	if man.Version == "" || man.Artifact == "" {
		return pendingManifest{}, errBrokenPending
	}
	return man, nil
}

func writeManifest(root string, man pendingManifest) error {
	raw, err := json.Marshal(man)
	if err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	path := filepath.Join(root, manifestName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("pending update: %w", err)
	}
	return nil
}

func payloadForSwap(artifact, target string) (string, error) {
	same, err := sameDeviceFn(artifact, target)
	if err != nil {
		return "", fmt.Errorf("pending update: %w", err)
	}
	if same {
		return artifact, nil
	}
	dest := filepath.Join(filepath.Dir(target), filepath.Base(artifact)+".pending")
	if filepath.Clean(dest) == filepath.Clean(target) {
		return "", fmt.Errorf("pending update: swap path collides with the installed app")
	}
	if err := os.RemoveAll(dest); err != nil {
		return "", fmt.Errorf("pending update: %w", err)
	}
	if err := copyAny(artifact, dest); err != nil {
		_ = os.RemoveAll(dest)
		return "", fmt.Errorf("pending update: %w", err)
	}
	return dest, nil
}

func bundleTarget(exe string) string {
	if runtime.GOOS != "darwin" {
		return exe
	}
	return darwinBundle(exe)
}

func darwinBundle(exe string) string {
	parts := strings.Split(filepath.Clean(exe), string(os.PathSeparator))
	for i, part := range parts {
		if !strings.HasSuffix(part, ".app") {
			continue
		}
		if i == 0 {
			return parts[0]
		}
		return string(os.PathSeparator) + filepath.Join(parts[1:i+1]...)
	}
	return exe
}

func withinDir(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func helperCommand(self, target, newPath string, pid int) *exec.Cmd {
	cmd := detachedCommand(self)
	cmd.Env = append(withoutUpdaterEnv(os.Environ()),
		envHelperMode+"=1",
		envHelperTarget+"="+target,
		envHelperNew+"="+newPath,
		envHelperPID+"="+strconv.Itoa(pid),
		envHelperLog+"="+filepath.Join(os.TempDir(), fmt.Sprintf("wails-update-%d.log", pid)),
	)
	return cmd
}

func detachedCommand(path string) *exec.Cmd {
	cmd := exec.Command(path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	detach(cmd)
	return cmd
}

func withoutUpdaterEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "WAILS_UPDATER_HELPER") || strings.HasPrefix(entry, envWaitPID+"=") {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func copyAny(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		link, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(link, dst)
	}
	if info.IsDir() {
		return copyTree(src, dst, info.Mode())
	}
	return copyFile(src, dst, info.Mode())
}

func copyTree(src, dst string, mode os.FileMode) error {
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o755
	}
	if err := os.MkdirAll(dst, perm); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := copyAny(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	perm := mode.Perm()
	if perm == 0 {
		perm = 0o644
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}
