//go:build linux || darwin
// +build linux darwin

package impl

import (
	"errors"
	"os/exec"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/types"
)

func startSh(t *testing.T, script string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("sh", "-c", script)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sh: %v", err)
	}
	return cmd
}

func TestWaitProcess_Success(t *testing.T) {
	ex := DefaultExecutor{}
	if err := ex.WaitProcess(startSh(t, "exit 0"), 0); err != nil {
		t.Errorf("expected nil for exit 0, got %v", err)
	}
}

func TestWaitProcess_NonZeroExitCodePropagated(t *testing.T) {
	ex := DefaultExecutor{}
	err := ex.WaitProcess(startSh(t, "exit 3"), 0)
	var exitErr *types.ProcessExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *types.ProcessExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode != 3 {
		t.Errorf("expected exit code 3, got %d", exitErr.ExitCode)
	}
}

func TestWaitProcess_TimeoutKillsProcess(t *testing.T) {
	ex := DefaultExecutor{}
	cmd := startSh(t, "sleep 30")
	start := time.Now()
	err := ex.WaitProcess(cmd, 100*time.Millisecond)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	var exitErr *types.ProcessExitError
	if errors.As(err, &exitErr) {
		t.Fatalf("timeout must not be reported as exit error, got %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("process was not killed promptly; WaitProcess took %v", elapsed)
	}
	// A SIGKILLed process does not "exit" in ProcessState terms, but
	// Wait must have completed and filled in ProcessState.
	if cmd.ProcessState == nil {
		t.Error("expected killed process to have been reaped (ProcessState set)")
	} else if cmd.ProcessState.Success() {
		t.Error("killed process must not report success")
	}
}
