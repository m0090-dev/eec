package domain_test

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// --- ExeCandidates / PathExtList: pure, GOOS-parameterized ---

func TestExeCandidates_WindowsDefaultPathExt(t *testing.T) {
	got := domain.ExeCandidates("windows", []string{"PATH=C:\\bin"}, "tool")
	want := []string{"tool.COM", "tool.EXE", "tool.BAT", "tool.CMD"}
	if !slices.Equal(got, want) {
		t.Errorf("ExeCandidates = %v, want %v", got, want)
	}
}

func TestExeCandidates_WindowsPathExtFromEnv(t *testing.T) {
	env := []string{"PATHEXT=.EXE;BAT;.PS1"}
	got := domain.ExeCandidates("windows", env, "tool")
	want := []string{"tool.EXE", "tool.BAT", "tool.PS1"}
	if !slices.Equal(got, want) {
		t.Errorf("ExeCandidates = %v, want %v", got, want)
	}
}

func TestExeCandidates_WindowsKeyCaseInsensitive(t *testing.T) {
	// Windows env keys are case-insensitive: a lowercase "pathext"
	// entry in the searched environment must be honored.
	env := []string{"pathext=.XYZ"}
	got := domain.ExeCandidates("windows", env, "tool")
	want := []string{"tool.XYZ"}
	if !slices.Equal(got, want) {
		t.Errorf("ExeCandidates = %v, want %v", got, want)
	}
}

func TestExeCandidates_WindowsNameWithExtensionUsedAsIs(t *testing.T) {
	for _, name := range []string{"tool.EXE", "tool.txt", "tool.bat"} {
		got := domain.ExeCandidates("windows", nil, name)
		if !slices.Equal(got, []string{name}) {
			t.Errorf("ExeCandidates(%q) = %v, want [%q]", name, got, name)
		}
	}
}

func TestExeCandidates_Unix(t *testing.T) {
	got := domain.ExeCandidates("linux", []string{"PATHEXT=.EXE"}, "tool")
	if !slices.Equal(got, []string{"tool"}) {
		t.Errorf("ExeCandidates = %v, want [tool]", got)
	}
	if exts := domain.PathExtList("linux", nil); exts != nil {
		t.Errorf("PathExtList(linux) = %v, want nil", exts)
	}
}

// --- FindInPath on the real (Unix) file system ---

func writeFileWithMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), mode); err != nil {
		t.Fatal(err)
	}
}

func TestFindInPath_UnixRequiresExecutableBit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix executable-bit semantics")
	}
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	tool := filepath.Join(dir, "mytool")
	writeFileWithMode(t, tool, 0644)

	env := []string{"PATH=" + dir}
	if _, ok := domain.FindInPath(rt, env, "mytool"); ok {
		t.Fatal("non-executable file must not be found")
	}

	if err := os.Chmod(tool, 0755); err != nil {
		t.Fatal(err)
	}
	found, ok := domain.FindInPath(rt, env, "mytool")
	if !ok {
		t.Fatal("executable file not found in PATH")
	}
	if found != tool {
		t.Errorf("found = %q, want %q", found, tool)
	}
}

func TestFindInPath_UnixSkipsDirectoriesAndMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix semantics")
	}
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "somedir"), 0755); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + dir}
	if _, ok := domain.FindInPath(rt, env, "somedir"); ok {
		t.Error("a directory must not count as an executable")
	}
	if _, ok := domain.FindInPath(rt, env, "nope"); ok {
		t.Error("missing program reported found")
	}
	if _, ok := domain.FindInPath(rt, []string{}, "anything"); ok {
		t.Error("empty PATH must not find anything")
	}
}

func TestFindInPath_DirectPathWithSeparator(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix semantics")
	}
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	tool := filepath.Join(dir, "direct")
	writeFileWithMode(t, tool, 0755)

	found, ok := domain.FindInPath(rt, []string{"PATH=/nonexistent"}, tool)
	if !ok || found != tool {
		t.Errorf("direct path lookup = (%q, %v), want (%q, true)", found, ok, tool)
	}
	if _, ok := domain.FindInPath(rt, nil, filepath.Join(dir, "missing")); ok {
		t.Error("missing direct path reported found")
	}
}

func TestFindInPath_FirstPathEntryWins(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix semantics")
	}
	rt := &impl.DefaultRuntime{}
	d1, d2 := t.TempDir(), t.TempDir()
	writeFileWithMode(t, filepath.Join(d1, "dup"), 0755)
	writeFileWithMode(t, filepath.Join(d2, "dup"), 0755)
	env := []string{"PATH=" + d1 + ":" + d2}
	found, ok := domain.FindInPath(rt, env, "dup")
	if !ok || found != filepath.Join(d1, "dup") {
		t.Errorf("found = (%q, %v), want first PATH entry", found, ok)
	}
}
