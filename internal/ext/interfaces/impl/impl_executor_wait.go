package impl

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// WaitProcess waits for a process started via StartProcess/Command to
// finish. A non-zero exit status is returned as *types.ProcessExitError
// so callers can propagate the child's exit code. If timeout > 0 and the
// process is still running when it elapses, the process is killed and a
// timeout error is returned.
func (d DefaultExecutor) WaitProcess(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return fmt.Errorf("process not started")
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	if timeout == 0 {
		err = <-done
	} else {
		select {
		case err = <-done:
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
			<-done // reap the killed process
			return fmt.Errorf("wait timeout after %s: process killed", timeout)
		}
	}

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &types.ProcessExitError{ExitCode: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
