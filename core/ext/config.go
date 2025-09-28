package ext

import (
	"github.com/google/uuid"
	"github.com/m0090-dev/eec-go/core/utils/general"
	"github.com/pelletier/go-toml/v2"
	"encoding/json"
	"gopkg.in/yaml.v3"
	//"github.com/rs/zerolog/log"
	// "os"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Envs    []Environ   `toml:"envs" yaml:"envs" json:"envs"`
	Program ProgramData `toml:"program" taml:"program" json:"program"`
}
type ProgramData struct {
	Path string   `toml:"path" yaml:"path" json:"path"`
	Args []string `toml:"args" yaml:"args" json:"args"`
}
type Environ struct {
	Key   string      `toml:"key" yaml:"key" json:"key"`
	Value interface{} `toml:"value" yaml:"value" json:"value"`
}

func ReadConfig(os OS,logger Logger,fileName string) (Config, error) {
	ext := general.FileExt(fileName)
	if ext == ".toml" {
		return ReadToml(os,logger,fileName)
	} else if ext == ".yaml" || ext == ".yml" {
		return ReadYaml(os,logger,fileName)
	} else if ext == ".json" {
		return ReadJson(os,logger,fileName)
	}
	return Config{}, nil
}
func ReadInlineConfig(os OS,logger Logger,fileName string) (Config, error) {
	ext := os.FS.FileExt(fileName)
	if ext == ".toml" {
		return ReadInlineToml(os,logger,fileName)
	} else if ext == ".yaml" || ext == ".yml" {
		return ReadInlineYaml(os,logger,fileName)
	} else if ext == ".json" {
		return ReadInlineJson(os,logger,fileName)
	}
	return Config{}, nil
}

func ReadJson(os OS,logger Logger,fileName string) (Config, error) {
	data, err := os.FS.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return config, err

}
func ReadYaml(os OS,logger Logger,fileName string) (Config, error) {
	data, err := os.FS.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	return config, err

}

func ReadToml(os OS,logger Logger,fileName string) (Config, error) {
	data, err := os.FS.ReadFile(fileName)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = toml.Unmarshal(data, &config)
	return config, err
}

