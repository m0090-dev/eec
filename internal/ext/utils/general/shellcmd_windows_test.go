//go:build windows
// +build windows

package general

import (
	"os/exec"
	"syscall"
	"testing"
)

// UseRawCmdExeLine must set SysProcAttr.CmdLine AND preserve the
// fields the executor already put there (HideWindow / CreationFlags).
func TestUseRawCmdExeLine_PreservesSysProcAttr(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "ignored")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000010}

	UseRawCmdExeLine(cmd, `"C:\dir with spaces\cap.bat" & set`)

	if want := `cmd.exe /C ""C:\dir with spaces\cap.bat" & set"`; cmd.SysProcAttr.CmdLine != want {
		t.Errorf("CmdLine = %q, want %q", cmd.SysProcAttr.CmdLine, want)
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow was lost")
	}
	if cmd.SysProcAttr.CreationFlags != 0x00000010 {
		t.Errorf("CreationFlags = %#x, want 0x10", cmd.SysProcAttr.CreationFlags)
	}
}

// UseRawCmdExeLine must also work when nothing set SysProcAttr yet.
func TestUseRawCmdExeLine_NilSysProcAttr(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "ignored")
	if cmd.SysProcAttr != nil {
		t.Fatal("precondition: SysProcAttr should start nil")
	}

	UseRawCmdExeLine(cmd, "echo hi")

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr was not created")
	}
	if want := `cmd.exe /C "echo hi"`; cmd.SysProcAttr.CmdLine != want {
		t.Errorf("CmdLine = %q, want %q", cmd.SysProcAttr.CmdLine, want)
	}
}
