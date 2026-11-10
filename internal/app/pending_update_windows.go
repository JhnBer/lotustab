//go:build windows

package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const processQueryLimitedInformation = 0x1000

func detach(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	const (
		detachedProcess    = 0x00000008
		createNoWindow     = 0x08000000
		createNewProcGroup = 0x00000200
	)
	cmd.SysProcAttr.CreationFlags |= detachedProcess | createNoWindow | createNewProcGroup
	cmd.SysProcAttr.HideWindow = true
}

func commandDetached(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.SysProcAttr == nil {
		return false
	}
	const detachedProcess = 0x00000008
	return cmd.SysProcAttr.HideWindow && cmd.SysProcAttr.CreationFlags&detachedProcess != 0
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = syscall.CloseHandle(handle)
	return true
}

func sameDevice(a, b string) (bool, error) {
	if _, err := os.Stat(a); err != nil {
		return false, err
	}
	if _, err := os.Stat(b); err != nil {
		return false, err
	}
	return strings.EqualFold(filepath.VolumeName(a), filepath.VolumeName(b)), nil
}
