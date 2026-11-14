package impl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// writeFakeTool creates a file that resolveProgramPath should treat
// as executable on the current OS, and returns its full path. On
// Windows that is a file carrying a PATHEXT extension; on Unix it is
// a file with an executable bit.
func writeFakeTool(t *testing.T, dir, base string) string {
	t.Helper()
	name := base
	if runtime.GOOS == "windows" {
		name = base + ".EXE"
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte("fake"), 0o755); err != nil {
		t.Fatalf("failed to write fake tool: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(full, 0o755); err != nil {
			t.Fatalf("failed to chmod fake tool: %v", err)
		}
	}
	return full
}

func TestResolveProgramPath_FindsToolOnEnvPathOnly(t *testing.T) {
	dir := t.TempDir()
	want := writeFakeTool(t, dir, "eec-envpath-tool")

	// The env PATH is the only place listing dir: the resolver reads
	// env, never the process PATH, so a hit here proves the fix for
	// the `nvim not found in %PATH%` regression.
	env := []string{"PATH=" + dir}
	if got := resolveProgramPath("eec-envpath-tool", env); got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestResolveProgramPath_NotFoundReturnsOriginal(t *testing.T) {
	env := []string{"PATH=" + t.TempDir()}
	if got := resolveProgramPath("eec-no-such-tool-xyz", env); got != "eec-no-such-tool-xyz" {
		t.Errorf("expected original name, got %q", got)
	}
}

func TestResolveProgramPath_NoPathInEnvReturnsOriginal(t *testing.T) {
	if got := resolveProgramPath("anything", []string{"HOME=/nonexistent"}); got != "anything" {
		t.Errorf("expected original name, got %q", got)
	}
}

func TestResolveProgramPath_NameWithSeparatorUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeFakeTool(t, dir, "eec-envpath-tool")
	name := filepath.Join("sub", "eec-envpath-tool")
	env := []string{"PATH=" + dir}
	if got := resolveProgramPath(name, env); got != name {
		t.Errorf("expected separator name unchanged (%q), got %q", name, got)
	}
}

func TestResolveProgramPath_SkipsNonExecutableAndDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable-bit semantics are Unix-only")
	}
	dir := t.TempDir()
	// A directory with the tool's name must not match.
	if err := os.Mkdir(filepath.Join(dir, "eec-tool-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A non-executable file must not match either.
	plain := filepath.Join(dir, "eec-plain-file")
	if err := os.WriteFile(plain, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + dir}
	if got := resolveProgramPath("eec-tool-dir", env); got != "eec-tool-dir" {
		t.Errorf("directory must not resolve, got %q", got)
	}
	if got := resolveProgramPath("eec-plain-file", env); got != "eec-plain-file" {
		t.Errorf("non-executable file must not resolve, got %q", got)
	}
}

func TestResolveProgramPath_WindowsPathExtFromEnv(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATHEXT semantics are Windows-only")
	}
	dir := t.TempDir()
	// A non-default extension that only resolves because the env's
	// PATHEXT lists it (mirrors tools exposed through shims).
	full := filepath.Join(dir, "eec-shimtool.XYZ")
	if err := os.WriteFile(full, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + dir, "PATHEXT=.XYZ;.EXE"}
	got := resolveProgramPath("eec-shimtool", env)
	if !strings.EqualFold(got, full) {
		t.Errorf("expected %q, got %q", full, got)
	}
}
