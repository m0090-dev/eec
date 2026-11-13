package domain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Phase 4-3: a relative import stored in a tag (relative to the data
// directory) must resolve against the data directory when the CWD
// does not contain it. Uses the real runtime: EEC_HOME points at a
// temp data dir, and the test runs from a different CWD.
func setupPortableLayout(t *testing.T) (dataDir string) {
	t.Helper()
	root := t.TempDir()
	dataDir = filepath.Join(root, "eec-home")
	cfgDir := filepath.Join(dataDir, "configs")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	cfgBody := "[envs]\nFROM_DATADIR = \"yes\"\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "dev.toml"), []byte(cfgBody), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EEC_HOME", dataDir)

	// Run from an unrelated CWD where "configs/dev.toml" does not exist.
	other := filepath.Join(root, "elsewhere")
	if err := os.MkdirAll(other, 0755); err != nil {
		t.Fatal(err)
	}
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })
	return dataDir
}

func TestResolveRunOptions_DataDirRelativeImport(t *testing.T) {
	setupPortableLayout(t)
	rt := &impl.DefaultRuntime{}

	opts := types.RunOptions{Imports: []string{filepath.Join("configs", "dev.toml")}}
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)
	if env["FROM_DATADIR"] != "yes" {
		t.Errorf("FROM_DATADIR = %q, want \"yes\" (import should resolve against EEC_HOME)", env["FROM_DATADIR"])
	}
}

func TestResolveRunOptions_DataDirRelativeTagConfigFile(t *testing.T) {
	dataDir := setupPortableLayout(t)
	rt := &impl.DefaultRuntime{}

	// TagData as it would be read back from a tag file: the config
	// path is stored relative to the data directory.
	tagData := types.TagData{ConfigFile: filepath.Join("configs", "dev.toml")}
	configFile, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{}, tagData, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	wantCfg := filepath.Join(dataDir, "configs", "dev.toml")
	if configFile != wantCfg {
		t.Errorf("configFile = %q, want %q", configFile, wantCfg)
	}
	if env := finalEnvMapOf(finalEnv); env["FROM_DATADIR"] != "yes" {
		t.Errorf("FROM_DATADIR = %q, want \"yes\"", env["FROM_DATADIR"])
	}
}

func finalEnvMapOf(finalEnv []string) map[string]string {
	m := make(map[string]string, len(finalEnv))
	for _, e := range finalEnv {
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				m[e[:i]] = e[i+1:]
				break
			}
		}
	}
	return m
}
