package domain_test

import (
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

func TestInteractiveShell_Unix(t *testing.T) {
	rt := newMockRuntime() // goos = linux
	if got := domain.InteractiveShell(rt, nil); got != "sh" {
		t.Errorf("default = %q, want sh", got)
	}
	rt.env.lookupMap["SHELL"] = "/bin/bash"
	if got := domain.InteractiveShell(rt, nil); got != "/bin/bash" {
		t.Errorf("process SHELL = %q, want /bin/bash", got)
	}
	// The resolved environment wins over the process environment.
	if got := domain.InteractiveShell(rt, []string{"SHELL=/bin/zsh"}); got != "/bin/zsh" {
		t.Errorf("resolved SHELL = %q, want /bin/zsh", got)
	}
}

func TestInteractiveShell_Windows(t *testing.T) {
	rt := newMockRuntime()
	rt.env.goos = "windows"
	if got := domain.InteractiveShell(rt, nil); got != "cmd" {
		t.Errorf("default = %q, want cmd", got)
	}
	rt.env.lookupMap["ComSpec"] = `C:\Windows\System32\cmd.exe`
	if got := domain.InteractiveShell(rt, nil); got != `C:\Windows\System32\cmd.exe` {
		t.Errorf("process ComSpec = %q", got)
	}
	// Resolved-env lookup is case-insensitive on Windows.
	if got := domain.InteractiveShell(rt, []string{`comspec=D:\tools\cmd.exe`}); got != `D:\tools\cmd.exe` {
		t.Errorf("resolved ComSpec = %q", got)
	}
}

func TestRunShell_StartsResolvedShell(t *testing.T) {
	rt := newMockRuntime()
	if err := domain.RunShell(rt, types.RunOptions{}, types.TagData{}, nil, nil, nil); err != nil {
		t.Fatalf("RunShell: %v", err)
	}
	if len(rt.exec.startedNames) != 1 || rt.exec.startedNames[0] != "sh" {
		t.Errorf("started = %v, want [sh]", rt.exec.startedNames)
	}
}

func TestRunShell_ResolutionErrorStartsNothing(t *testing.T) {
	rt := newMockRuntime()
	err := domain.RunShell(rt, types.RunOptions{Imports: []string{"missing.toml"}}, types.TagData{}, nil, nil, nil)
	if err == nil {
		t.Fatal("RunShell with a missing import must fail")
	}
	if !strings.Contains(err.Error(), "missing.toml") && len(rt.exec.startedNames) != 0 {
		t.Errorf("err = %v, started = %v", err, rt.exec.startedNames)
	}
	if len(rt.exec.startedNames) != 0 {
		t.Errorf("no process must start on resolution error, started %v", rt.exec.startedNames)
	}
}
