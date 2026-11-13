package types_test

import (
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// Phase 3-1: OS-aware env key case handling.
// Windows: keys are case-insensitive (canonical form is upper case).
// Linux/macOS: keys are case-sensitive and must be preserved verbatim.

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

func TestCanonicalEnvKey(t *testing.T) {
	cases := []struct{ goos, key, want string }{
		{"windows", "Path", "PATH"},
		{"windows", "http_proxy", "HTTP_PROXY"},
		{"linux", "http_proxy", "http_proxy"},
		{"linux", "Path", "Path"},
		{"darwin", "Home", "Home"},
	}
	for _, c := range cases {
		if got := types.CanonicalEnvKey(c.goos, c.key); got != c.want {
			t.Errorf("CanonicalEnvKey(%q, %q) = %q, want %q", c.goos, c.key, got, c.want)
		}
	}
}

func TestEnvKeyEqual(t *testing.T) {
	if !types.EnvKeyEqual("windows", "Path", "PATH") {
		t.Error("windows: Path should equal PATH")
	}
	if types.EnvKeyEqual("linux", "http_proxy", "HTTP_PROXY") {
		t.Error("linux: http_proxy should NOT equal HTTP_PROXY")
	}
	if !types.EnvKeyEqual("linux", "http_proxy", "http_proxy") {
		t.Error("linux: identical keys should be equal")
	}
}

func TestBuildEnvs_Linux_PreservesCaseAndKeepsVarsSeparate(t *testing.T) {
	rt := newMockRuntime() // goos = linux
	cfg := types.Config{Envs: []types.Environ{
		{Key: "http_proxy", Value: "http://lower"},
		{Key: "HTTP_PROXY", Value: "http://upper"},
		{Key: "USES", Value: "${http_proxy}"},
	}}
	out := envSliceToMap(cfg.BuildEnvs(rt, []string{"MiXeD=base"}, ":"))

	if got := out["http_proxy"]; got != "http://lower" {
		t.Errorf("http_proxy = %q, want %q (case must be preserved)", got, "http://lower")
	}
	if got := out["HTTP_PROXY"]; got != "http://upper" {
		t.Errorf("HTTP_PROXY = %q, want %q (must stay a separate variable)", got, "http://upper")
	}
	if got := out["USES"]; got != "http://lower" {
		t.Errorf("USES = %q, want %q (${http_proxy} must resolve to the lower-case var)", got, "http://lower")
	}
	if got := out["MiXeD"]; got != "base" {
		t.Errorf("MiXeD = %q, want %q (base env case must be preserved)", got, "base")
	}
	if _, ok := out["MIXED"]; ok {
		t.Error("base env key was upper-cased on linux: found MIXED")
	}
}

func TestBuildEnvs_Windows_PathAndPATHCollide(t *testing.T) {
	rt := newMockRuntime()
	rt.env.goos = "windows"
	cfg := types.Config{Envs: []types.Environ{
		{Key: "PATH", Value: []interface{}{`C:\tools`}},
		{Key: "Ref", Value: "${Path}"},
	}}
	out := envSliceToMap(cfg.BuildEnvs(rt, []string{`Path=C:\base`}, ";"))

	merged, ok := out["PATH"]
	if !ok {
		t.Fatalf("expected canonical PATH key, got %v", out)
	}
	if _, ok := out["Path"]; ok {
		t.Errorf("windows: base key 'Path' should have been canonicalized into PATH, got %v", out)
	}
	if !strings.Contains(merged, `C:\base`) || !strings.Contains(merged, `C:\tools`) {
		t.Errorf("PATH = %q, want both C:\\base and C:\\tools merged", merged)
	}
	if got := out["REF"]; got != merged {
		t.Errorf("REF = %q, want %q (${Path} must resolve via the canonical key)", got, merged)
	}
}

func TestSortEnvsByDependency_CaseRules(t *testing.T) {
	// Exact-case dependency resolves on every OS.
	for _, ci := range []bool{false, true} {
		cfg := types.Config{Envs: []types.Environ{
			{Key: "B", Value: "${a}"},
			{Key: "a", Value: "1"},
		}}
		if err := cfg.SortEnvsByDependency(ci); err != nil {
			t.Fatalf("caseInsensitive=%v: %v", ci, err)
		}
		if cfg.Envs[0].Key != "a" {
			t.Errorf("caseInsensitive=%v: expected dependency 'a' first, got %v", ci, cfg.Envs)
		}
	}

	// Case-mismatched dependency: NOT a dependency on case-sensitive OSes,
	// but a dependency on Windows.
	mismatch := func() types.Config {
		return types.Config{Envs: []types.Environ{
			{Key: "B", Value: "${A}"},
			{Key: "a", Value: "1"},
		}}
	}
	cfgLinux := mismatch()
	if err := cfgLinux.SortEnvsByDependency(false); err != nil {
		t.Fatal(err)
	}
	if cfgLinux.Envs[0].Key != "B" {
		t.Errorf("linux: ${A} must not depend on 'a'; order should be unchanged, got %v", cfgLinux.Envs)
	}
	cfgWin := mismatch()
	if err := cfgWin.SortEnvsByDependency(true); err != nil {
		t.Fatal(err)
	}
	if cfgWin.Envs[0].Key != "a" {
		t.Errorf("windows: ${A} must depend on 'a'; expected 'a' first, got %v", cfgWin.Envs)
	}
}

func TestTrackEnvOverrides_CaseRulesByOS(t *testing.T) {
	configs := []types.Config{
		{SourcePath: "file1.toml", Envs: []types.Environ{{Key: "http_proxy", Value: "1"}}},
		{SourcePath: "file2.toml", Envs: []types.Environ{{Key: "HTTP_PROXY", Value: "2"}}},
	}

	linux := types.TrackEnvOverrides(configs, "linux")
	if len(linux.Vars) != 2 {
		t.Fatalf("linux: expected 2 separate vars, got %v", linux.Vars)
	}
	for k, v := range linux.Vars {
		if v.OverriddenBy != "" {
			t.Errorf("linux: %s should not be an override, got %+v", k, v)
		}
	}

	win := types.TrackEnvOverrides(configs, "windows")
	if len(win.Vars) != 1 {
		t.Fatalf("windows: expected 1 canonical var, got %v", win.Vars)
	}
	src, ok := win.Vars["HTTP_PROXY"]
	if !ok || src.OverriddenBy != "file2.toml" {
		t.Errorf("windows: expected HTTP_PROXY overridden by file2.toml, got %+v", win.Vars)
	}
}
