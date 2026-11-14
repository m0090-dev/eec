package types

import "fmt"

// ProcessExitError reports that a child process started by eec exited
// with a non-zero exit code. It carries the code so callers (notably the
// CLI) can exit with the same code instead of collapsing every failure
// to exit code 1.
type ProcessExitError struct {
	ExitCode int
}

func (e *ProcessExitError) Error() string {
	return fmt.Sprintf("process exited with code %d", e.ExitCode)
}
