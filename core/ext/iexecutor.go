// iexecutor.go
package ext
import (
	"os"
	"time"
)
// Executor runs commands. Default uses os/exec.
type Executor interface {
	//LookPath(name string) (string, error)
	StartProcess(path string, args []string, env []string, stdin, stdout, stderr *os.File,hideWindow bool) (pid int, proc *os.Process, err error)
	WaitProcess(proc *os.Process, timeout time.Duration) error
	Getpid() int
}

// DefaultExecutor uses os/exec
type DefaultExecutor struct{}


