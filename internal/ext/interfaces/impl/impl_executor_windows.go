//go:build windows
// +build windows

// executor_windows.go
package impl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// DefaultExecutor uses os/exec
type DefaultExecutor struct{}

// isBatchFile reports whether path refers to a Windows batch script,
// which can only be executed through cmd.exe.
func isBatchFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".bat" || ext == ".cmd"
}

func (d DefaultExecutor) Command(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	// Batch scripts have no executable format of their own, so only
	// those are routed through cmd.exe /C. Everything else is executed
	// directly so argument quoting is not mangled by cmd.exe.
	if isBatchFile(path) {
		cmdArgs := append([]string{"/C", path}, args...)
		cmd = exec.Command("cmd.exe", cmdArgs...)
	} else {
		cmd = exec.Command(path, args...)
	}

	// Windows 専用フラグ
	if hideWindow {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false, CreationFlags: 0x00000010}
	}

	cmd.Env = env
	if stdin != nil {
		cmd.Stdin = stdin
	}
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}

	return cmd, nil
}

// StartProcess は組み立ててから Start() を呼ぶ
func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	cmd, err := d.Command(path, args, env, stdin, stdout, stderr, hideWindow)
	if err != nil {
		return nil, err
	}

	// ここで実際にプロセスを開始する
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (d DefaultExecutor) Getpid() int { return os.Getpid() }

func (d DefaultExecutor) FindProcess(pid int) (*os.Process, error) { return os.FindProcess(pid) }
