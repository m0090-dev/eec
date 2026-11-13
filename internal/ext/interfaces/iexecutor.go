// iexecutor.go
package interfaces

import (
	"os"
	"os/exec"
	"time"
)

// Executor runs commands. Default uses os/exec.
type Executor interface {
	StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (cmd *exec.Cmd, err error)
	Command(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (*exec.Cmd, error)
	WaitProcess(cmd *exec.Cmd, timeout time.Duration) error
	Getpid() int
	FindProcess(pid int) (*os.Process, error)
}
