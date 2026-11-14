package domain

import (
	"fmt"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"path/filepath"
)

func ReadOrFallback(opts types.RunOptions, rt interfaces.Runtime, name string) (types.Config, error) {
	return readOrFallbackInternal(opts, rt, name, make(map[string]bool))
}

func ReadOrFallbackRecursive(opts types.RunOptions, rt interfaces.Runtime, name string) (types.Config, error) {
	return readOrFallbackInternal(opts, rt, name, make(map[string]bool))
}

// ResolveDataDirRelative retries a relative path against the
// resolved data directory (EEC_HOME / portable / ~/.eec). Tag files
// store imports inside the data directory in relative form so they
// survive the directory moving (Phase 4); this is where those
// relative imports find their way back. Paths that exist relative to
// the CWD, absolute paths, and paths found nowhere are returned
// unchanged.
func ResolveDataDirRelative(rt interfaces.Runtime, name string) string {
	if name == "" || filepath.IsAbs(name) {
		return name
	}
	if rt.FS().FileExists(name) {
		return name
	}
	dataDir := types.ResolveDataDir(rt)
	if dataDir == "" {
		return name
	}
	candidate := filepath.Join(dataDir, name)
	if rt.FS().FileExists(candidate) {
		return candidate
	}
	return name
}

func readOrFallbackInternal(opts types.RunOptions, rt interfaces.Runtime, name string, visited map[string]bool) (types.Config, error) {
	// データディレクトリ相対の import を先に解決する
	name = ResolveDataDirRelative(rt, name)

	// 循環チェック
	absPath, _ := filepath.Abs(name)
	if visited[absPath] {
		return types.Config{}, fmt.Errorf("circular import detected: %s", absPath)
	}
	visited[absPath] = true

	var cfg types.Config
	if rt.FS().FileExists(name) {
		return types.ReadConfig(rt, name)
	}

	tagData, err := types.ReadTagData(rt, name)
	if err != nil {
		return cfg, err
	}

	// Compose the tag's sources in the same order ResolveRunOptions
	// uses for a top-level tag: the tag's own config file first, then
	// its imports. A config file that is also listed as an import is
	// only composed once.
	sources := make([]string, 0, len(tagData.ImportConfigFiles)+1)
	if tagData.ConfigFile != "" {
		sources = append(sources, tagData.ConfigFile)
	}
	for _, imp := range tagData.ImportConfigFiles {
		if tagData.ConfigFile != "" && samePath(tagData.ConfigFile, imp) {
			continue
		}
		sources = append(sources, imp)
	}

	for _, f := range sources {
		// ★ここが重要：同じ visited map を渡して再帰する
		fcfg, err := readOrFallbackInternal(opts, rt, f, visited)
		if err != nil {
			// 循環参照エラーなら即座に復帰（logger.Warnで流さず、上位にエラーを伝播させる）
			return cfg, err
		}
		// Flatten EVERYTHING a config contributes, not just its
		// envs. Merging only Envs silently dropped the inner tag's
		// [capture] commands (a vcvars capture in an inner tag never
		// ran when the tag was composed into an aggregate tag), its
		// [configs] toggles, and its [program].
		cfg.Envs = append(cfg.Envs, fcfg.Envs...)
		cfg.Configs = append(cfg.Configs, fcfg.Configs...)
		cfg.CaptureCommands = append(cfg.CaptureCommands, fcfg.CaptureCommands...)
		if cfg.Program.Path == "" && fcfg.Program.Path != "" {
			cfg.Program = fcfg.Program
		}
	}
	cfg.SourcePath = name
	return cfg, nil
}

// samePath reports whether two config source references point at the
// same file, comparing absolute forms so differently-spelled
// references to one file still match.
func samePath(a, b string) bool {
	if a == b {
		return true
	}
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}
