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
	// RawCapture holds the [capture] section: either a table with a
	// "commands" list or a plain list of command strings. The commands
	// are generic shell commands whose resulting environment is merged
	// into the final env (see NormalizeCapture and domain capture).
	RawCapture      interface{} `toml:"capture" yaml:"capture" json:"capture"`
	CaptureCommands []string    `toml:"-" yaml:"-" json:"-"`
}
type MetaConfig struct {
	Separator   string `toml:"separator" yaml:"separator" json:"separator"`
	Description string `toml:"description" yaml:"description" json:"description"`
	// ExpandVars / ExecCommands are tri-state toggles for ${} expansion
	// and $() command substitution: nil means "not specified here"
	// (inherit / default true), a non-nil value sets the behavior.
	// When several configs are merged, the last non-nil value wins.
	ExpandVars   *bool `toml:"expand_vars" yaml:"expand_vars" json:"expand_vars"`
	ExecCommands *bool `toml:"exec_commands" yaml:"exec_commands" json:"exec_commands"`
}
type ProgramData struct {
	Path string   `toml:"path" yaml:"path" json:"path"`
	Args []string `toml:"args" yaml:"args" json:"args"`
}
type Environ struct {
	Key   string      `toml:"key" yaml:"key" json:"key"`
	Value interface{} `toml:"value" yaml:"value" json:"value"`
	// Prepend is the merge mode for list values: false (default) keeps
	// the historical append behavior (existing parts first, new parts
	// appended); true — set for entries from a [envs.prepend] section —
	// puts the new parts BEFORE the existing value. A scalar entry
	// with Prepend=true is treated as a one-element prepend list.
	Prepend bool `toml:"-" yaml:"-" json:"-"`
	// Section records which [envs] sub-section defined the entry:
	// "" for a plain [envs] entry, "append" for [envs.append] and
	// "prepend" for [envs.prepend]. Used for display (tree badges).
	Section string `toml:"-" yaml:"-" json:"-"`
	// Origin records the OS section the entry came from: the canonical
	// GOOS name ("windows" / "linux" / "darwin") for entries defined
	// under [envs.<os>], "" for common entries. Used for display
	// (tree badges).
	Origin string `toml:"-" yaml:"-" json:"-"`
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
// true on Windows, false elsewhere). When true, dependency matching
// folds case ("Path" and "PATH" are the same variable); when false,
// keys must match exactly, mirroring case-sensitive Unix environments.
//
// Duplicate keys are preserved: a config may define the same variable
// several times (a plain [envs] entry plus [envs.append] /
// [envs.prepend] entries, or a common entry plus an OS-section entry),
// and every definition must reach BuildEnvs in order. (An earlier
// version of this sort silently dropped all but the first definition
// of a key.) A ${} reference to a duplicated key resolves against the
// OTHER entries with that key; a variable whose only definition
// references itself is still reported as a circular reference, as are
// genuine cycles (A -> B -> A), with the chain included in the error.
func (c *Config) SortEnvsByDependency(caseInsensitive bool) error {
	equalKey := func(a, b string) bool {
		if caseInsensitive {
			return strings.EqualFold(a, b)
		}
		return a == b
	}

	n := len(c.Envs)
	resolved := make([]Environ, 0, n)
	emitted := make([]bool, n)
	inStack := make([]bool, n)

	var resolve func(i int, chain []string) error
	resolve = func(i int, chain []string) error {
		if emitted[i] {
			return nil
		}
		if inStack[i] {
			// 循環: 現在の解決スタック上に同じエントリがいる。
			// 経路全体（例: A -> B -> A）を添えてエラーにする。
			full := append(append([]string{}, chain...), c.Envs[i].Key)
			return fmt.Errorf("circular reference detected: %s", strings.Join(full, " -> "))
		}
		inStack[i] = true
		chain = append(chain, c.Envs[i].Key)

		// 値から依存している変数名を取得し、依存先を先に解決する
		deps := extractDependencies(c.Envs[i].Value)
		for _, depKey := range deps {
			provided := false
			for j := range c.Envs {
				if j == i {
					continue
				}
				if equalKey(c.Envs[j].Key, depKey) {
					provided = true
					if err := resolve(j, chain); err != nil {
						return err
					}
				}
			}
			if !provided && equalKey(c.Envs[i].Key, depKey) {
				// 自分以外に提供者がいない自己参照は循環として報告する
				full := append(append([]string{}, chain...), c.Envs[i].Key)
				return fmt.Errorf("circular reference detected: %s", strings.Join(full, " -> "))
			}
		}

		inStack[i] = false
		emitted[i] = true
		resolved = append(resolved, c.Envs[i])
		return nil
	}

	for i := range c.Envs {
		if err := resolve(i, nil); err != nil {
			return err
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

// canonicalOSName maps an [envs.<name>] sub-table name to a canonical
// GOOS name, resolving aliases ("win" -> "windows", "mac"/"macos" ->
// "darwin"). ok=false means the name is not a reserved OS name.
func canonicalOSName(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "windows", "win":
		return "windows", true
	case "linux":
		return "linux", true
	case "darwin", "mac", "macos":
		return "darwin", true
	}
	return "", false
}

// toStringEnvMap converts a nested env table (an OS bucket value) into
// map[string]interface{}; ok=false when v is not a map at all.
func toStringEnvMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	}
	return nil, false
}

func (c *Config) NormalizeEnvs(rt interfaces.Runtime) error {
	if c.RawEnvs == nil {
		return nil
	}

	// セクション振り分けの受け皿。
	//
	// [envs] 直下には2種類の予約サブテーブルがある (どちらも「予約名 +
	// map 値」のときだけセクションとして扱い、それ以外は通常の変数):
	//   - OS セクション ([envs.windows] 等): 現在の OS のものだけを
	//     共通エントリの後ろに付け足す。SortEnvsByDependency の前に
	//     混ざるので、共通 <-> OS 別の相互参照が ${} で解決できる。
	//   - マージセクション ([envs.prepend] / [envs.append]): リスト値の
	//     マージ方向を指定する。OS セクションの中でも使える
	//     ([envs.windows.prepend] 等)。
	//
	// 1スコープ (共通 or 1つの OS) の中では plain -> append -> prepend
	// の順に flush する。スカラーの置換が先に起き、append が後ろに、
	// prepend が前に来るので、3つを同じキーに併用しても素直な
	// 「prepend ++ 本体 ++ append」の並びになる。
	currentOS, _ := canonicalOSName(rt.Env().GOOS())
	common := &envSectionSet{}
	osSets := make(map[string]*envSectionSet)
	osSet := func(osName string) *envSectionSet {
		set, ok := osSets[osName]
		if !ok {
			set = &envSectionSet{}
			osSets[osName] = set
		}
		return set
	}

	// addToSet は1スコープ分のエントリを振り分ける。prepend / append の
	// 予約名 + map 値はサブセクションとして展開し、それ以外は plain。
	addToSet := func(set *envSectionSet, origin, key string, val interface{}) {
		if section, reserved := mergeSectionName(key); reserved {
			if vars, isMap := toStringEnvMap(val); isMap {
				for _, k := range sortedKeys(vars) {
					e := Environ{Key: k, Value: vars[k], Origin: origin, Section: section, Prepend: section == "prepend"}
					if section == "prepend" {
						set.prepends = append(set.prepends, e)
					} else {
						set.appends = append(set.appends, e)
					}
				}
				return
			}
		}
		set.plain = append(set.plain, Environ{Key: key, Value: val, Origin: origin})
	}

	addVarOrBucket := func(key string, val interface{}) {
		if key == "" {
			return
		}
		if osName, reserved := canonicalOSName(key); reserved {
			if vars, isMap := toStringEnvMap(val); isMap {
				set := osSet(osName)
				for _, k := range sortedKeys(vars) {
					addToSet(set, osName, k, vars[k])
				}
				return
			}
		}
		addToSet(common, "", key, val)
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
				addVarOrBucket(key, m["value"])
				continue
			}

			// B. リスト内直接定義のチェック: - TEST: "aiueo"
			// 1要素に複数キーがあっても全部取り出す（以前は map を
			// 回して最初の1つで break し、残りをランダムに落としていた）。
			// 順序を決定的にするためキーをソートしてから追加する。
			for _, k := range sortedKeys(m) {
				addVarOrBucket(k, m[k])
			}
		}

	case map[string]interface{}:
		// 【パターン2：マップ形式】 { "VARIABLE": "VALUE", "PATH": ["A", "B"] }
		// map のイテレーション順はランダムなため、キーをソートして
		// 決定的な順序にする（依存関係の最終順序は SortEnvsByDependency が決める）。
		for _, k := range sortedKeys(data) {
			addVarOrBucket(k, data[k])
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
			addVarOrBucket(k, strMap[k])
		}
	default:
		// 想定外の型（文字列や数値などが直接 envs に指定された場合）
		return fmt.Errorf("unsupported type for envs: %T (must be map or list)", c.RawEnvs)
	}

	// 共通スコープを flush し、現在の OS のスコープを後ろに追加する
	// (後勝ちの通常処理で共通値を上書き・マージできる)。
	flush := func(set *envSectionSet) {
		c.Envs = append(c.Envs, set.plain...)
		c.Envs = append(c.Envs, set.appends...)
		c.Envs = append(c.Envs, set.prepends...)
	}
	flush(common)
	if set, ok := osSets[currentOS]; ok {
		flush(set)
	}
	return nil
}

