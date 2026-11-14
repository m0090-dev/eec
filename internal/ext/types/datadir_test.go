package types_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// Phase 4-1: ResolveDataDir precedence — EEC_HOME > portable
// (exe-adjacent .eec) > ~/.eec fallback.

func withExeDir(t *testing.T, dir string) {
	t.Helper()
	orig := types.ExecutableDirFunc
	types.ExecutableDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { types.ExecutableDirFunc = orig })
}

func TestResolveDataDir_EecHomeWins(t *testing.T) {
	rt := newMockRuntime()
	rt.env.homeDir = "/home/tester"
	rt.env.lookupMap["EEC_HOME"] = "/data/custom"

	// Even with a portable dir present, EEC_HOME has priority.
	exeDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(exeDir, ".eec"), 0755); err != nil {
		t.Fatal(err)
	}
	withExeDir(t, exeDir)

	if got := types.ResolveDataDir(rt); got != "/data/custom" {
		t.Errorf("ResolveDataDir = %q, want EEC_HOME value %q", got, "/data/custom")
	}
}

func TestResolveDataDir_PortableMode(t *testing.T) {
	rt := newMockRuntime()
	rt.env.homeDir = "/home/tester"

	exeDir := t.TempDir()
	portable := filepath.Join(exeDir, ".eec")
	if err := os.MkdirAll(portable, 0755); err != nil {
		t.Fatal(err)
	}
	withExeDir(t, exeDir)

	if got := types.ResolveDataDir(rt); got != portable {
		t.Errorf("ResolveDataDir = %q, want portable dir %q", got, portable)
	}
}

func TestResolveDataDir_HomeFallback(t *testing.T) {
	rt := newMockRuntime()
	home := t.TempDir()
	rt.env.homeDir = home

	// Executable dir without a .eec sibling -> home fallback.
	withExeDir(t, t.TempDir())

	want := filepath.Join(home, ".eec")
	if got := types.ResolveDataDir(rt); got != want {
		t.Errorf("ResolveDataDir = %q, want %q", got, want)
	}
}

func TestRelativizeDataDirPath(t *testing.T) {
	rt := newMockRuntime()
	dataDir := t.TempDir()
	rt.env.lookupMap["EEC_HOME"] = dataDir
	withExeDir(t, t.TempDir())

	inside := filepath.Join(dataDir, "configs", "dev.toml")
	if got := types.RelativizeDataDirPath(rt, inside); got != filepath.Join("configs", "dev.toml") {
		t.Errorf("inside path relativized to %q, want %q", got, filepath.Join("configs", "dev.toml"))
	}

	outside := filepath.Join(t.TempDir(), "elsewhere.toml")
	if got := types.RelativizeDataDirPath(rt, outside); got != outside {
		t.Errorf("outside path = %q, want unchanged %q", got, outside)
	}

	if got := types.RelativizeDataDirPath(rt, "relative.toml"); got != "relative.toml" {
		t.Errorf("relative path = %q, want unchanged", got)
	}
	if got := types.RelativizeDataDirPath(rt, ""); got != "" {
		t.Errorf("empty path = %q, want empty", got)
	}
}
