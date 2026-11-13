package general_test

import (
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// Phase 3-3: ExpandOptions toggles for ${} expansion and $() execution.

func TestExpandWithOptions_ExecDisabled_LeavesLiteralButExpandsVars(t *testing.T) {
	rt, logger := newCaptureRuntime()
	opts := general.ExpandOptions{ExpandVars: true, ExecCommands: false}
	got := general.ExpandEnvAndCommandsWithOptions(rt,
		"a=${X} b=$(echo hi)", map[string][]string{"X": {"1"}}, opts)
	if got != "a=1 b=$(echo hi)" {
		t.Errorf("got %q, want %q (${} expanded, $() left literal)", got, "a=1 b=$(echo hi)")
	}
	if len(logger.warns) == 0 {
		t.Error("expected a log line noting the skipped command substitution")
	}
	if !strings.Contains(logger.warns[0], "echo hi") {
		t.Errorf("skip log should name the command, got: %q", logger.warns[0])
	}
}

func TestExpandWithOptions_ExpandDisabled_ReturnsInputVerbatim(t *testing.T) {
	rt, logger := newCaptureRuntime()
	opts := general.ExpandOptions{ExpandVars: false, ExecCommands: false}
	in := "a=${X} b=$(echo hi)"
	got := general.ExpandEnvAndCommandsWithOptions(rt, in, map[string][]string{"X": {"1"}}, opts)
	if got != in {
		t.Errorf("got %q, want the input completely unexpanded %q", got, in)
	}
	if len(logger.warns) != 0 {
		t.Errorf("no command should have run or been skipped-logged, got warns %v", logger.warns)
	}
}

func TestExpandWithOptions_Defaults_BehaveLikeLegacy(t *testing.T) {
	rt, _ := newCaptureRuntime()
	got := general.ExpandEnvAndCommandsWithOptions(rt,
		"a=${X} b=$(echo hi)", map[string][]string{"X": {"1"}}, general.DefaultExpandOptions())
	if got != "a=1 b=hi" {
		t.Errorf("got %q, want %q", got, "a=1 b=hi")
	}
}
