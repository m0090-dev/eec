//go:build windows
// +build windows

package impl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommand_BareNameResolvedAgainstEnvPath is the Windows side of
// the 2.0.0 regression where `eec run -t dev nvim` failed with
// "executable file not found in %PATH%": the tool exists only on the
// PATH of the environment handed to the child (e.g. a mise shims
// directory added by the config), and Command must resolve the bare
// name against that PATH, including the PATHEXT extension search.
func TestCommand_BareNameResolvedAgainstEnvPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "eec-envpath-tool.EXE")
	if err := os.WriteFile(exe, []byte("fake"), 0o755); err != nil {
		t.Fatalf("failed to write fake exe: %v", err)
	}

	ex := DefaultExecutor{}
	env := []string{"PATH=" + dir, "PATHEXT=.COM;.EXE;.BAT;.CMD"}
	cmd, err := ex.Command("eec-envpath-tool", nil, env, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.EqualFold(cmd.Path, exe) {
		t.Errorf("expected resolved path %q, got %q", exe, cmd.Path)
	}
}

// TestCommand_BareBatchNameRoutedThroughCmd verifies that a bare
// name resolving to a .bat file is still launched through cmd.exe
// /C, with the resolved path as the script argument.
func TestCommand_BareBatchNameRoutedThroughCmd(t *testing.T) {
	dir := t.TempDir()
	bat := filepath.Join(dir, "eec-envpath-bat.BAT")
	if err := os.WriteFile(bat, []byte("@echo off\r\n"), 0o755); err != nil {
		t.Fatalf("failed to write fake bat: %v", err)
	}

	ex := DefaultExecutor{}
	env := []string{"PATH=" + dir, "PATHEXT=.COM;.EXE;.BAT;.CMD"}
	cmd, err := ex.Command("eec-envpath-bat", []string{"arg1"}, env, nil, nil, nil, false)
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(cmd.Path), "cmd.exe") {
		t.Fatalf("expected cmd.exe for batch file, got %q", cmd.Path)
	}
	// cmd.Args is ["cmd.exe", "/C", <resolved bat>, "arg1"].
	if len(cmd.Args) < 3 || cmd.Args[1] != "/C" || !strings.EqualFold(cmd.Args[2], bat) {
		t.Errorf("expected batch arg %q after /C, got args %v", bat, cmd.Args)
	}
}
