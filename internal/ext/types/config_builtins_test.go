package types_test

import (
	"path/filepath"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// Phase 4-2: built-in variables ${CONFIG_DIR} / ${EEC_HOME} are
// expansion-only, lowest priority, and never leak into the output
// unless really defined by the base env or the config itself.

func TestBuildEnvs_BuiltinConfigDir(t *testing.T) {
	rt := newMockRuntime()
	cfgDir := filepath.Join("some", "dir", "sub")
	cfg := types.Config{
		SourcePath: filepath.Join(cfgDir, "dev.toml"),
		Envs: []types.Environ{
			{Key: "TOOL", Value: "${CONFIG_DIR}/bin/tool"},
		},
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))

	want := cfgDir + "/bin/tool"
	if got := out["TOOL"]; got != want {
		t.Errorf("TOOL = %q, want %q (${CONFIG_DIR} must resolve to the config's directory)", got, want)
	}
	if _, ok := out["CONFIG_DIR"]; ok {
		t.Error("CONFIG_DIR leaked into the final env; builtins are expansion-only")
	}
}

func TestBuildEnvs_BuiltinConfigDirSkippedForInline(t *testing.T) {
	rt := newMockRuntime()
	cfg := types.Config{
		SourcePath: "", // inline config: no CONFIG_DIR builtin
		Envs: []types.Environ{
			{Key: "TOOL", Value: "${CONFIG_DIR}/bin/tool"},
		},
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))
	if got := out["TOOL"]; got != "${CONFIG_DIR}/bin/tool" {
		t.Errorf("TOOL = %q, want the reference left literal (no builtin for inline configs)", got)
	}
}

func TestBuildEnvs_ConfigDefinedConfigDirWins(t *testing.T) {
	rt := newMockRuntime()
	cfg := types.Config{
		SourcePath: filepath.Join("cfg", "dev.toml"),
		Envs: []types.Environ{
			{Key: "CONFIG_DIR", Value: "/custom/dir"},
			{Key: "TOOL", Value: "${CONFIG_DIR}/tool"},
		},
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))

	if got := out["TOOL"]; got != "/custom/dir/tool" {
		t.Errorf("TOOL = %q, want %q (config-defined CONFIG_DIR beats the builtin)", got, "/custom/dir/tool")
	}
	if got := out["CONFIG_DIR"]; got != "/custom/dir" {
		t.Errorf("CONFIG_DIR = %q, want %q (really defined, so it IS in the output)", got, "/custom/dir")
	}
}

func TestBuildEnvs_BuiltinEecHome(t *testing.T) {
	rt := newMockRuntime()
	rt.env.lookupMap["EEC_HOME"] = "/data/eec-home"
	cfg := types.Config{
		Envs: []types.Environ{
			{Key: "TAGDIR", Value: "${EEC_HOME}/tags"},
		},
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))

	if got := out["TAGDIR"]; got != "/data/eec-home/tags" {
		t.Errorf("TAGDIR = %q, want %q", got, "/data/eec-home/tags")
	}
	if _, ok := out["EEC_HOME"]; ok {
		t.Error("EEC_HOME leaked into the final env; builtins are expansion-only")
	}
}

func TestBuildEnvs_BaseEnvEecHomeWinsOverBuiltin(t *testing.T) {
	rt := newMockRuntime()
	rt.env.lookupMap["EEC_HOME"] = "/data/from-resolver"
	cfg := types.Config{
		Envs: []types.Environ{
			{Key: "TAGDIR", Value: "${EEC_HOME}/tags"},
		},
	}
	base := []string{"EEC_HOME=/data/from-base"}
	out := envSliceToMap(cfg.BuildEnvs(rt, base, ":", general.DefaultExpandOptions()))

	if got := out["TAGDIR"]; got != "/data/from-base/tags" {
		t.Errorf("TAGDIR = %q, want %q (a real base-env value beats the builtin)", got, "/data/from-base/tags")
	}
	if got := out["EEC_HOME"]; got != "/data/from-base" {
		t.Errorf("EEC_HOME = %q, want %q (really defined in base env, so it stays)", got, "/data/from-base")
	}
}
