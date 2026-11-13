package general

import "testing"

// RawCmdExeLine must wrap the line in exactly one pair of quotes and
// leave the content byte-for-byte untouched: cmd.exe strips only the
// outer pair, and any backslash Go would add is exactly the bug this
// helper exists to avoid. These tests are platform-neutral on
// purpose — the produced string matters on Windows, but it is built
// and verified everywhere.
func TestRawCmdExeLine(t *testing.T) {
	cases := []struct {
		name string
		line string
		want string
	}{
		{
			name: "plain line",
			line: `echo hello & set`,
			want: `cmd.exe /C "echo hello & set"`,
		},
		{
			name: "quoted path with spaces",
			line: `call "C:\My Tools\setup env.bat" & echo done & set`,
			want: `cmd.exe /C "call "C:\My Tools\setup env.bat" & echo done & set"`,
		},
		{
			name: "quoted path without spaces (the CI failure token)",
			line: `"C:\Users\runner\AppData\Local\Temp\eec\cap.bat" & echo __EEC_CAPTURE_SENTINEL__ & set`,
			want: `cmd.exe /C ""C:\Users\runner\AppData\Local\Temp\eec\cap.bat" & echo __EEC_CAPTURE_SENTINEL__ & set"`,
		},
		{
			name: "line already containing quotes is not re-escaped",
			line: `set GREETING="a b" & echo %GREETING%`,
			want: `cmd.exe /C "set GREETING="a b" & echo %GREETING%"`,
		},
		{
			name: "backslashes in content stay single",
			line: `D:\tools\ninja.exe --version`,
			want: `cmd.exe /C "D:\tools\ninja.exe --version"`,
		},
		{
			name: "empty line",
			line: ``,
			want: `cmd.exe /C ""`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RawCmdExeLine(tc.line); got != tc.want {
				t.Errorf("RawCmdExeLine(%q)\n got: %q\nwant: %q", tc.line, got, tc.want)
			}
		})
	}
}
