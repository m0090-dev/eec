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
	for _, f := range tagData.ImportConfigFiles {
		// ★ここが重要：同じ visited map を渡して再帰する
		fcfg, err := readOrFallbackInternal(opts, rt, f, visited)
		if err != nil {
			// 循環参照エラーなら即座に復帰（logger.Warnで流さず、上位にエラーを伝播させる）
			return cfg, err
		}
		// env = fcfg.BuildEnvs(rt, env, opts.Separator)
		cfg.Envs = append(cfg.Envs, fcfg.Envs...)
	}
	cfg.SourcePath = name
	return cfg, nil
}
