//go:build linux || darwin
// +build linux darwin

// executor_unix.go
package impl

import (
	"os"
	"os/exec"
)

// DefaultExecutor uses os/exec
type DefaultExecutor struct{}

func (d DefaultExecutor) Command(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	cmd = exec.Command(path, args...)
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

// StartProcess: Command を呼んでから Start する
func (d DefaultExecutor) StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error) {
	cmd, err := d.Command(path, args, env, stdin, stdout, stderr, hideWindow)
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (d DefaultExecutor) Getpid() int { return os.Getpid() }

func (d DefaultExecutor) FindProcess(pid int) (*os.Process, error) { return os.FindProcess(pid) }
