package domain_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

func writeTempFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestValidateSources_ValidConfigOK(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	good := writeTempFile(t, dir, "good.toml", "[envs]\nFOO = \"bar\"\nBAR = \"${FOO}-baz\"\n")

	results := domain.ValidateSources(rt, types.RunOptions{ConfigFile: good}, types.TagData{})
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Err != nil {
		t.Errorf("valid config reported NG: %v", results[0].Err)
	}
	if results[0].Source != good {
		t.Errorf("source = %q, want %q", results[0].Source, good)
	}
}

func TestValidateSources_BrokenConfigNG(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()
	bad := writeTempFile(t, dir, "bad.toml", "[envs\nFOO = \n")

	results := domain.ValidateSources(rt, types.RunOptions{ConfigFile: bad}, types.TagData{})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("broken config must be NG, got %+v", results)
	}
}

func TestValidateSources_MissingFileNG(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	missing := filepath.Join(t.TempDir(), "nope.toml")
	results := domain.ValidateSources(rt, types.RunOptions{Imports: []string{missing}}, types.TagData{})
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("missing import must be NG, got %+v", results)
	}
}

func TestValidateSources_CircularTagImportsNG(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dataDir := t.TempDir()
	t.Setenv("EEC_HOME", dataDir)
	writeTempFile(t, dataDir, "loopy-a.tag", `{"import_config_files":["loopy-b"]}`)
	writeTempFile(t, dataDir, "loopy-b.tag", `{"import_config_files":["loopy-a"]}`)

	err := domain.ValidateSource(rt, types.RunOptions{}, "loopy-a")
	if err == nil {
		t.Fatal("circular tag imports must fail validation")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("error = %v, want a circular-import error", err)
	}
}

func TestConfigSources_Order(t *testing.T) {
	opts := types.RunOptions{ConfigFile: "main.toml", Imports: []string{"i1.toml", "i2.toml"}}
	tag := types.TagData{ConfigFile: "tag.toml", ImportConfigFiles: []string{"t1.toml"}}
	got := domain.ConfigSources(opts, tag)
	want := []string{"main.toml", "t1.toml", "i1.toml", "i2.toml"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ConfigSources = %v, want %v (-c wins over tag config file)", got, want)
	}
	// Without -c, the tag's config file is the first source.
	got = domain.ConfigSources(types.RunOptions{}, tag)
	want = []string{"tag.toml", "t1.toml"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ConfigSources = %v, want %v", got, want)
	}
}
