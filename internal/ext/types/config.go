package types

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

type Config struct {
	RawConfigs interface{}  `toml:"configs" yaml:"configs" json:"configs"`
	Configs    []MetaConfig `toml:"-" yaml:"-" json:"-"`
	RawEnvs    interface{}  `toml:"envs" yaml:"envs" json:"envs"`
	Envs       []Environ    `toml:"-" yaml:"-" json:"-"`
	Program    ProgramData  `toml:"program" yaml:"program" json:"program"`
	SourcePath string       `json:"-" yaml:"-" toml:"-"`
}
type MetaConfig struct {
	Separator   string `toml:"separator" yaml:"separator" json:"separator"`
	Description string `toml:"description" yaml:"description" json:"description"`
}
type ProgramData struct {
	Path string   `toml:"path" yaml:"path" json:"path"`
	Args []string `toml:"args" yaml:"args" json:"args"`
}
type Environ struct {
	Key   string      `toml:"key" yaml:"key" json:"key"`
	Value interface{} `toml:"value" yaml:"value" json:"value"`
}

// extractDependencies は値の中から ${VAR} 形式の依存先を抽出します
func extractDependencies(val interface{}) []string {
	var deps []string
	reVar := regexp.MustCompile(`\$\{([^}]+)\}`) // env.go の正規表現と統一

	process := func(s string) {
		matches := reVar.FindAllStringSubmatch(s, -1)
		for _, m := range matches {
			if len(m) > 1 {
				deps = append(deps, m[1])
			}
		}
	}

	switch v := val.(type) {
	case string:
		process(v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				process(s)
			}
		}
	}
	return deps
}

// SortEnvsByDependency orders c.Envs so that variables referenced via
// ${} are defined before the variables that use them.
//
// caseInsensitive must reflect the target OS (see EnvKeysCaseInsensitive:
// true on Windows, false elsewhere). When true, dependency matching and
// de-duplication fold case ("Path" and "PATH" are the same variable);
// when false, keys must match exactly, mirroring case-sensitive Unix
// environments.
func (c *Config) SortEnvsByDependency(caseInsensitive bool) error {
	canon := func(key string) string {
		if caseInsensitive {
			return strings.ToUpper(key)
		}
		return key
	}
	equalKey := func(a, b string) bool {
		if caseInsensitive {
			return strings.EqualFold(a, b)
		}
		return a == b
	}

	resolved := make([]Environ, 0, len(c.Envs))
	seen := make(map[string]bool)

	var resolve func(e Environ, chain []string) error
	resolve = func(e Environ, chain []string) error {
		// 循環チェック: 現在の依存チェーンに自分が含まれていたら、
		// 経路全体（例: A -> B -> A）を添えてエラーにする。
		for _, link := range chain {
			if equalKey(link, e.Key) {
				full := append(append([]string{}, chain...), e.Key)
				return fmt.Errorf("circular reference detected: %s", strings.Join(full, " -> "))
			}
		}
		if seen[canon(e.Key)] {
			return nil
		}

		chain = append(chain, e.Key)

		// 値から依存している変数名を取得
		deps := extractDependencies(e.Value)
		for _, depKey := range deps {
			// 設定ファイル内の他の変数に依存しているか確認
			for _, other := range c.Envs {
				if equalKey(other.Key, depKey) {
					if err := resolve(other, chain); err != nil {
						return err
					}
				}
			}
		}

		seen[canon(e.Key)] = true
		resolved = append(resolved, e)
		return nil
	}

	for _, e := range c.Envs {
		if !seen[canon(e.Key)] {
			if err := resolve(e, nil); err != nil {
				return err
			}
		}
	}

	c.Envs = resolved
	return nil
}

