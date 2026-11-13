package core_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	core "github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// Phase 4-3: TagAdd stores imports that live inside the data
// directory as paths relative to the data directory, so a tag
// survives the data directory moving (portable drives).
func TestEngine_TagAdd_StoresDataDirRelativePaths(t *testing.T) {
	rt := newCoreRuntime()
	dataDir := filepath.Join(t.TempDir(), "eec-home")
	rt.env.lookupMap["EEC_HOME"] = dataDir

	insideCfg := filepath.Join(dataDir, "configs", "dev.toml")
	insideImp := filepath.Join(dataDir, "configs", "extra.toml")
	outsideCfg := filepath.Join(t.TempDir(), "outside.toml")
	rt.fs.existFiles[insideCfg] = true
	rt.fs.existFiles[insideImp] = true
	rt.fs.existFiles[outsideCfg] = true

	e := core.NewEngine(rt)
	tag := types.TagData{
		ConfigFile:        insideCfg,
		Program:           "prog",
		ImportConfigFiles: []string{insideImp, outsideCfg},
	}
	if err := e.TagAdd("portabletag", tag); err != nil {
		t.Fatalf("TagAdd: %v", err)
	}

	raw, ok := rt.fs.fileData[filepath.Join(dataDir, "portabletag.tag")]
	if !ok {
		t.Fatalf("tag file was not written under the data dir; writes: %v", rt.fs.fileData)
	}
	var stored types.TagData
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("stored tag is not valid JSON: %v", err)
	}

	wantCfg := filepath.Join("configs", "dev.toml")
	if stored.ConfigFile != wantCfg {
		t.Errorf("stored ConfigFile = %q, want data-dir-relative %q", stored.ConfigFile, wantCfg)
	}
	wantImp := filepath.Join("configs", "extra.toml")
	if len(stored.ImportConfigFiles) != 2 {
		t.Fatalf("stored imports = %v, want 2 entries", stored.ImportConfigFiles)
	}
	if stored.ImportConfigFiles[0] != wantImp {
		t.Errorf("stored import[0] = %q, want data-dir-relative %q", stored.ImportConfigFiles[0], wantImp)
	}
	if stored.ImportConfigFiles[1] != outsideCfg {
		t.Errorf("stored import[1] = %q, want unchanged absolute %q (outside the data dir)",
			stored.ImportConfigFiles[1], outsideCfg)
	}
}
