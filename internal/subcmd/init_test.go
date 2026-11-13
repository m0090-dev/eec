package subcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
)

func TestWriteStarterConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "eec.toml")

	if err := writeStarterConfig(path, false); err != nil {
		t.Fatalf("writeStarterConfig: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	for _, want := range []string{"[envs]", "[envs.windows]", "[envs.linux]", "[envs.darwin]", "[capture]"} {
		if !strings.Contains(body, want) {
			t.Errorf("starter config missing %q", want)
		}
	}

	// The starter must be a valid config as written.
	rt := core.NewEngine(nil).Runtime
	cfg, err := types.ReadConfig(rt, path)
	if err != nil {
		t.Fatalf("starter config does not parse: %v", err)
	}
	found := false
	for _, env := range cfg.Envs {
		if env.Key == "GREETING" {
			found = true
		}
	}
	if !found {
		t.Errorf("starter config envs = %+v, want a GREETING entry", cfg.Envs)
	}
}

func TestWriteStarterConfigRefusesOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "eec.toml")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := writeStarterConfig(path, false); err == nil {
		t.Fatal("overwrite without --force must fail")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "existing" {
		t.Errorf("file was modified despite refusal: %q", data)
	}
	if err := writeStarterConfig(path, true); err != nil {
		t.Fatalf("overwrite with --force: %v", err)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "[envs]") {
		t.Error("--force did not replace the file with the starter config")
	}
}