// sortedKeys returns the keys of m in deterministic (sorted) order.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *Config) NormalizeEnvs(rt interfaces.Runtime) error {
	if c.RawEnvs == nil {
		return nil
	}

	switch data := c.RawEnvs.(type) {
	case []interface{}:
		// 【パターン1：リスト形式】
		// 1. [ {key: "K", value: "V"}, ... ] (従来形式)
		// 2. [ {TEST: "aiueo"}, ... ] (リスト内直接定義)
		for _, item := range data {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			// A. 従来形式のチェック: - key: "NAME", value: "VAL"
			if key, ok := m["key"].(string); ok && key != "" {
				val := m["value"]
				c.Envs = append(c.Envs, Environ{Key: key, Value: val})
				continue
			}

			// B. リスト内直接定義のチェック: - TEST: "aiueo"
			// 1要素に複数キーがあっても全部取り出す（以前は map を
			// 回して最初の1つで break し、残りをランダムに落としていた）。
			// 順序を決定的にするためキーをソートしてから追加する。
			for _, k := range sortedKeys(m) {
				if k != "" {
					c.Envs = append(c.Envs, Environ{Key: k, Value: m[k]})
				}
			}
		}

	case map[string]interface{}:
		// 【パターン2：マップ形式】 { "VARIABLE": "VALUE", "PATH": ["A", "B"] }
		// map のイテレーション順はランダムなため、キーをソートして
		// 決定的な順序にする（依存関係の最終順序は SortEnvsByDependency が決める）。
		for _, k := range sortedKeys(data) {
			c.Envs = append(c.Envs, Environ{Key: k, Value: data[k]})
		}

	case map[interface{}]interface{}:
		// YAMLパーサーの型互換性ケア
		strMap := make(map[string]interface{}, len(data))
		for k, v := range data {
			if strKey, ok := k.(string); ok {
				strMap[strKey] = v
			}
		}
		for _, k := range sortedKeys(strMap) {
			c.Envs = append(c.Envs, Environ{Key: k, Value: strMap[k]})
		}
	default:
		// 想定外の型（文字列や数値などが直接 envs に指定された場合）
		return fmt.Errorf("unsupported type for envs: %T (must be map or list)", c.RawEnvs)
	}
	return nil
}
func (c *Config) NormalizeConfigs(rt interfaces.Runtime) error {
	if c.RawConfigs == nil {
		return nil
	}

	switch data := c.RawConfigs.(type) {
	case []interface{}:
		// [[configs]] (リスト形式) の処理
		for _, item := range data {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			c.Configs = append(c.Configs, mapToMetaConfig(m))
		}
	case map[string]interface{}:
		// [configs] (マップ形式) の処理
		c.Configs = append(c.Configs, mapToMetaConfig(data))
	case map[interface{}]interface{}:
		// YAML用互換性
		m := make(map[string]interface{})
		for k, v := range data {
			if strKey, ok := k.(string); ok {
				m[strKey] = v
			}
		}
		c.Configs = append(c.Configs, mapToMetaConfig(m))
	}
	return nil
}

// 補助関数: map から MetaConfig 構造体に変換
func mapToMetaConfig(m map[string]interface{}) MetaConfig {
	var mc MetaConfig
	if s, ok := m["separator"].(string); ok {
		mc.Separator = s
	}
	if d, ok := m["description"].(string); ok {
		mc.Description = d
	}
	return mc
}

func ReadConfig(rt interfaces.Runtime, fileName string) (Config, error) {
	ext := general.FileExt(fileName)
	config := Config{}
	config.SourcePath = general.NormalizeSourcePath(fileName)
	if ext == ".toml" {
		return ReadToml(rt, fileName)
	} else if ext == ".yaml" || ext == ".yml" {
		return ReadYaml(rt, fileName)
	} else if ext == ".json" {
		return ReadJson(rt, fileName)
	} else if ext == ".env" {
		return ReadEnv(rt, fileName)
	}
	return config, fmt.Errorf("unsupported config file extension %q: %s", ext, fileName)
}

func ReadInlineConfig(rt interfaces.Runtime, content string, format string) (Config, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return Config{}, nil
	}

	var cfg Config
	var err error
	format = strings.ToLower(strings.TrimSpace(format))

	// --- 1. パース処理 (手動指定 or 自動判別) ---
	if format != "" {
		// フォーマットが明示されている場合
		switch format {
		case "toml":
			err = toml.Unmarshal([]byte(content), &cfg)
		case "yaml", "yml":
			err = yaml.Unmarshal([]byte(content), &cfg)
		case "json":
			err = json.Unmarshal([]byte(content), &cfg)
		default:
			return Config{}, fmt.Errorf("unsupported format: %s", format)
		}
	} else {
		// フォーマットが空の場合：自動判別
		// A. JSON: 先頭が { か [ なら JSON として試行
		if strings.HasPrefix(content, "{") || strings.HasPrefix(content, "[") {
			if err = json.Unmarshal([]byte(content), &cfg); err == nil {
				goto PARSED
			}
		}

		// B. TOML: 構造が厳格なので YAML より先に試す
		if err = toml.Unmarshal([]byte(content), &cfg); err == nil {
			// TOMLとして受理されても、中身が全くパースできていない(空の)場合は次へ
			if cfg.RawEnvs != nil || len(cfg.Configs) > 0 || cfg.Program.Path != "" {
				goto PARSED
			}
		}

		// C. YAML: 最も寛容なので最後に試す
		if err = yaml.Unmarshal([]byte(content), &cfg); err == nil {
			goto PARSED
		}

		return Config{}, fmt.Errorf("failed to auto-detect inline config format (tried JSON, TOML, YAML)")
	}

