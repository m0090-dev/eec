package domain_test

import (
	"path/filepath"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// Tag-in-tag composition (an aggregate tag importing other tags by
// name) used to flatten ONLY the inner tags' envs: their [capture]
// commands were silently dropped, so a vcvars-style capture defined
// in an inner tag never ran in the aggregate. These tests pin the
// fixed behavior: composing a tag by name must behave like importing
// the files it resolves to.

func setupTagHome(t *testing.T) string {
	t.Helper()
	dataDir := filepath.Join(t.TempDir(), "eec-home")
	t.Setenv("EEC_HOME", dataDir)
	return dataDir
}

func writeTag(t *testing.T, rt *impl.DefaultRuntime, name string, data types.TagData) {
	t.Helper()
	td := data
	if err := td.Write(rt, name); err != nil {
		t.Fatalf("failed to write tag %q: %v", name, err)
	}
}

func TestReadOrFallbackRecursive_TagInTagCarriesCaptureAndConfigs(t *testing.T) {
	setupTagHome(t)
	rt := &impl.DefaultRuntime{}

	// Inner config carries a capture and a [configs] toggle.
	cfgPath := writeCaptureFixture(t, "[configs]\nexpand_vars = false\n")
	writeTag(t, rt, "inner-vc", types.TagData{ImportConfigFiles: []string{cfgPath}})

	cfg, err := domain.ReadOrFallbackRecursive(types.RunOptions{}, rt, "inner-vc")
	if err != nil {
		t.Fatalf("ReadOrFallbackRecursive: %v", err)
	}
	if len(cfg.CaptureCommands) != 2 {
		t.Errorf("CaptureCommands = %v, want the inner config's 2 commands carried through the tag", cfg.CaptureCommands)
	}
	foundToggle := false
	for _, mc := range cfg.Configs {
		if mc.ExpandVars != nil && !*mc.ExpandVars {
			foundToggle = true
		}
	}
	if !foundToggle {
		t.Error("[configs] expand_vars=false from the inner config was not carried through the tag")
	}
	if len(cfg.Envs) == 0 {
		t.Error("Envs empty; the inner config's envs must still be flattened in")
	}
}

func TestResolveRunOptions_CaptureRunsThroughTagInTag(t *testing.T) {
	setupTagHome(t)
	rt := &impl.DefaultRuntime{}

	cfgPath := writeCaptureFixture(t, "")
	writeTag(t, rt, "inner-vc", types.TagData{ImportConfigFiles: []string{cfgPath}})

	// The aggregate tag composes the inner tag BY NAME — the shape
	// of real setups such as `dev` = dev-base + dev-lang + win-dev.
	outer := types.TagData{ImportConfigFiles: []string{"inner-vc"}}
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{}, outer, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)
	if got := env["EEC_CAP_NEW"]; got != "brand-new" {
		t.Errorf("EEC_CAP_NEW = %q, want %q (capture from the inner tag must run)", got, "brand-new")
	}
	if got := env["KEEP_ME"]; got != "untouched" {
		t.Errorf("KEEP_ME = %q, want %q (inner config envs must be present)", got, "untouched")
	}
}

func TestResolveRunOptions_CaptureRunsFromInnerTagConfigFile(t *testing.T) {
	setupTagHome(t)
	rt := &impl.DefaultRuntime{}

	cfgPath := writeCaptureFixture(t, "")
	// The capture lives in the inner tag's MAIN config file (-c),
	// not in its imports: flattening must follow ConfigFile too.
	writeTag(t, rt, "inner-main", types.TagData{ConfigFile: cfgPath})

	outer := types.TagData{ImportConfigFiles: []string{"inner-main"}}
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(types.RunOptions{}, outer, rt)
	if err != nil {
		t.Fatalf("ResolveRunOptions: %v", err)
	}
	env := finalEnvMapOf(finalEnv)
	if got := env["EEC_CAP_NEW"]; got != "brand-new" {
		t.Errorf("EEC_CAP_NEW = %q, want %q (capture from the inner tag's config file must run)", got, "brand-new")
	}
}