func ReadInlineToml(os OS,logger Logger,tomlData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(os.FS.TempDir(), "inline-"+uuid.NewString()+".toml")

	// 一時ファイルに書き込み
	err := os.FS.WriteFile(tmpFileName, []byte(tomlData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer os.FS.Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadToml(os,logger,tmpFileName)
}
func ReadInlineJson(os OS,logger Logger,jsonData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(os.FS.TempDir(), "inline-"+uuid.NewString()+".json")

	// 一時ファイルに書き込み
	err := os.FS.WriteFile(tmpFileName, []byte(jsonData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer os.FS.Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadJson(os,logger,tmpFileName)

}
func ReadInlineYaml(os OS,logger Logger,yamlData string) (Config, error) {
	// UUID を使って一時ファイル名を生成
	tmpFileName := filepath.Join(os.FS.TempDir(), "inline-"+uuid.NewString()+".yaml")

	// 一時ファイルに書き込み
	err := os.FS.WriteFile(tmpFileName, []byte(yamlData), 0600)
	if err != nil {
		return Config{}, err
	}

	// defer で削除を確実に実行
	defer os.FS.Remove(tmpFileName)

	// 通常の読み込み処理を使う
	return ReadYaml(os,logger,tmpFileName)

}

/*func (c *Config) ApplyEnvs() error {*/

/*// -- OSごとの区切り文字分岐 --*/
/*var separator string*/
/*if runtime.GOOS == "windows" {*/
/*separator = ";" // Windowsではセミコロン*/
/*} else {*/
/*separator = ":" // Unix/Linux/macOSではコロン*/
/*}*/

/*currentPaths := strings.Split(os.Getenv("PATH"), separator)*/
/*for _, env := range c.Envs {*/
/*key := env.Key*/
/*if key == "" {*/
/*log.Warn().Interface("env", env).Msg("envのキーが空です")*/
/*continue*/
/*}*/

/*switch val := env.Value.(type) {*/
/*case string:*/
/*// スカラー文字列*/
/*expanded := utils.ExpandEnvVariables(val)*/
/*os.Setenv(key, expanded)*/

/*case []interface{}:*/
/*// 文字列の配列（interface{}スライス）*/
/*strVals := make([]string, 0, len(val))*/
/*for _, v := range val {*/
/*if s, ok := v.(string); ok {*/
/*strVals = append(strVals, utils.ExpandEnvVariables(s))*/
/*} else {*/
/*log.Warn().*/
/*Str("key", key).*/
/*Interface("element", v).*/
/*Msg("env配列の要素が文字列でない")*/
/*}*/
/*}*/

/*if strings.EqualFold(key, "Path") {*/
/*configPaths := strings.Join(strVals, separator)*/
/*newPaths := append(currentPaths, configPaths)*/
/*os.Setenv(key, strings.Join(newPaths, separator))*/
/*} else {*/

/*os.Setenv(key, strings.Join(strVals, separator))*/
/*}*/
/*default:*/
/*log.Warn().*/
/*Str("key", key).*/
/*Interface("value", env.Value).*/
/*Msg("envの値の型が未対応")*/
/*}*/
/*}*/

/*return nil*/
/*}*/

/*func (c *Config) BuildEnvs(baseEnv []string) []string {*/
/*envMap := make(map[string]string)*/

/*// baseEnv を map に変換*/
/*for _, e := range baseEnv {*/
/*parts := strings.SplitN(e, "=", 2)*/
/*if len(parts) == 2 {*/
/*envMap[strings.ToUpper(parts[0])] = parts[1]*/
/*}*/
/*}*/

/*separator := ";"*/
/*if runtime.GOOS != "windows" {*/
/*separator = ":"*/
/*}*/

/*for _, env := range c.Envs {*/
/*key := env.Key*/
/*if key == "" {*/
/*log.Warn().Interface("env", env).Msg("envのキーが空です")*/
/*continue*/
/*}*/

/*keyUpper := strings.ToUpper(key)*/

/*switch val := env.Value.(type) {*/
/*case string:*/
/*envMap[keyUpper] = utils.ExpandEnvVariables(val)*/

/*case []interface{}:*/
/*strVals := make([]string, 0, len(val))*/
/*for _, v := range val {*/
/*if s, ok := v.(string); ok {*/
/*strVals = append(strVals, utils.ExpandEnvVariables(s))*/
/*}*/
/*}*/

/*if keyUpper == "PATH" {*/
/*basePaths := []string{}*/
/*if existing, ok := envMap["PATH"]; ok {*/
/*basePaths = strings.Split(existing, separator)*/
/*}*/
/*// merged := append(basePaths, strVals...)*/
/*merged := append(strVals, basePaths...)*/
/*envMap["PATH"] = strings.Join(merged, separator)*/
/*} else {*/
/*envMap[keyUpper] = strings.Join(strVals, separator)*/
/*}*/
/*}*/
/*}*/

/*// map を []string に戻す*/
/*newEnv := make([]string, 0, len(envMap))*/
/*for k, v := range envMap {*/
/*newEnv = append(newEnv, k+"="+v)*/
/*}*/

/*return newEnv*/
/*}*/

/*func (c *Config) ApplyEnvs() error {*/
	/*// OSごとの区切り文字*/
	/*separator := ";"*/
	/*if runtime.GOOS != "windows" {*/
		/*separator = ":"*/
	/*}*/

	/*// 現在の環境を map に変換して重複チェック用*/
	/*envMap := make(map[string]map[string]struct{})*/
	/*for _, e := range os.Environ() {*/
		/*parts := strings.SplitN(e, "=", 2)*/
		/*if len(parts) != 2 {*/
			/*continue*/
		/*}*/
		/*key := strings.ToUpper(parts[0])*/
		/*val := parts[1]*/
		/*if _, ok := envMap[key]; !ok {*/
			/*envMap[key] = make(map[string]struct{})*/
		/*}*/
		/*envMap[key][val] = struct{}{}*/
	/*}*/

	/*for _, env := range c.Envs {*/
		/*key := strings.ToUpper(env.Key)*/
		/*if key == "" {*/
			/*log.Warn().Interface("env", env).Msg("envのキーが空です")*/
			/*continue*/
		/*}*/

		/*// string でも []interface{} でも統一して処理*/
		/*var strVals []string*/
		/*switch val := env.Value.(type) {*/
		/*case string:*/
			/*strVals = []string{general.ExpandEnvVariables(val)}*/
		/*case []interface{}:*/
			/*for _, v := range val {*/
				/*if s, ok := v.(string); ok {*/
					/*strVals = append(strVals, general.ExpandEnvVariables(s))*/
				/*}*/
			/*}*/
		/*default:*/
			/*log.Warn().Str("key", key).Interface("value", env.Value).Msg("envの値の型が未対応")*/
			/*continue*/
		/*}*/

		/*// 重複チェック用 map 初期化*/
		/*if _, ok := envMap[key]; !ok {*/
			/*envMap[key] = make(map[string]struct{})*/
		/*}*/

		/*// 重複しない値だけ追加*/
		/*for _, v := range strVals {*/
			/*if _, exists := envMap[key][v]; !exists {*/
				/*envMap[key][v] = struct{}{}*/
			/*}*/
		/*}*/

		/*// マップから文字列スライスに変換してセット*/
		/*newVals := make([]string, 0, len(envMap[key]))*/
		/*for val := range envMap[key] {*/
			/*newVals = append(newVals, val)*/
		/*}*/
		/*os.Setenv(key, strings.Join(newVals, separator))*/
	/*}*/

	/*return nil*/
/*}*/


func (c *Config) ApplyEnvs(os OS,logger Logger) error {
	// OSごとの区切り文字
	separator := ";"
	if runtime.GOOS != "windows" {
		separator = ":"
	}

	// 現在の環境を map に変換して重複チェック用
	// key: ENV 名（大文字化）
	// value: 値ごとの set
	envMap := make(map[string]map[string]struct{})
	for _, e := range os.Env.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		if _, ok := envMap[key]; !ok {
			envMap[key] = make(map[string]struct{})
		}

		// 値を separator で分割して格納
		for _, part := range strings.Split(parts[1], separator) {
			part = strings.TrimSpace(part)
			if part != "" {
				envMap[key][part] = struct{}{}
			}
		}
	}

	for _, env := range c.Envs {
		key := strings.ToUpper(env.Key)
		if key == "" {
			logger.Warn().Interface("env", env).Msg("envのキーが空です")
			continue
		}

		// string でも []interface{} でも統一して処理
		var strVals []string
		switch val := env.Value.(type) {
		case string:
			strVals = []string{general.ExpandEnvVariables(val)}
		case []interface{}:
			for _, v := range val {
				if s, ok := v.(string); ok {
					strVals = append(strVals, general.ExpandEnvVariables(s))
				}
			}
		default:
			logger.Warn().Str("key", key).Interface("value", env.Value).Msg("envの値の型が未対応")
			continue
		}

		// 重複チェック用 map 初期化
		if _, ok := envMap[key]; !ok {
			envMap[key] = make(map[string]struct{})
		}

		// 値を separator で分割して追加
		for _, v := range strVals {
			for _, part := range strings.Split(v, separator) {
				part = strings.TrimSpace(part)
				if part != "" {
					envMap[key][part] = struct{}{}
				}
			}
		}

		// マップから文字列スライスに変換してセット
		newVals := make([]string, 0, len(envMap[key]))
		for val := range envMap[key] {
			newVals = append(newVals, val)
		}
		os.Env.Setenv(key, strings.Join(newVals, separator))
	}

	return nil
}


func (c *Config) BuildEnvs(os OS,logger Logger,baseEnv []string) []string {
	envMap := make(map[string][]string)
	separator := ";"
	if runtime.GOOS != "windows" {
		separator = ":"
	}

	// baseEnv を map に変換（複数同キーをスライスに格納）
	for _, e := range baseEnv {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			key := strings.ToUpper(parts[0])
			envMap[key] = append(envMap[key], parts[1])
		}
	}

	// c.Envs をマージ（重複排除）
	for _, env := range c.Envs {
		if env.Key == "" {
			logger.Warn().Interface("env", env).Msg("envのキーが空です")
			continue
		}
		keyUpper := strings.ToUpper(env.Key)

		var values []string
		switch val := env.Value.(type) {
		case string:
			values = []string{general.ExpandEnvVariables(val)}
		case []interface{}:
			for _, v := range val {
				if s, ok := v.(string); ok {
					values = append(values, general.ExpandEnvVariables(s))
				}
			}
		default:
			logger.Warn().Interface("env", env).Msg("無効な値タイプ")
			continue
		}

		/*// 既存値に追加（重複チェック）*/
		/*existing := make(map[string]struct{})*/
		/*for _, v := range envMap[keyUpper] {*/
		/*existing[v] = struct{}{}*/
		/*}*/

		/*for _, v := range values {*/
		/*if _, ok := existing[v]; !ok {*/
		/*envMap[keyUpper] = append(envMap[keyUpper], v)*/
		/*existing[v] = struct{}{}*/
		/*}*/
		/*}*/

		// 既存値と新規値をマージして重複排除
		existing := make(map[string]struct{})
		merged := []string{}

		// 既存値
		for _, v := range envMap[keyUpper] {
			for _, part := range strings.Split(v, separator) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if _, ok := existing[part]; !ok {
					existing[part] = struct{}{}
					merged = append(merged, part)
				}
			}
		}

		// 新しい値
		for _, v := range values {
			for _, part := range strings.Split(v, separator) {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if _, ok := existing[part]; !ok {
					existing[part] = struct{}{}
					merged = append(merged, part)
				}
			}
		}

		envMap[keyUpper] = merged

	}

	// map を []string に変換
	newEnv := make([]string, 0, len(envMap))
	for k, v := range envMap {
		joined := strings.Join(v, separator)
		newEnv = append(newEnv, k+"="+joined)
	}

	return newEnv
}
