// iexecutor.go
package interfaces
import (
	"os"
	"time"
	"os/exec"
	"github.com/aymanbagabas/go-pty"
)
// Executor runs commands. Default uses os/exec.
type Executor interface {
	//LookPath(name string) (string, error)
	StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File,hideWindow bool) (cmd *exec.Cmd, err error)
	WaitProcess(proc *os.Process, timeout time.Duration) error
	Getpid() int
       /* StartProcessWithCmd(path string, args []string, env []string, stdin, stdout, stderr *os.File, hideWindow bool) (pid int, proc *os.Process, cmd *exec.Cmd, err error)*/
	FindProcess(pid int) (*os.Process,error)
	// StartProcessPty(shell string, args []string) (*pty.Cmd, pty.Pty, error)
	// RestartProcessPty(p pty.Pty, shell string, args []string) (*pty.Cmd, error)
	StartProcessPty(shell string, args []string, env []string) (*pty.Cmd, pty.Pty, error)
	RestartProcessPty(p pty.Pty, shell string, args []string, env []string) (*pty.Cmd, error)
}