PARSED:
	if err != nil {
		return Config{}, fmt.Errorf("parse error (%s): %w", format, err)
	}

	// --- 2. 共通の正規化・依存関係ソート ---
	// 先ほど修正した、errorを返すようになった NormalizeEnvs を使用
	if err := cfg.NormalizeConfigs(rt); err != nil {
		return Config{}, fmt.Errorf("normalize configs error: %w", err)
	}
	if err := cfg.NormalizeEnvs(rt); err != nil {
		return Config{}, fmt.Errorf("normalize error: %w", err)
	}

	if err := cfg.SortEnvsByDependency(EnvKeysCaseInsensitive(rt.Env().GOOS())); err != nil {
		return Config{}, fmt.Errorf("dependency sort error: %w", err)
	}

	return cfg, nil
}

func ReadJson(rt interfaces.Runtime, fileName string) (Config, error) {
	data, err := rt.FS().ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	config.SourcePath = general.NormalizeSourcePath(fileName)
	err = json.Unmarshal(data, &config)
	if err == nil {
		if err := config.NormalizeConfigs(rt); err != nil {
			return Config{}, err
		}
		if err := config.NormalizeEnvs(rt); err != nil {
			return Config{}, err
		}
		if err := config.SortEnvsByDependency(EnvKeysCaseInsensitive(rt.Env().GOOS())); err != nil {
			return Config{}, err
		}
	}
	return config, err

}
func ReadYaml(rt interfaces.Runtime, fileName string) (Config, error) {
	data, err := rt.FS().ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	config.SourcePath = general.NormalizeSourcePath(fileName)
	err = yaml.Unmarshal(data, &config)
	if err == nil {
		if err := config.NormalizeConfigs(rt); err != nil {
			return Config{}, err
		}
		if err := config.NormalizeEnvs(rt); err != nil {
			return Config{}, err
		}
		if err := config.SortEnvsByDependency(EnvKeysCaseInsensitive(rt.Env().GOOS())); err != nil {
			return Config{}, err
		}
	}
	return config, err

}

func ReadToml(rt interfaces.Runtime, fileName string) (Config, error) {
	data, err := rt.FS().ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	config.SourcePath = general.NormalizeSourcePath(fileName)
	err = toml.Unmarshal(data, &config)

	if err == nil {
		if err := config.NormalizeConfigs(rt); err != nil {
			return Config{}, err
		}
		if err := config.NormalizeEnvs(rt); err != nil {
			return Config{}, err
		}
		if err := config.SortEnvsByDependency(EnvKeysCaseInsensitive(rt.Env().GOOS())); err != nil {
			return Config{}, err
		}
	}
	return config, err
}

func ReadEnv(rt interfaces.Runtime, fileName string) (Config, error) {
	var config Config
	var envMap map[string]string
	var err error
	config.SourcePath = general.NormalizeSourcePath(fileName)
	envMap, err = godotenv.Read(fileName)
	if err != nil {
		return Config{}, fmt.Errorf("error reading .env file %s: %w", fileName, err)
	}

	// map のイテレーション順に依存しないようキーをソートする
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	config.Envs = make([]Environ, 0, len(envMap))
	for _, key := range keys {
		config.Envs = append(config.Envs, Environ{
			Key:   key,
			Value: envMap[key],
		})
	}

	return config, nil
}

