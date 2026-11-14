package domain_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Phase 3-3: [configs] expand_vars / exec_commands and the CLI
// NoExpand / NoExec overrides must take effect through ResolveRunOptions.

func resolveToEnvMap(t *testing.T, rt *mockRuntime, opts types.RunOptions) map[string]string {
	t.Helper()
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	m := make(map[string]string, len(finalEnv))
	for _, e := range finalEnv {
		if parts := strings.SplitN(e, "=", 2); len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func finalEnvMap(t *testing.T, opts types.RunOptions) map[string]string {
	t.Helper()
	return resolveToEnvMap(t, newMockRuntime(), opts)
}

func TestResolveRunOptions_ConfigExecCommandsFalse(t *testing.T) {
	opts := types.RunOptions{InlineConfigToml: `
[configs]
exec_commands = false

[envs]
A = "hello"
B = "${A}-$(echo nope)"
`}
	env := finalEnvMap(t, opts)
	if env["B"] != "hello-$(echo nope)" {
		t.Errorf("B = %q, want %q (config exec_commands=false: ${} expands, $() stays literal)",
			env["B"], "hello-$(echo nope)")
	}
}

func TestResolveRunOptions_ConfigExpandVarsFalse(t *testing.T) {
	opts := types.RunOptions{InlineConfigToml: `
[configs]
expand_vars = false

[envs]
A = "hello"
B = "${A}-$(echo nope)"
`}
	env := finalEnvMap(t, opts)
	if env["B"] != "${A}-$(echo nope)" {
		t.Errorf("B = %q, want the completely unexpanded %q", env["B"], "${A}-$(echo nope)")
	}
}

func TestResolveRunOptions_CLINoExecOverridesConfig(t *testing.T) {
	// No [configs] section at all: the CLI flag alone must disable exec.
	opts := types.RunOptions{
		NoExec: true,
		InlineConfigToml: `
[envs]
A = "hello"
B = "${A}-$(echo nope)"
`,
	}
	env := finalEnvMap(t, opts)
	if env["B"] != "hello-$(echo nope)" {
		t.Errorf("B = %q, want %q (NoExec)", env["B"], "hello-$(echo nope)")
	}
}

func TestResolveRunOptions_CLINoExpandOverridesConfigTrue(t *testing.T) {
	// Config explicitly enables expansion; the CLI flag still wins.
	opts := types.RunOptions{
		NoExpand: true,
		InlineConfigToml: `
[configs]
expand_vars = true
exec_commands = true

[envs]
A = "hello"
B = "${A}-x"
`,
	}
	env := finalEnvMap(t, opts)
	if env["B"] != "${A}-x" {
		t.Errorf("B = %q, want unexpanded %q (NoExpand beats config)", env["B"], "${A}-x")
	}
}

// Later configs override earlier ones for the toggles, mirroring the
// later-wins semantics of env vars: the main config file disables
// exec, an inline config appended afterwards re-enables it (and the
// reverse direction too).
func TestResolveRunOptions_LaterConfigOverridesEarlierToggle(t *testing.T) {
	// ResolveRunOptions rewrites ConfigFile with filepath.Abs before
	// it consults the FS, so the mock FS must register the file under
	// that same normalized key. A Unix-style "/mock/main.toml" is NOT
	// absolute on Windows (it has no volume name): Abs supplies the
	// current drive there ("C:\mock\main.toml"), the un-normalized key
	// would miss, and the main config would silently be skipped —
	// leaving B undefined and both sub-cases testing nothing.
	path, err := filepath.Abs("/mock/main.toml")
	if err != nil {
		t.Fatal(err)
	}
	fileWith := func(execCmd string) *mockRuntime {
		rt := newMockRuntime()
		rt.fs.existingFiles[path] = true
		rt.fs.fileData[path] = []byte("[configs]\nexec_commands = " + execCmd + "\n\n[envs]\nB = \"$(echo nope)\"\n")
		return rt
	}

	// file: false, inline (later): true -> exec happens, so $() does
	// not stay literal (the mock executor substitutes something else).
	env := resolveToEnvMap(t, fileWith("false"), types.RunOptions{
		ConfigFile:       path,
		InlineConfigToml: "[configs]\nexec_commands = true\n",
	})
	if _, ok := env["B"]; !ok {
		t.Fatal("B is not defined; the main config file was not loaded")
	}
	if env["B"] == "$(echo nope)" {
		t.Error("B stayed literal; the later exec_commands=true should have re-enabled execution")
	}

	// file: true, inline (later): false -> $() stays literal.
	env = resolveToEnvMap(t, fileWith("true"), types.RunOptions{
		ConfigFile:       path,
		InlineConfigToml: "[configs]\nexec_commands = false\n",
	})
	if _, ok := env["B"]; !ok {
		t.Fatal("B is not defined; the main config file was not loaded")
	}
	if env["B"] != "$(echo nope)" {
		t.Errorf("B = %q, want literal %q (later exec_commands=false wins)", env["B"], "$(echo nope)")
	}
}
