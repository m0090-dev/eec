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

func findCheck(checks []domain.DoctorCheck, status domain.DoctorStatus, labelPart string) *domain.DoctorCheck {
	for i := range checks {
		if checks[i].Status == status && strings.Contains(checks[i].Label, labelPart) {
			return &checks[i]
		}
	}
	return nil
}

func TestRunDoctor_HealthyConfigWithProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix executable-bit semantics")
	}
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.Mkdir(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeFileWithMode(t, filepath.Join(binDir, "doctor-tool"), 0755)
	missingDir := filepath.Join(dir, "missing-bin")

	cfg := writeTempFile(t, dir, "dev.toml",
		"[envs]\nPATH = [\""+binDir+"\", \""+missingDir+"\"]\n")

	checks := domain.RunDoctor(rt, types.RunOptions{ConfigFile: cfg}, types.TagData{}, "doctor-tool")

	if c := findCheck(checks, domain.DoctorOK, "configs resolve"); c == nil {
		t.Errorf("expected OK 'configs resolve', got %+v", checks)
	}
	// The missing PATH entry is a warning, never a failure.
	if c := findCheck(checks, domain.DoctorWarn, "PATH entry does not exist"); c == nil || c.Detail != missingDir {
		t.Errorf("expected warning for missing PATH entry %q, got %+v", missingDir, checks)
	}
	if c := findCheck(checks, domain.DoctorOK, "program found: doctor-tool"); c == nil {
		t.Errorf("expected OK program check, got %+v", checks)
	}
	if domain.DoctorFailed(checks) {
		t.Errorf("warnings must not fail doctor: %+v", checks)
	}
}

func TestRunDoctor_MissingProgramFails(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	cfg := writeTempFile(t, dir, "dev.toml", "[envs]\nFOO = \"bar\"\n")

	checks := domain.RunDoctor(rt, types.RunOptions{ConfigFile: cfg}, types.TagData{}, "no-such-program-eec-test")
	if c := findCheck(checks, domain.DoctorFail, "program not found"); c == nil {
		t.Errorf("expected FAIL program check, got %+v", checks)
	}
	if !domain.DoctorFailed(checks) {
		t.Error("DoctorFailed must be true when a program check fails")
	}
}

func TestRunDoctor_BrokenConfigFailsResolution(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	bad := writeTempFile(t, dir, "bad.toml", "[envs\nFOO = \n")

	checks := domain.RunDoctor(rt, types.RunOptions{ConfigFile: bad}, types.TagData{}, "")
	if len(checks) != 1 || checks[0].Status != domain.DoctorFail {
		t.Fatalf("broken config must yield a single FAIL check, got %+v", checks)
	}
	if !domain.DoctorFailed(checks) {
		t.Error("DoctorFailed must be true")
	}
}

func TestDoctorStatusSymbols(t *testing.T) {
	if domain.DoctorOK.Symbol() != "✓" || domain.DoctorWarn.Symbol() != "⚠" || domain.DoctorFail.Symbol() != "✗" {
		t.Error("unexpected doctor status symbols")
	}
}