func ReadInlineToml(rt interfaces.Runtime, logger interfaces.Logger, tomlData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(rt.FS().TempDir(), "inline-"+uuid.NewString()+".toml")

	// 一時ファイルに書き込み
	err := rt.FS().WriteFile(tmpFileName, []byte(tomlData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer rt.FS().Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadToml(rt, tmpFileName)
}
func ReadInlineJson(rt interfaces.Runtime, jsonData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(rt.FS().TempDir(), "inline-"+uuid.NewString()+".json")

	// 一時ファイルに書き込み
	err := rt.FS().WriteFile(tmpFileName, []byte(jsonData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer rt.FS().Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadJson(rt, tmpFileName)

}
func ReadInlineYaml(rt interfaces.Runtime, yamlData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(rt.FS().TempDir(), "inline-"+uuid.NewString()+".yaml")

	// 一時ファイルに書き込み
	err := rt.FS().WriteFile(tmpFileName, []byte(yamlData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer rt.FS().Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadYaml(rt, tmpFileName)

}

func (c *Config) BuildEnvs(rt interfaces.Runtime, baseEnv []string, separator string) []string {

	// Description 出力とセパレータ決定ロジック（中略）
	for _, cfgs := range c.Configs {
		if cfgs.Description != "" {
			rt.Logger().Debug().Str("Config Description", cfgs.Description).Msg("")
		}
	}
	if separator == "" {
		for _, cfgs := range c.Configs {
			if cfgs.Separator != "" {
				separator = cfgs.Separator
				break
			}
		}
	}
	if separator == "" {
		if rt.Env().GOOS() == "windows" {
			separator = ";"
		} else {
			separator = ":"
		}
	}

	goos := rt.Env().GOOS()
	envMap := make(map[string][]string)
	// baseEnv (既存の環境変数) を map に変換。
	// キーは OS のケース規則で正規化する (Windows: 大文字化、それ以外: 原形)。
	for _, e := range baseEnv {
		if strings.HasPrefix(e, "=") {
			continue
		}
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key := CanonicalEnvKey(goos, parts[0])
			if key == "" {
				continue
			}
			envMap[key] = append(envMap[key], parts[1])
		}
	}

	// c.Envs (設定ファイルの envs) を処理
	for _, env := range c.Envs {
		if env.Key == "" {
			rt.Logger().Warn().Interface("env", env).Msg("envのキーが空です")
			continue
		}
		canonKey := CanonicalEnvKey(goos, env.Key)

		// 1. まず「生の値 (raw)」をスライスとして取り出す
		var rawStrings []string
		isListType := false
		rv := reflect.ValueOf(env.Value)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			isListType = true
			for i := 0; i < rv.Len(); i++ {
				// 中身を文字列に変換して追加
				rawStrings = append(rawStrings, fmt.Sprint(rv.Index(i).Interface()))
			}
		} else if s, ok := env.Value.(string); ok {
			rawStrings = []string{s}
			isListType = false
		} else {
			continue
		}
		rt.Logger().Debug().
			Str("key", canonKey).
			Bool("isListType", isListType).
			Interface("rawValues", rawStrings).
			Msg("Processing environment variable")
		// 2. 重要：ここまでの envMap（上の行の変数が反映済み）を使って展開する
		var expandedValues []string
		for _, raw := range rawStrings {
			// この時点の envMap には直前のループの結果が入っている
			expanded := general.ExpandEnvAndCommands(rt, raw, envMap)
			expandedValues = append(expandedValues, expanded)
		}

		// 3. 既存値と展開後の新規値をマージして重複排除
		existing := make(map[string]struct{})
		merged := []string{}
		if isListType {

			// a) 既存の値を登録
			for _, v := range envMap[canonKey] {
				for _, part := range strings.Split(v, separator) {
					part = strings.TrimSpace(part)
					if part != "" {
						if _, ok := existing[part]; !ok {
							existing[part] = struct{}{}
							merged = append(merged, part)
						}
					}
				}
			}

			// b) 展開された新しい値を登録
			for _, v := range expandedValues {
				for _, part := range strings.Split(v, separator) {
					part = strings.TrimSpace(part)
					if part != "" {
						if _, ok := existing[part]; !ok {
							existing[part] = struct{}{}
							merged = append(merged, part)
						}
					}
				}
			}
		} else {
			merged = expandedValues
		}

		// 4. 重要：envMap を即座に更新 (これで次のループの展開で参照可能になる)
		envMap[canonKey] = merged
	}

	// map を []string に変換して返す
	newEnv := make([]string, 0, len(envMap))
	for k, v := range envMap {
		joined := strings.Join(v, separator)
		newEnv = append(newEnv, k+"="+joined)
	}

	return newEnv
}
