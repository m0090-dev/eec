//go:build windows
// +build windows

package general

import (
	"os/exec"
	"syscall"
)

// UseRawCmdExeLine rewrites cmd so that cmd.exe receives the shell
// line verbatim instead of Go's backslash-escaped argument encoding
// (see RawCmdExeLine). cmd must have been created for "cmd"/"cmd.exe"
// with args ["/c", line]; the same line is passed again here.
//
// The executor that produced cmd already populated SysProcAttr
// (HideWindow / CreationFlags), so the existing struct is mutated in
// place; it is created only when nil.
func UseRawCmdExeLine(cmd *exec.Cmd, line string) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CmdLine = RawCmdExeLine(line)
}
