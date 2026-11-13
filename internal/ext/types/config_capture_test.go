package types_test

import (
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// Phase 4-4: [capture] parsing — table form, plain list form, and
// single-string form across TOML / YAML / JSON.

func TestReadInlineConfig_CaptureTableForm(t *testing.T) {
	rt := newMockRuntime()
	cfg, err := types.ReadInlineConfig(rt, `
[capture]
commands = ["setup-env.sh", "scripts/more.sh"]
`, "toml")
	if err != nil {
		t.Fatalf("ReadInlineConfig: %v", err)
	}
	want := []string{"setup-env.sh", "scripts/more.sh"}
	if len(cfg.CaptureCommands) != len(want) {
		t.Fatalf("CaptureCommands = %v, want %v", cfg.CaptureCommands, want)
	}
	for i := range want {
		if cfg.CaptureCommands[i] != want[i] {
			t.Errorf("CaptureCommands[%d] = %q, want %q", i, cfg.CaptureCommands[i], want[i])
		}
	}
}

func TestReadInlineConfig_CapturePlainListForm(t *testing.T) {
	rt := newMockRuntime()
	cfg, err := types.ReadInlineConfig(rt, `capture = ["only.sh"]`, "toml")
	if err != nil {
		t.Fatalf("ReadInlineConfig: %v", err)
	}
	if len(cfg.CaptureCommands) != 1 || cfg.CaptureCommands[0] != "only.sh" {
		t.Errorf("CaptureCommands = %v, want [only.sh]", cfg.CaptureCommands)
	}
}

func TestReadInlineConfig_CaptureJsonListForm(t *testing.T) {
	rt := newMockRuntime()
	cfg, err := types.ReadInlineConfig(rt, `{"capture": ["a.sh", "b.sh"]}`, "json")
	if err != nil {
		t.Fatalf("ReadInlineConfig: %v", err)
	}
	if len(cfg.CaptureCommands) != 2 {
		t.Errorf("CaptureCommands = %v, want [a.sh b.sh]", cfg.CaptureCommands)
	}
}

func TestReadInlineConfig_CaptureYamlTableForm(t *testing.T) {
	rt := newMockRuntime()
	cfg, err := types.ReadInlineConfig(rt, "capture:\n  commands:\n    - setup.sh\n", "yaml")
	if err != nil {
		t.Fatalf("ReadInlineConfig: %v", err)
	}
	if len(cfg.CaptureCommands) != 1 || cfg.CaptureCommands[0] != "setup.sh" {
		t.Errorf("CaptureCommands = %v, want [setup.sh]", cfg.CaptureCommands)
	}
}

func TestReadInlineConfig_NoCapture(t *testing.T) {
	rt := newMockRuntime()
	cfg, err := types.ReadInlineConfig(rt, "[envs]\nA = \"1\"\n", "toml")
	if err != nil {
		t.Fatalf("ReadInlineConfig: %v", err)
	}
	if len(cfg.CaptureCommands) != 0 {
		t.Errorf("CaptureCommands = %v, want empty", cfg.CaptureCommands)
	}
}
