package domain_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Phase 4-4: generic env capture. A capture command runs in a shell
// on top of the resolved env; whatever it exports / changes / unsets
// is diffed back into the final env. Uses the real runtime on Linux:
// the script must be sourced (". script") so its exports happen in
// the capture shell itself.

const captureScript = `export EEC_CAP_NEW="brand-new"
export EEC_CAP_EXISTING="changed"
unset EEC_CAP_REMOVE_ME
echo "garbage printed by the command itself"
echo "FAKE_VAR=pollution-attempt"
`

func writeCaptureFixture(t *testing.T, extraConfigs string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cap.sh"), []byte(captureScript), 0644); err != nil {
		t.Fatal(err)
	}
	toml := fmt.Sprintf(`%s
[envs]
CAPDIR = %q
EEC_CAP_EXISTING = "original"
EEC_CAP_REMOVE_ME = "to-be-removed"
KEEP_ME = "untouched"

[capture]
commands = [". \"${CAPDIR}/cap.sh\"", "export EEC_CAP_SECOND=${EEC_CAP_NEW}-plus"]
`, extraConfigs, dir)
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

	opts := types.RunOptions{InlineConfigToml: "[capture]\ncommands = [\"sleep 5\"]\n"}
	start := time.Now()
	_, _, _, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "sleep 5") || !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want it to name the command and the timeout", err)
	}
	if elapsed > 4*time.Second {
		t.Errorf("timeout took %s; the capture should be killed near %s", elapsed, domain.CaptureTimeout)
	}
}

func TestResolveRunOptions_CaptureFailureNamesCommand(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	// A command whose shell cannot even produce the sentinel dump:
	// `exec` replaces the shell, so no sentinel is ever printed.
	opts := types.RunOptions{InlineConfigToml: "[capture]\ncommands = [\"exec true\"]\n"}
	_, _, _, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err == nil {
		t.Fatal("expected an error for unparseable capture output, got nil")
	}
	if !strings.Contains(err.Error(), "exec true") {
		t.Errorf("error = %q, want it to name the failing command", err)
	}
}
