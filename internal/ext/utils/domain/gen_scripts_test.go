package domain_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Launcher script improvements: scripts generated from a portable
// tag store (EEC_HOME / exe-adjacent .eec) embed that data directory
// so they resolve the same tags from shells where EEC_HOME is unset;
// Windows also gets a PowerShell variant whose arguments survive
// spaces and commas; running a launcher with no arguments opens the
// tag's shell.

const testEecHome = `D:\win\program\eec`

func TestGenWindowsTagUtilsScript_EmbedsEecHome(t *testing.T) {
	script := domain.GenWindowsTagUtilsScript("dev", testEecHome)
	want := `if not defined EEC_HOME set "EEC_HOME=` + testEecHome + `"`
	if !strings.Contains(script, want) {
		t.Errorf("script should embed EEC_HOME line %q, got:\n%s", want, script)
	}
	if strings.Contains(script, "%EECHOME_LINE%") || strings.Contains(script, "%TAGNAME%") {
		t.Error("placeholders were not fully replaced")
	}
}

func TestGenWindowsTagUtilsScript_NoEecHomeLineWhenEmpty(t *testing.T) {
	script := domain.GenWindowsTagUtilsScript("dev", "")
	if strings.Contains(script, "EEC_HOME") {
		t.Errorf("script should not mention EEC_HOME when no portable dir, got:\n%s", script)
	}
}

func TestGenWindowsTagUtilsScript_NoArgsOpensShellAndPropagatesExitCode(t *testing.T) {
	script := domain.GenWindowsTagUtilsScript("dev", "")
	if !strings.Contains(script, "eec shell --tag dev") {
		t.Errorf("no-arg launcher should open the tag shell, got:\n%s", script)
	}
	if !strings.Contains(script, "exit /b %ERRORLEVEL%") {
		t.Errorf("launcher should propagate eec's exit code, got:\n%s", script)
	}
	fixed := domain.GenWindowsTagUtilsScriptProgramFixed("dev", testEecHome)
	if !strings.Contains(fixed, "exit /b %ERRORLEVEL%") {
		t.Errorf("fixed launcher should propagate eec's exit code, got:\n%s", fixed)
	}
	if !strings.Contains(fixed, `set "EEC_HOME=`+testEecHome+`"`) {
		t.Errorf("fixed launcher should embed EEC_HOME, got:\n%s", fixed)
	}
}

func TestGenPowerShellTagUtilsScript_Variants(t *testing.T) {
	script := domain.GenPowerShellTagUtilsScript("dev", testEecHome)
	for _, want := range []string{
		`if (-not $env:EEC_HOME) { $env:EEC_HOME = "` + testEecHome + `" }`,
		"ConvertTo-CsvField",
		"eec shell --tag dev",
		"eec run --tag dev --program $program --program-args $csv",
		"exit $LASTEXITCODE",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("ps1 script should contain %q, got:\n%s", want, script)
		}
	}

	fixed := domain.GenPowerShellTagUtilsScriptProgramFixed("dev", "")
	if strings.Contains(fixed, "EEC_HOME") {
		t.Errorf("fixed ps1 should not mention EEC_HOME when no portable dir, got:\n%s", fixed)
	}
	if !strings.Contains(fixed, "eec run --tag dev --program-args $csv") {
		t.Errorf("fixed ps1 should pass args through, got:\n%s", fixed)
	}
}

func TestGenUnixTagUtilsScript_EmbedsEecHomeAndShellFallback(t *testing.T) {
	script := domain.GenUnixTagUtilsScript("dev", "/mnt/drive/eec-home")
	if !strings.Contains(script, `EEC_HOME:=`) || !strings.Contains(script, "/mnt/drive/eec-home") {
		t.Errorf("sh script should embed EEC_HOME default, got:\n%s", script)
	}
	if !strings.Contains(script, "exec eec shell --tag dev") {
		t.Errorf("no-arg sh launcher should open the tag shell, got:\n%s", script)
	}
	plain := domain.GenUnixTagUtilsScript("dev", "")
	if strings.Contains(plain, "EEC_HOME") {
		t.Errorf("sh script should not mention EEC_HOME when no portable dir, got:\n%s", plain)
	}
}

// GenUtilsScript on a portable store writes the host-OS launcher
// with the data directory baked in: tdev.sh on Unix, tdev.cmd (plus
// the tdev.ps1 variant) on Windows.
func TestGenUtilsScript_BakesPortableDataDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "eec-home")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EEC_HOME", dataDir)
	rt := &impl.DefaultRuntime{}

	td := types.TagData{Program: "prog"}
	if err := td.Write(rt, "dev"); err != nil {
		t.Fatalf("failed to write tag: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "scripts")
	domain.GenUtilsScript(rt, outDir, false)

	launcher := "tdev.sh"
	if runtime.GOOS == "windows" {
		launcher = "tdev.cmd"
	}
	body, err := os.ReadFile(filepath.Join(outDir, launcher))
	if err != nil {
		t.Fatalf("generated script not found: %v", err)
	}
	if !strings.Contains(string(body), dataDir) {
		t.Errorf("generated script should embed the data dir %q, got:\n%s", dataDir, body)
	}

	if runtime.GOOS == "windows" {
		ps, err := os.ReadFile(filepath.Join(outDir, "tdev.ps1"))
		if err != nil {
			t.Fatalf("generated ps1 script not found: %v", err)
		}
		if !strings.Contains(string(ps), dataDir) {
			t.Errorf("generated ps1 should embed the data dir %q, got:\n%s", dataDir, ps)
		}
	}
}

func TestPortableDataDir_FromEnv(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "eec-home")
	t.Setenv("EEC_HOME", dataDir)
	rt := &impl.DefaultRuntime{}
	if got := types.PortableDataDir(rt); got != dataDir {
		t.Errorf("PortableDataDir = %q, want %q", got, dataDir)
	}
}
