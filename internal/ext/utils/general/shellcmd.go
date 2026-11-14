package general

// RawCmdExeLine returns the verbatim process command line to use when
// running a shell line through cmd.exe on Windows:
//
//	cmd.exe /C "<line>"
//
// The line is wrapped in exactly one pair of double quotes and its
// content is NOT escaped in any way.
//
// Why this is needed: Go's os/exec normally builds the Windows
// command line from the argument slice (makeCmdLine), escaping every
// embedded `"` in an argument as `\"`. That escaping is the C-runtime
// convention, but cmd.exe does not understand backslash escaping at
// all — it treats `\"C:\path\cap.bat\"` as a literal program name with
// backslashes in it and fails with "is not recognized". Any eec code
// path that hands a whole shell line to `cmd /c` (capture, $()
// substitution, the REPL) therefore breaks as soon as the line
// contains a double quote — e.g. a quoted path with no spaces in it.
//
// Setting syscall.SysProcAttr.CmdLine (see UseRawCmdExeLine) bypasses
// makeCmdLine entirely: the string returned here is passed to
// CreateProcess verbatim.
//
// Why exactly one pair of quotes is correct: for /C, cmd preserves
// a leading/trailing quote pair only under a narrow set of
// conditions (exactly two quote characters in total, no special
// characters such as & between them, the quoted text being an
// executable name, ...). A wrapped shell line essentially never
// meets them — a plain line has & between the quotes, a quoted-path
// line has more than two quotes — so cmd falls back to its default
// handling: strip the first and last quote characters and execute
// everything in between. That is exactly the original line, with
// its own quotes surviving untouched because cmd removes only that
// single outer pair and then parses the inner text normally. This
// is the standard invocation form used by tools that spawn cmd
// non-interactively with a pre-built command string.
func RawCmdExeLine(line string) string {
	return `cmd.exe /C "` + line + `"`
}
