package types_test

import (
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// Phase 3-3: BuildEnvs must honor ExpandOptions.

func toggleCfg() types.Config {
	return types.Config{Envs: []types.Environ{
		{Key: "A", Value: "hello"},
		{Key: "B", Value: "${A} $(echo hi)"},
	}}
}

func TestBuildEnvs_ExecDisabled_KeepsCommandLiteral(t *testing.T) {
	rt := newMockRuntime()
	cfg := toggleCfg()
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":",
		general.ExpandOptions{ExpandVars: true, ExecCommands: false}))
	if out["B"] != "hello $(echo hi)" {
		t.Errorf("B = %q, want %q (vars expanded, command left literal)", out["B"], "hello $(echo hi)")
	}
}

func TestBuildEnvs_ExpandDisabled_KeepsEverythingLiteral(t *testing.T) {
	rt := newMockRuntime()
	cfg := toggleCfg()
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":",
		general.ExpandOptions{ExpandVars: false, ExecCommands: false}))
	if out["B"] != "${A} $(echo hi)" {
		t.Errorf("B = %q, want the raw value %q", out["B"], "${A} $(echo hi)")
	}
}
