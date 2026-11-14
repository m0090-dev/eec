//go:build linux || darwin
// +build linux darwin

package domain_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// runWithStubEec writes script to a temp dir together with a stub
// "eec" executable that echoes its arguments, then runs the script.
func runWithStubEec(t *testing.T, script string, args ...string) string {
	t.Helper()
	dir := t.TempDir()

	stub := "#!/bin/sh\necho \"EEC ARGS: $*\"\n"
	if err := os.WriteFile(filepath.Join(dir, "eec"), []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "tdev")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("sh", append([]string{scriptPath}, args...)...)
	cmd.Env = append(os.Environ(), "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("script failed: %v\noutput: %s", err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestGenUnixTagUtilsScript_ExecutesLikeWindowsVariant(t *testing.T) {
	got := runWithStubEec(t, domain.GenUnixTagUtilsScript("dev"), "prog", "a", "b", "c")
	want := "EEC ARGS: run --tag dev --program prog --program-args=a,b,c"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGenUnixTagUtilsScriptProgramFixed_ExecutesLikeWindowsVariant(t *testing.T) {
	got := runWithStubEec(t, domain.GenUnixTagUtilsScriptProgramFixed("dev"), "x", "y")
	want := "EEC ARGS: run --tag dev --program-args=x,y"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
