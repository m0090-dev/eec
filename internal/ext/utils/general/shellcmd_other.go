//go:build !windows
// +build !windows

package general

import "os/exec"

// UseRawCmdExeLine is a no-op on non-Windows platforms: POSIX shells
// receive the line as a single argv element (`sh -c <line>`), so no
// command-line re-encoding is needed. It exists so Windows call sites
// do not need their own build-tagged code. See RawCmdExeLine.
func UseRawCmdExeLine(cmd *exec.Cmd, line string) {}
