package types

import (
	"fmt"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"sort"
)

// EnvDefinition は1回分の変数定義を表す（どのファイルが、どんな値を、
// どのセクションで定義したか）。上書きチェーンの表示に使う。
type EnvDefinition struct {
	Source  string      // 定義した設定ファイル ("<inline>" の場合もある)
	Value   interface{} // 設定に書かれた生の値
	Prepend bool        // [envs.prepend] 由来なら true
	Section string      // "", "append", "prepend"
	Origin  string      // OS セクション名 (共通なら "")
}

// EnvSource は環境変数の定義元と上書き情報を保持する。
type EnvSource struct {
	Value        string
	DefinedIn    string // 初回定義ファイル
	OverriddenBy string // 上書きしたファイル（なければ空文字）
	// History は解決順に並んだ全定義（初回定義を含む）。
	History []EnvDefinition
}

// EnvTracker は全環境変数の追跡情報を保持する。
type EnvTracker struct {
	Vars map[string]*EnvSource
}

// NewEnvTracker は空の EnvTracker を返す。
func NewEnvTracker() *EnvTracker {
	return &EnvTracker{Vars: make(map[string]*EnvSource)}
}

// TrackEnvOverrides は複数の Config を順に処理し、上書きの発生を追跡する。
// 引数の configs は優先度が低い順（後ろほど強い）で渡すこと。
// goos は対象 OS で、キー比較のケース規則を決める (CanonicalEnvKey 参照)。
// Windows では "Path" の再定義が "PATH" の上書きとして検出され、
// Linux/macOS ではケースの異なる変数は別変数として扱われる。
func TrackEnvOverrides(configs []Config, goos string) *EnvTracker {
	tracker := NewEnvTracker()
	for _, cfg := range configs {
		source := cfg.SourcePath
		if source == "" {
			source = "<inline>"
		}
		if cfg.SourcePath == "" && cfg.RawEnvs == nil {
			continue
		}
		for _, env := range cfg.Envs {
			key := CanonicalEnvKey(goos, env.Key)
			val := fmt.Sprint(env.Value)
			def := EnvDefinition{
				Source:  source,
				Value:   env.Value,
				Prepend: env.Prepend,
				Section: env.Section,
				Origin:  env.Origin,
			}
			if existing, ok := tracker.Vars[key]; ok {
				// すでに定義済み → 上書き記録
				existing.OverriddenBy = source
				existing.Value = val
				existing.History = append(existing.History, def)
			} else {
				tracker.Vars[key] = &EnvSource{
					Value:     val,
					DefinedIn: source,
					History:   []EnvDefinition{def},
				}
			}
		}
	}
	return tracker
}

// OverriddenKeys は複数回定義された（=上書きされた）変数のキーを
// ソートして返す。
func (t *EnvTracker) OverriddenKeys() []string {
	var keys []string
	for key, src := range t.Vars {
		if len(src.History) > 1 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// PrintOverrides はログや標準出力に上書きされた変数の一覧を出す。
// strict=true の場合はエラーを返す。noWarn=true の場合は何もしない。
func (t *EnvTracker) PrintOverrides(logger interfaces.Logger, strict bool, noWarn bool) error {
	if noWarn {
		return nil
	}
	var overridden []string
	for key, src := range t.Vars {
		if src.OverriddenBy != "" {
			overridden = append(overridden, key)
		}
	}
	if len(overridden) == 0 {
		return nil
	}
	sort.Strings(overridden)
	for _, key := range overridden {
		src := t.Vars[key]
		msg := fmt.Sprintf("env override: %s (defined in %s, overridden by %s)",
			key, src.DefinedIn, src.OverriddenBy)
		if strict {
			logger.Error().Msg(msg)
		} else {
			logger.Warn().Msg(msg)
		}
	}
	if strict {
		return fmt.Errorf("%d environment variable(s) were overridden (--duplicate-strict)", len(overridden))
	}
	return nil
}
