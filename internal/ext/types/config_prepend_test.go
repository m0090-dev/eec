package types_test

import (
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// Phase 6-1: [envs.prepend] / [envs.append] merge sections.
// Plain list values keep the historical append behavior; prepend
// sections put their parts BEFORE the existing value (deduped, first
// occurrence wins, so a prepended part already present moves to the
// front). Sections are reserved names only when their value is a map,
// and they nest inside OS sections too.

func buildWithBase(t *testing.T, rt *mockRuntime, path, baseKey, baseVal, sep string) map[string]string {
	t.Helper()
	cfg, err := types.ReadToml(rt, path)
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	base := []string{}
	if baseKey != "" {
		base = []string{baseKey + "=" + baseVal}
	}
	return envSliceToMap(cfg.BuildEnvs(rt, base, sep, general.DefaultExpandOptions()))
}

func TestPrepend_ToExistingBaseValue(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs.prepend]
PATH = ["priority", "tools"]
`)
	out := buildWithBase(t, rt, path, "PATH", "/usr/bin:/bin", ":")
	if out["PATH"] != "priority:tools:/usr/bin:/bin" {
		t.Errorf("PATH = %q, want %q", out["PATH"], "priority:tools:/usr/bin:/bin")
	}
}

func TestPrepend_NoBaseValue(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs.prepend]
PATH = ["only"]
`)
	out := buildWithBase(t, rt, path, "", "", ":")
	if out["PATH"] != "only" {
		t.Errorf("PATH = %q, want %q (prepend with no base acts as a plain list definition)", out["PATH"], "only")
	}
}

func TestPrependAndAppend_OneConfig(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs]
PATH = ["mid"]

[envs.append]
PATH = ["last"]

[envs.prepend]
PATH = ["first"]
`)
	out := buildWithBase(t, rt, path, "PATH", "base", ":")
	want := "first:base:mid:last"
	if out["PATH"] != want {
		t.Errorf("PATH = %q, want %q (prepend ++ base ++ appends in order)", out["PATH"], want)
	}
}

func TestPrepend_DedupMovesToFront(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs.prepend]
PATH = ["shared", "new"]
`)
	out := buildWithBase(t, rt, path, "PATH", "a:shared:b", ":")
	if out["PATH"] != "shared:new:a:b" {
		t.Errorf("PATH = %q, want %q (existing part 'shared' moves to front, no duplicate)", out["PATH"], "shared:new:a:b")
	}
}

func TestPrepend_ScalarIsOneElementList(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs.prepend]
PATH = "solo"
`)
	out := buildWithBase(t, rt, path, "PATH", "/usr/bin", ":")
	if out["PATH"] != "solo:/usr/bin" {
		t.Errorf("PATH = %q, want %q (scalar in [envs.prepend] is a one-element prepend)", out["PATH"], "solo:/usr/bin")
	}
}

func TestAppendSection_EquivalentToPlainList(t *testing.T) {
	rt := newMockRuntime()
	plain := writeTempConfig(t, "plain.toml", `
[envs]
PATH = ["x", "y"]
`)
	section := writeTempConfig(t, "section.toml", `
[envs.append]
PATH = ["x", "y"]
`)
	a := buildWithBase(t, rt, plain, "PATH", "/base", ":")
	b := buildWithBase(t, rt, section, "PATH", "/base", ":")
	if a["PATH"] != "/base:x:y" || b["PATH"] != a["PATH"] {
		t.Errorf("[envs.append] = %q, plain list = %q; want both %q", b["PATH"], a["PATH"], "/base:x:y")
	}

	// The section origin must be recorded for display badges.
	cfg, err := types.ReadToml(rt, section)
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	found := false
	for _, e := range cfg.Envs {
		if e.Key == "PATH" {
			found = true
			if e.Section != "append" || e.Prepend {
				t.Errorf("entry = %+v, want Section=append, Prepend=false", e)
			}
		}
	}
	if !found {
		t.Errorf("PATH entry missing from [envs.append] config: %+v", cfg.Envs)
	}
}

func TestPrepend_OSSectionHonoredOnlyOnMatchingGOOS(t *testing.T) {
	const body = `
[envs]
PATH = ["common"]

[envs.linux.prepend]
PATH = ["linux-first"]

