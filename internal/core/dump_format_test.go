package core_test

import (
	"testing"

	core "github.com/m0090-dev/eec/internal/core"
)

func TestFormatDumpLine(t *testing.T) {
	cases := []struct {
		name  string
		kv    string
		shell string
		want  string
		ok    bool
	}{
		{"unix simple", "FOO=bar", "unix", "export FOO='bar'", true},
		{"unix space", "FOO=a b", "unix", "export FOO='a b'", true},
		{"unix semicolon", "FOO=10;10", "unix", "export FOO='10;10'", true},
		{"unix dollar", "FOO=$HOME/x", "unix", "export FOO='$HOME/x'", true},
		{"unix single quote", "FOO=it's", "unix", `export FOO='it'\''s'`, true},
		{"unix empty value", "FOO=", "unix", "export FOO=''", true},
		{"unix value with equals", "FOO=a=b", "unix", "export FOO='a=b'", true},
		{"unix no equals", "FOO", "unix", "export FOO", true},
		{"win simple", "FOO=bar", "win", `set "FOO=bar"`, true},
		{"win space and semicolon", "PATH=C:\\a b;C:\\x", "win", `set "PATH=C:\a b;C:\x"`, true},
		{"win no equals skipped", "FOO", "win", "", false},
		{"pwsh simple", "FOO=bar", "pwsh", "$env:FOO = 'bar'", true},
		{"pwsh alias powershell", "FOO=bar", "powershell", "$env:FOO = 'bar'", true},
		{"pwsh space and semicolon", "PATH=C:\\a b;C:\\x", "pwsh", `$env:PATH = 'C:\a b;C:\x'`, true},
		{"pwsh single quote doubled", "FOO=it's", "pwsh", "$env:FOO = 'it''s'", true},
		{"pwsh empty value", "FOO=", "pwsh", "$env:FOO = ''", true},
		{"pwsh value with equals", "FOO=a=b", "pwsh", "$env:FOO = 'a=b'", true},
		{"pwsh newline kept literal", "FOO=line1\nline2", "pwsh", "$env:FOO = 'line1\nline2'", true},
		{"pwsh no equals skipped", "FOO", "pwsh", "", false},
		{"default raw", "FOO=a b;c", "", "FOO=a b;c", true},
		{"unknown shell raw", "FOO=bar", "fish", "FOO=bar", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := core.FormatDumpLine(tc.kv, tc.shell)
			if got != tc.want || ok != tc.ok {
				t.Errorf("FormatDumpLine(%q, %q) = (%q, %v), want (%q, %v)",
					tc.kv, tc.shell, got, ok, tc.want, tc.ok)
			}
		})
	}
}
