package types_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
)

func TestNormalizeEnvs_ListItemMultipleKeys_AllKeptSorted(t *testing.T) {
	cfg := types.Config{RawEnvs: []interface{}{
		map[string]interface{}{"B": "2", "A": "1", "C": "3"},
	}}
	if err := cfg.NormalizeEnvs(newMockRuntime()); err != nil {
		t.Fatalf("NormalizeEnvs: %v", err)
	}
	if len(cfg.Envs) != 3 {
		t.Fatalf("expected all 3 keys kept, got %d: %v", len(cfg.Envs), cfg.Envs)
	}
	want := []string{"A", "B", "C"}
	for i, k := range want {
		if cfg.Envs[i].Key != k {
			t.Errorf("Envs[%d].Key = %q, want %q (deterministic sorted order)", i, cfg.Envs[i].Key, k)
		}
	}
}

func TestNormalizeEnvs_MapFormat_DeterministicOrder(t *testing.T) {
	cfg := types.Config{RawEnvs: map[string]interface{}{"Z": "1", "A": "2", "M": "3"}}
	if err := cfg.NormalizeEnvs(newMockRuntime()); err != nil {
		t.Fatalf("NormalizeEnvs: %v", err)
	}
	want := []string{"A", "M", "Z"}
	if len(cfg.Envs) != len(want) {
		t.Fatalf("expected %d envs, got %d", len(want), len(cfg.Envs))
	}
	for i, k := range want {
		if cfg.Envs[i].Key != k {
			t.Errorf("Envs[%d].Key = %q, want %q", i, cfg.Envs[i].Key, k)
		}
	}
}

func TestSortEnvsByDependency_CircularReportsChain(t *testing.T) {
	cfg := types.Config{Envs: []types.Environ{
		{Key: "A", Value: "${B}"},
		{Key: "B", Value: "${A}"},
	}}
	err := cfg.SortEnvsByDependency(false)
	if err == nil {
		t.Fatal("expected circular reference error, got nil")
	}
	if !strings.Contains(err.Error(), "A -> B -> A") {
		t.Errorf("error should contain the dependency chain %q, got: %v", "A -> B -> A", err)
	}
}

func TestReadConfig_UnknownExtension_ReturnsError(t *testing.T) {
	if _, err := types.ReadConfig(newMockRuntime(), "config.ini"); err == nil {
		t.Error("expected error for unknown extension, got nil")
	}
}

func TestReadEnv_MissingFile_ReturnsError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist.env")
	if _, err := types.ReadEnv(newMockRuntime(), missing); err == nil {
		t.Error("expected error for missing .env file, got nil (previously this called Logger.Fatal)")
	}
}

func TestReadEnv_ValidFile_SortedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.env")
	if err := os.WriteFile(path, []byte("ZULU=1\nALPHA=2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := types.ReadEnv(newMockRuntime(), path)
	if err != nil {
		t.Fatalf("ReadEnv: %v", err)
	}
	if len(cfg.Envs) != 2 || cfg.Envs[0].Key != "ALPHA" || cfg.Envs[1].Key != "ZULU" {
		t.Errorf("expected sorted [ALPHA ZULU], got %v", cfg.Envs)
	}
}