// envSectionSet accumulates the entries of one scope (the common
// [envs] table or one [envs.<os>] bucket), split by merge section so
// NormalizeEnvs can flush them in plain -> append -> prepend order.
type envSectionSet struct {
	plain    []Environ
	appends  []Environ
	prepends []Environ
}

// mergeSectionName maps an [envs] sub-table name to its merge section
// ("prepend" / "append"). ok=false means the name is not a reserved
// merge-section name. Callers must only treat it as a section when
// the value is a map; a variable literally named e.g. "prepend" with
// a scalar or list value stays a regular variable.
func mergeSectionName(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "prepend", "append":
		return strings.ToLower(name), true
	}
	return "", false
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

// NormalizeCapture flattens the [capture] section into
// c.CaptureCommands. Accepted forms:
//
//	[capture]
//	commands = ["setup-env.bat", "scripts/more.bat"]
//
//	capture = ["setup-env.sh"]   (plain list form)
//
//	capture = "setup-env.sh"     (single command)
func (c *Config) NormalizeCapture() error {
	if c.RawCapture == nil {
		return nil
	}
	add := func(v interface{}) {
		if s, ok := v.(string); ok && s != "" {
			c.CaptureCommands = append(c.CaptureCommands, s)
		}
	}
	addList := func(list []interface{}) {
		for _, item := range list {
			add(item)
		}
	}

	switch data := c.RawCapture.(type) {
	case map[string]interface{}:
		if list, ok := data["commands"].([]interface{}); ok {
			addList(list)
		} else if s, ok := data["commands"].(string); ok {
			add(s)
		}
	case map[interface{}]interface{}:
		// YAML 用互換性
		m := make(map[string]interface{}, len(data))
		for k, v := range data {
			if strKey, ok := k.(string); ok {
				m[strKey] = v
			}
		}
		if list, ok := m["commands"].([]interface{}); ok {
			addList(list)
		} else if s, ok := m["commands"].(string); ok {
			add(s)
		}
	case []interface{}:
		addList(data)
	case []string:
		for _, s := range data {
			add(s)
		}
	case string:
		add(data)
	default:
		return fmt.Errorf("unsupported type for capture: %T (must be a list of commands or a table with a commands list)", c.RawCapture)
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
	if b, ok := m["expand_vars"].(bool); ok {
		v := b
		mc.ExpandVars = &v
	}
	if b, ok := m["exec_commands"].(bool); ok {
		v := b
		mc.ExecCommands = &v
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
			if cfg.RawEnvs != nil || len(cfg.Configs) > 0 || cfg.Program.Path != "" || cfg.RawCapture != nil {
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
	if err := cfg.NormalizeCapture(); err != nil {
		return Config{}, fmt.Errorf("normalize capture error: %w", err)
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
		if err := config.NormalizeCapture(); err != nil {
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
		if err := config.NormalizeCapture(); err != nil {
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
		if err := config.NormalizeCapture(); err != nil {
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

// BuildEnvs resolves the config's envs on top of baseEnv. opts
// controls ${} expansion and $() execution (see
// general.ExpandOptions; use general.DefaultExpandOptions for the
// historical behavior).
func (c *Config) BuildEnvs(rt interfaces.Runtime, baseEnv []string, separator string, opts general.ExpandOptions) []string {

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

	// 組み込み変数 (Phase 4): 展開専用・最低優先度で envMap に seed する。
	//   CONFIG_DIR: この設定ファイルのあるディレクトリ。SourcePath が
	//     空 (inline 設定など) のときは定義しない。
	//   EEC_HOME:   解決済みデータディレクトリ (ResolveDataDir)。
	// baseEnv に同名キーがある場合はそちらが勝つ。設定ファイル自身が
	// 同名キーを定義した場合は builtin 扱いをやめ、通常の変数として
	// 出力する (下のループで hidden から外す)。
	hiddenBuiltins := make(map[string]bool)
	builtins := make(map[string]string)
	if c.SourcePath != "" {
		builtins["CONFIG_DIR"] = filepath.Dir(c.SourcePath)
	}
	if dataDir := ResolveDataDir(rt); dataDir != "" {
		builtins["EEC_HOME"] = dataDir
	}
	for name, val := range builtins {
		canon := CanonicalEnvKey(goos, name)
		if _, exists := envMap[canon]; !exists {
			envMap[canon] = []string{val}
			hiddenBuiltins[canon] = true
		}
	}

	// c.Envs (設定ファイルの envs) を処理
	for _, env := range c.Envs {
		if env.Key == "" {
			rt.Logger().Warn().Interface("env", env).Msg("envのキーが空です")
			continue
		}
		canonKey := CanonicalEnvKey(goos, env.Key)
		// 設定自身が組み込み変数と同名キーを定義したら、そちらが
		// builtin を上書きし、通常の変数として出力対象になる。
		delete(hiddenBuiltins, canonKey)

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
			expanded := general.ExpandEnvAndCommandsWithOptions(rt, raw, envMap, opts)
			expandedValues = append(expandedValues, expanded)
		}

		// 3. 既存値と展開後の新規値をマージして重複排除
		merged := []string{}
		if env.Prepend {
			// prepend ([envs.prepend] 由来): 展開後の新しい値を前に、
			// 既存値を後ろに並べる。重複は最初に現れた位置が勝つので、
			// 既存値に含まれていた要素を prepend すると先頭へ移動する。
			// スカラー値でも1要素の prepend リストとして扱う。
			seen := make(map[string]struct{})
			addParts := func(vals []string) {
				for _, v := range vals {
					for _, part := range strings.Split(v, separator) {
						part = strings.TrimSpace(part)
						if part == "" {
							continue
						}
						if _, ok := seen[part]; !ok {
							seen[part] = struct{}{}
							merged = append(merged, part)
						}
					}
				}
			}
			addParts(expandedValues)
			addParts(envMap[canonKey])
		} else if isListType {
			existing := make(map[string]struct{})

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

	// map を []string に変換して返す。
	// 展開専用の組み込み変数は、baseEnv / 設定のどちらでも実定義
	// されなかった限り最終環境には混ぜない。
	newEnv := make([]string, 0, len(envMap))
	for k, v := range envMap {
		if hiddenBuiltins[k] {
			continue
		}
		joined := strings.Join(v, separator)
		newEnv = append(newEnv, k+"="+joined)
	}

	return newEnv
}
