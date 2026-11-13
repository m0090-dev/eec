package domain_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Phase 4-4: generic env capture. A capture command runs in a shell
// on top of the resolved env; whatever it sets / changes / unsets is
// diffed back into the final env. These tests use the real runtime,
// so the fixture is written in the host shell's own language:
//
//   - Unix: the script must be sourced (". script") so its exports
//     happen in the capture shell itself.
//   - Windows: a .bat file invoked with CALL (batch files run
//     in-process of cmd); "set X=" is how cmd removes a variable.

type captureFixture struct {
	scriptName string
	script     string
	commands   []string
}

func captureFixtureFor(goos string) captureFixture {
	if goos == "windows" {
		return captureFixture{
			scriptName: "cap.bat",
			script: "@echo off\r\n" +
				"set EEC_CAP_NEW=brand-new\r\n" +
				"set EEC_CAP_EXISTING=changed\r\n" +
				"set EEC_CAP_REMOVE_ME=\r\n" +
				"echo garbage printed by the command itself\r\n" +
				"echo FAKE_VAR=pollution-attempt\r\n",
			commands: []string{
				`call "${CAPDIR}\cap.bat"`,
				"set EEC_CAP_SECOND=${EEC_CAP_NEW}-plus",
			},
		}
	}
	return captureFixture{
		scriptName: "cap.sh",
		script: `export EEC_CAP_NEW="brand-new"
export EEC_CAP_EXISTING="changed"
unset EEC_CAP_REMOVE_ME
echo "garbage printed by the command itself"
echo "FAKE_VAR=pollution-attempt"
`,
		commands: []string{
			`. "${CAPDIR}/cap.sh"`,
			"export EEC_CAP_SECOND=${EEC_CAP_NEW}-plus",
		},
	}
}

