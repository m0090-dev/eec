//go:build linux || darwin
// +build linux darwin

package impl

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// TestStartProcess_BareNameResolvedAgainstEnvPath is the regression
// test for the 2.0.0 breakage where `eec run -t dev nvim` failed with
// "executable file not found in $PATH%": the tool exists only on the
// PATH of the environment handed to the child, and the executor must
// resolve the bare name against that PATH, not eec's own.
func TestStartProcess_BareNameResolvedAgainstEnvPath(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "eec-envpath-tool")
	script := "#!/bin/sh\nexit 7\n"
	if err := os.WriteFile(tool, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write tool: %v", err)
	}

	ex := DefaultExecutor{}
	env := []string{"PATH=" + dir}
	cmd, err := ex.StartProcess("eec-envpath-tool", nil, env, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("StartProcess failed for env-PATH-only tool: %v", err)
	}
	err = ex.WaitProcess(cmd, 0)
	var exitErr *types.ProcessExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode != 7 {
		t.Fatalf("expected exit code 7 from the resolved tool, got %v", err)
	}
}

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
