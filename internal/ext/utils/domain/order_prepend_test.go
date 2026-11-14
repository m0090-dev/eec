package domain_test

import (
	"os"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Phase 6-1: multi-config composition of merge sections falls out of
// ResolveRunOptions' sequential BuildEnvs calls: config1 appends,
// config2 prepends, so the final value is
// cfg2-prepend ++ base ++ cfg1-append (with dedup).
func TestResolveRunOptions_PrependAppendAcrossConfigs(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	dir := t.TempDir()

	cfg1 := writeTempFile(t, dir, "one.toml", `
[envs]
PATH = ["cfg1-appended"]

[envs.append]
EXTRA = ["cfg1-extra"]
`)
	cfg2 := writeTempFile(t, dir, "two.toml", `
[envs.prepend]
PATH = ["cfg2-first", "cfg1-appended"]
`)

	sep := string(os.PathListSeparator)
	base := "base0" + sep + "base1"
	t.Setenv("PATH", base)

	tagData := types.TagData{
		ConfigFile:        cfg1,
		ImportConfigFiles: []string{cfg2},
	}
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{}, tagData, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)
	// cfg2's prepend wins the front; its duplicate of cfg1's appended
	// part also moves that part to the front; base stays in the
	// middle; cfg1's append is not repeated at the back.
	want := "cfg2-first" + sep + "cfg1-appended" + sep + base
	if env["PATH"] != want {
		t.Errorf("PATH = %q, want %q", env["PATH"], want)
	}
	if env["EXTRA"] != "cfg1-extra" {
		t.Errorf("EXTRA = %q, want %q", env["EXTRA"], "cfg1-extra")
	}
}