func writeCaptureFixture(t *testing.T, extraConfigs string) string {
	t.Helper()
	dir := t.TempDir()
	fx := captureFixtureFor(runtime.GOOS)
	if err := os.WriteFile(filepath.Join(dir, fx.scriptName), []byte(fx.script), 0644); err != nil {
		t.Fatal(err)
	}
	quoted := make([]string, len(fx.commands))
	for i, c := range fx.commands {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	toml := fmt.Sprintf(`%s
[envs]
CAPDIR = %q
EEC_CAP_EXISTING = "original"
EEC_CAP_REMOVE_ME = "to-be-removed"
KEEP_ME = "untouched"

[capture]
commands = [%s]
`, extraConfigs, dir, strings.Join(quoted, ", "))
	cfgPath := filepath.Join(dir, "with-capture.toml")
	if err := os.WriteFile(cfgPath, []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func TestResolveRunOptions_CaptureMergesEnvDiff(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	cfgPath := writeCaptureFixture(t, "")

	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{ConfigFile: cfgPath}, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)

	if got := env["EEC_CAP_NEW"]; got != "brand-new" {
		t.Errorf("EEC_CAP_NEW = %q, want %q (added by capture)", got, "brand-new")
	}
	if got := env["EEC_CAP_EXISTING"]; got != "changed" {
		t.Errorf("EEC_CAP_EXISTING = %q, want %q (changed by capture)", got, "changed")
	}
	if _, ok := env["EEC_CAP_REMOVE_ME"]; ok {
		t.Error("EEC_CAP_REMOVE_ME still present; capture unset must remove it")
	}
	if got := env["KEEP_ME"]; got != "untouched" {
		t.Errorf("KEEP_ME = %q, want %q (untouched keys stay)", got, "untouched")
	}
	if _, ok := env["FAKE_VAR"]; ok {
		t.Error("FAKE_VAR leaked from the command's stdout; only post-sentinel output may be parsed")
	}
	// The second command's ${} was expanded against the env produced
	// by the first capture.
	if got := env["EEC_CAP_SECOND"]; got != "brand-new-plus" {
		t.Errorf("EEC_CAP_SECOND = %q, want %q", got, "brand-new-plus")
	}
}

// The non-host fixture cannot be executed here, but it must still be
// well-formed: the script carries the batch set/unset lines and the
// generated commands survive a TOML round-trip unchanged.
func TestCaptureFixture_WindowsVariantWellFormed(t *testing.T) {
	fx := captureFixtureFor("windows")
	if fx.scriptName != "cap.bat" {
		t.Errorf("scriptName = %q, want cap.bat", fx.scriptName)
	}
	for _, want := range []string{"set EEC_CAP_NEW=brand-new", "set EEC_CAP_EXISTING=changed", "set EEC_CAP_REMOVE_ME="} {
		if !strings.Contains(fx.script, want) {
			t.Errorf("batch script is missing %q:\n%s", want, fx.script)
		}
	}
	quoted := make([]string, len(fx.commands))
	for i, c := range fx.commands {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	rt := &impl.DefaultRuntime{}
	cfg, err := types.ReadInlineConfig(rt,
		"[capture]\ncommands = ["+strings.Join(quoted, ", ")+"]\n", "toml")
	if err != nil {
		t.Fatalf("windows capture commands do not parse as TOML: %v", err)
	}
	if len(cfg.CaptureCommands) != len(fx.commands) {
		t.Fatalf("CaptureCommands = %v, want %v", cfg.CaptureCommands, fx.commands)
	}
	for i := range fx.commands {
		if cfg.CaptureCommands[i] != fx.commands[i] {
			t.Errorf("command %d = %q, want %q (TOML round-trip)", i, cfg.CaptureCommands[i], fx.commands[i])
		}
	}
}

func TestResolveRunOptions_CaptureSkippedWhenExecDisabled(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	cfgPath := writeCaptureFixture(t, "[configs]\nexec_commands = false\n")

	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{ConfigFile: cfgPath}, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)
	if _, ok := env["EEC_CAP_NEW"]; ok {
		t.Error("EEC_CAP_NEW present although exec_commands=false must skip captures")
	}
	if got := env["EEC_CAP_EXISTING"]; got != "original" {
		t.Errorf("EEC_CAP_EXISTING = %q, want %q (capture skipped)", got, "original")
	}
}

func TestResolveRunOptions_CaptureTimeout(t *testing.T) {
	rt := &impl.DefaultRuntime{}

	old := domain.CaptureTimeout
	domain.CaptureTimeout = 200 * time.Millisecond
	defer func() { domain.CaptureTimeout = old }()

	// A command that runs far longer than the shortened timeout, in
	// the host shell's own vocabulary: cmd has no "sleep" on a stock
	// Windows install (it only appears via Git for Windows), so wait
	// with ping's per-reply delay there.
	slowCmd := "sleep 5"
	if runtime.GOOS == "windows" {
		slowCmd = "ping -n 6 127.0.0.1 >nul"
	}
	opts := types.RunOptions{InlineConfigToml: fmt.Sprintf("[capture]\ncommands = [%q]\n", slowCmd)}
	start := time.Now()
	_, _, _, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), slowCmd) || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to name the command and the timeout", err)
	}
	if elapsed > 4*time.Second {
		t.Errorf("timeout took %s; the capture should be killed near %s", elapsed, domain.CaptureTimeout)
	}
}

func TestResolveRunOptions_CaptureFailureNamesCommand(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	// A command after which the shell can no longer print the
	// sentinel dump: on Unix `exec` replaces the shell process, so
	// nothing after it runs; on Windows `exit` terminates cmd itself
	// before the rest of the composite ("& echo <sentinel> & set")
	// can run. (cmd has no `exec`; the POSIX spelling would just be
	// an unknown token whose error the composite survives.)
	bailCmd := "exec true"
	if runtime.GOOS == "windows" {
		bailCmd = "exit"
	}
	opts := types.RunOptions{InlineConfigToml: fmt.Sprintf("[capture]\ncommands = [%q]\n", bailCmd)}
	_, _, _, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err == nil {
		t.Fatal("expected an error for unparseable capture output, got nil")
	}
	if !strings.Contains(err.Error(), bailCmd) {
		t.Errorf("error = %q, want it to name the failing command", err)
	}
}
