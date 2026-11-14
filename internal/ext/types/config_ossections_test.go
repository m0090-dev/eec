package types_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// Phase 3-2: OS-specific env sections ([envs.windows] / [envs.linux] /
// [envs.darwin], aliases win / mac / macos). Only the bucket matching
// the runtime GOOS is merged, after the common entries.

func writeTempConfig(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func envKeySet(cfg types.Config) map[string]string {
	m := make(map[string]string, len(cfg.Envs))
	for _, e := range cfg.Envs {
		if s, ok := e.Value.(string); ok {
			m[e.Key] = s
		} else {
			m[e.Key] = ""
		}
	}
	return m
}

const osSectionsToml = `
[envs]
COMMON = "common"
SHARED = "${COMMON}-shared"

[envs.windows]
WIN_ONLY = "win"
FINAL = "win:${SHARED}"

[envs.linux]
LINUX_ONLY = "linux"
FINAL = "linux:${SHARED}"
`

func TestOSSections_Toml_LinuxRuntime(t *testing.T) {
	rt := newMockRuntime() // linux
	cfg, err := types.ReadToml(rt, writeTempConfig(t, "c.toml", osSectionsToml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	vars := envKeySet(cfg)
	if _, ok := vars["LINUX_ONLY"]; !ok {
		t.Errorf("linux bucket missing, got %v", vars)
	}
	if _, ok := vars["WIN_ONLY"]; ok {
		t.Errorf("windows bucket leaked into linux config: %v", vars)
	}
	if vars["COMMON"] != "common" {
		t.Errorf("COMMON = %q, want common", vars["COMMON"])
	}
	// OS-specific FINAL must win over nothing/common and reference the
	// common SHARED var: after build it resolves through the common one.
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))
	if out["FINAL"] != "linux:common-shared" {
		t.Errorf("FINAL = %q, want %q (OS bucket referencing common var)", out["FINAL"], "linux:common-shared")
	}
}

func TestOSSections_Toml_WindowsRuntime(t *testing.T) {
	rt := newMockRuntime()
	rt.env.goos = "windows"
	cfg, err := types.ReadToml(rt, writeTempConfig(t, "c.toml", osSectionsToml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	vars := envKeySet(cfg)
	if _, ok := vars["WIN_ONLY"]; !ok {
		t.Errorf("windows bucket missing, got %v", vars)
	}
	if _, ok := vars["LINUX_ONLY"]; ok {
		t.Errorf("linux bucket leaked into windows config: %v", vars)
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ";", general.DefaultExpandOptions()))
	if out["FINAL"] != "win:common-shared" {
		t.Errorf("FINAL = %q, want %q", out["FINAL"], "win:common-shared")
	}
}

func TestOSSections_MacAliasForDarwin(t *testing.T) {
	const toml = `
[envs]
A = "1"

[envs.mac]
MAC_ONLY = "yes"
`
	rt := newMockRuntime()
	rt.env.goos = "darwin"
	cfg, err := types.ReadToml(rt, writeTempConfig(t, "c.toml", toml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	if _, ok := envKeySet(cfg)["MAC_ONLY"]; !ok {
		t.Errorf("darwin runtime should pick up the [envs.mac] alias bucket, got %v", envKeySet(cfg))
	}

	rtLinux := newMockRuntime()
	cfgLinux, err := types.ReadToml(rtLinux, writeTempConfig(t, "c.toml", toml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	if _, ok := envKeySet(cfgLinux)["MAC_ONLY"]; ok {
		t.Error("linux runtime must not pick up the mac bucket")
	}
}

func TestOSSections_CrossReferencesBothDirections(t *testing.T) {
	// Common var references an OS-specific var, and the OS-specific var
	// references a common one. Both must resolve because buckets are
	// merged before dependency sorting.
	const toml = `
[envs]
ROOT = "/opt"
FROM_OS = "${OSDIR}/bin"

[envs.linux]
OSDIR = "${ROOT}/linux"
`
	rt := newMockRuntime()
	cfg, err := types.ReadToml(rt, writeTempConfig(t, "c.toml", toml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	out := envSliceToMap(cfg.BuildEnvs(rt, nil, ":", general.DefaultExpandOptions()))
	if out["FROM_OS"] != "/opt/linux/bin" {
		t.Errorf("FROM_OS = %q, want %q", out["FROM_OS"], "/opt/linux/bin")
	}
	if out["OSDIR"] != "/opt/linux" {
		t.Errorf("OSDIR = %q, want %q", out["OSDIR"], "/opt/linux")
	}
}

func TestOSSections_ReservedNameWithScalarStaysVariable(t *testing.T) {
	const toml = `
[envs]
windows = "just-a-string"
linux = "also-a-string"
`
	rt := newMockRuntime()
	cfg, err := types.ReadToml(rt, writeTempConfig(t, "c.toml", toml))
	if err != nil {
		t.Fatalf("ReadToml: %v", err)
	}
	vars := envKeySet(cfg)
	if vars["windows"] != "just-a-string" {
		t.Errorf("windows = %q, want the scalar kept as a regular variable", vars["windows"])
	}
	if vars["linux"] != "also-a-string" {
		t.Errorf("linux = %q, want the scalar kept as a regular variable", vars["linux"])
	}
}

func TestOSSections_JsonAndYamlForms(t *testing.T) {
	jsonPath := writeTempConfig(t, "c.json",
		`{"envs": {"COMMON": "c", "windows": {"W": "1"}, "linux": {"L": "2"}}}`)
	rt := newMockRuntime()
	cfg, err := types.ReadJson(rt, jsonPath)
	if err != nil {
		t.Fatalf("ReadJson: %v", err)
	}
	vars := envKeySet(cfg)
	if vars["L"] != "2" {
		t.Errorf("JSON linux bucket missing: %v", vars)
	}
	if _, ok := vars["W"]; ok {
		t.Errorf("JSON windows bucket leaked into linux: %v", vars)
	}

	yamlPath := writeTempConfig(t, "c.yaml", "envs:\n  COMMON: c\n  linux:\n    L: \"2\"\n  darwin:\n    D: \"3\"\n")
	cfg, err = types.ReadYaml(rt, yamlPath)
	if err != nil {
		t.Fatalf("ReadYaml: %v", err)
	}
	vars = envKeySet(cfg)
	if vars["L"] != "2" {
		t.Errorf("YAML linux bucket missing: %v", vars)
	}
	if _, ok := vars["D"]; ok {
		t.Errorf("YAML darwin bucket leaked into linux: %v", vars)
	}
}
