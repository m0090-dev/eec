//go:build windows
// +build windows

// executor_windows.go
package impl

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	//"strings"
	"syscall"
	"time"
)

// DefaultExecutor uses os/exec
type DefaultExecutor struct{}

func (d DefaultExecutor) Executable() (string, error) {
	return os.Executable()
}

func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		if _, err := exec.LookPath("cmd.exe"); err == nil {
			cmdArgs := append([]string{"/C", path}, args...)
			cmd = exec.Command("cmd.exe", cmdArgs...)
			if hideWindow {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			} else {
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false, CreationFlags: 0x00000010}
			}
		} else {
			return nil, fmt.Errorf("cmd.exe not found in PATH")
		}
	case "linux", "darwin":
		if _, err := exec.LookPath("sh"); err == nil {
			cmdArgs := append([]string{"-c", path}, args...)
			cmd = exec.Command("sh", cmdArgs...)
		} else {
			return nil, fmt.Errorf("sh not found in PATH")
		}
	default:
		cmd = exec.Command(path, args...)
	}

	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}




func (d DefaultExecutor) WaitProcess(proc *os.Process, timeout time.Duration) error {
	// We need the *Cmd to call Wait; but we only have *os.Process here.
	// Simpler: poll process state.
	done := make(chan error, 1)
	go func() {
		_, err := proc.Wait()
		done <- err
	}()
	if timeout == 0 {
		return <-done
	}
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("wait timeout after %s", timeout)
	}
}
func (d DefaultExecutor) Getpid() int{return os.Getpid()}

func (d DefaultExecutor) FindProcess(pid int) (*os.Process,error) {return os.FindProcess(pid)}