[envs.windows.prepend]
PATH = ["win-first"]
`
	linux := newMockRuntime()
	out := buildWithBase(t, linux, writeTempConfig(t, "c.toml", body), "PATH", "base", ":")
	if out["PATH"] != "linux-first:base:common" {
		t.Errorf("linux PATH = %q, want %q", out["PATH"], "linux-first:base:common")
	}

	win := newMockRuntime()
	win.env.goos = "windows"
	cfg, err := types.ReadToml(win, writeTempConfig(t, "c.toml", body))
	if err != nil {
		t.Fatalf("ReadToml(windows): %v", err)
	}
	out = envSliceToMap(cfg.BuildEnvs(win, []string{"PATH=base"}, ";", general.DefaultExpandOptions()))
	if out["PATH"] != "win-first;base;common" {
		t.Errorf("windows PATH = %q, want %q", out["PATH"], "win-first;base;common")
	}

	// The OS prepend entry must carry its Origin and Prepend flags.
	seenPrepend := false
	for _, e := range cfg.Envs {
		if e.Prepend {
			seenPrepend = true
			if e.Origin != "windows" || e.Section != "prepend" {
				t.Errorf("prepend entry = %+v, want Origin=windows Section=prepend", e)
			}
		}
	}
	if !seenPrepend {
		t.Errorf("no prepend entry recorded for windows config: %+v", cfg.Envs)
	}
}

func TestPrepend_ExpansionUsesCurrentEnv(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs]
ROOT = "/opt/tools"

[envs.prepend]
PATH = ["${ROOT}/bin"]
`)
	out := buildWithBase(t, rt, path, "PATH", "/usr/bin", ":")
	if out["PATH"] != "/opt/tools/bin:/usr/bin" {
		t.Errorf("PATH = %q, want %q (prepend values expand like append values)", out["PATH"], "/opt/tools/bin:/usr/bin")
	}
}

func TestPrepend_VariableNamedPrependStaysVariable(t *testing.T) {
	rt := newMockRuntime()
	path := writeTempConfig(t, "c.toml", `
[envs]
prepend = ["still", "a", "variable"]
append = "scalar-var"
`)
	cfg, err := types.ReadToml(rt, path)
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	vars := envKeySet(cfg)
	if _, ok := vars["prepend"]; !ok {
		t.Errorf("variable 'prepend' (list value) was swallowed as a section: %+v", cfg.Envs)
	}
	if vars["append"] != "scalar-var" {
		t.Errorf("variable 'append' (scalar value) = %q, want scalar-var", vars["append"])
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))
	if out["prepend"] != "still:a:variable" {
		t.Errorf("prepend variable = %q, want %q", out["prepend"], "still:a:variable")
	}
}

func TestPrepend_YamlAndJsonForms(t *testing.T) {
	yamlPath := writeTempConfig(t, "c.yaml", "envs:\n  prepend:\n    PATH: [\"y1\"]\n  append:\n    PATH: [\"y2\"]\n")
	rt := newMockRuntime()
	cfg, err := types.ReadYaml(rt, yamlPath)
	if err != nil {
		t.Fatalf("ReadYaml: %v", err)
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, []string{"PATH=base"}, ":", general.DefaultExpandOptions()))
	if out["PATH"] != "y1:base:y2" {
		t.Errorf("YAML PATH = %q, want %q", out["PATH"], "y1:base:y2")
	}

	jsonPath := writeTempConfig(t, "c.json", `{"envs": {"prepend": {"PATH": ["j1"]}, "append": {"PATH": ["j2"]}}}`)
	cfg, err = types.ReadJson(rt, jsonPath)
	if err != nil {
		t.Fatalf("ReadJson: %v", err)
	}
	out = envSliceToMap(cfg.BuildEnvs(rt, []string{"PATH=base"}, ":", general.DefaultExpandOptions()))
	if out["PATH"] != "j1:base:j2" {
		t.Errorf("JSON PATH = %q, want %q", out["PATH"], "j1:base:j2")
	}
}

// The duplicate-preserving dependency sort must keep genuine cycles
// and self-references as errors while letting duplicated keys through.
func TestSortEnvs_DuplicatesSurvive_CyclesStillFail(t *testing.T) {
	rt := newMockRuntime()
	// Same key three times (plain + OS bucket): all definitions apply.
	path := writeTempConfig(t, "c.toml", `
[envs]
PATH = ["common"]

[envs.linux]
PATH = ["osbucket"]
`)
	out := buildWithBase(t, rt, path, "PATH", "/base", ":")
	if out["PATH"] != "/base:common:osbucket" {
		t.Errorf("PATH = %q, want %q ( OS bucket list now merges after common instead of being dropped)", out["PATH"], "/base:common:osbucket")
	}

	// Genuine cycle across distinct keys still errors with the chain.
	_, err := types.ReadInlineConfig(rt, "[envs]\nA = \"${B}\"\nB = \"${A}\"\n", "toml")
	if err == nil || !strings.Contains(err.Error(), "circular reference detected: A -> B -> A") {
		t.Errorf("cycle err = %v, want circular reference detected: A -> B -> A", err)
	}

	// A lone self-reference is still circular.
	_, err = types.ReadInlineConfig(rt, "[envs]\nPATH = [\"${PATH}/x\"]\n", "toml")
	if err == nil || !strings.Contains(err.Error(), "circular reference detected: PATH -> PATH") {
		t.Errorf("self-ref err = %v, want circular reference detected: PATH -> PATH", err)
	}
}
