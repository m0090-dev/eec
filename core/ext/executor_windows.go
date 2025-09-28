// executor_windows.go
// +build windows
//go:build windows
package ext

import (
	"syscall"
	"runtime"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)
// Executor runs commands. Default uses os/exec.
func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File,hideWindow bool) (int, *os.Process, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows の場合、cmd.exe が存在するか確認
		if _, err := exec.LookPath("cmd.exe"); err == nil {
			cmd = exec.Command("cmd.exe", "/C", path+" "+strings.Join(args, " "))
		}
		if hideWindow{
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		}
	case "linux", "darwin":
		// Unix系の場合、sh が存在するか確認
		if _, err := exec.LookPath("sh"); err == nil {
			cmd = exec.Command("sh", "-c", path+" "+strings.Join(args, " "))
		}
	}

	// どちらも存在しない場合はそのまま実行
	if cmd == nil {
		cmd = exec.Command(path, args...)
	}

	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return 0, nil, err
	}
	return cmd.Process.Pid, cmd.Process, nil
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

