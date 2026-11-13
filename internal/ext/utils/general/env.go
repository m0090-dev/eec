package general

import (
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"regexp"
	"strings"
)

func findMatchingParen(runes []rune, start int) (content string, nextIdx int) {
	stack := 1
	for i := start; i < len(runes); i++ {
		if runes[i] == '(' {
			stack++
		} else if runes[i] == ')' {
			stack--
			if stack == 0 {
				// 対応する閉じ括弧が見つかった
				return string(runes[start:i]), i
			}
		}
	}
	return "", -1
}

// canonicalEnvKey mirrors types.CanonicalEnvKey: keys are upper-cased
// only on Windows, where environment variables are case-insensitive.
// This package cannot import types (types imports general), so the rule
// is duplicated here in miniature — keep the two in sync.
func canonicalEnvKey(goos, key string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

// 型がバラバラな envMap から値を取り出すヘルパー。
// envMap のキーは BuildEnvs が OS のケース規則で正規化済みなので、
// 参照キーも同じ規則 (goos) で正規化してから引く。
func getValuesFromAny(goos string, envMap any, key string) []string {
	key = canonicalEnvKey(goos, key)
	//fmt.Printf("DEBUG: Looking for [%s]\n", key)
	switch m := envMap.(type) {
	case map[string]map[string]struct{}:
		if vMap, ok := m[key]; ok {
			var vals []string
			for v := range vMap {
				vals = append(vals, v)
			}
			return vals
		}
	case map[string][]string:
		return m[key]
	}
	return nil
}

// すべての envMap を OS の環境変数形式 (KEY=VALUE) に変換するヘルパー
func getAllEnvsFromAny(rt interfaces.Runtime, envMap any) []string {
	sep := string(rt.Env().PathListSeparator())
	var result []string
	switch m := envMap.(type) {
	case map[string]map[string]struct{}:
		for k, vMap := range m {
			if strings.HasPrefix(k, "=") || k == "" {
				continue
			}
			var vals []string
			for v := range vMap {
				vals = append(vals, v)
			}
			result = append(result, k+"="+strings.Join(vals, sep))
		}
	case map[string][]string:
		for k, v := range m {
			if strings.HasPrefix(k, "=") || k == "" {
				continue
			}
			result = append(result, k+"="+strings.Join(v, sep))
		}
	}
	return result
}

func replaceVariables(rt interfaces.Runtime, input string, envMap any) string {
	sep := rt.Env().PathListSeparator()
	reVar := regexp.MustCompile(`\$\{([^}]+)\}`)

	return reVar.ReplaceAllStringFunc(input, func(match string) string {
		submatches := reVar.FindStringSubmatch(match)
		if len(submatches) == 2 {
			key := submatches[1]
			// 1. envMap (JSONなど) から値を探す
			vals := getValuesFromAny(rt.Env().GOOS(), envMap, key)
			if len(vals) > 0 {
				return strings.Join(vals, sep)
			}
			// 2. なければ OS 環境変数から探す。
			// Windows では OS 側の検索がケース不感だが、モック等の
			// 実装に備えて正規化したキーでも試す。
			if val, ok := rt.Env().LookupEnv(key); ok {
				return val
			}
			if canon := canonicalEnvKey(rt.Env().GOOS(), key); canon != key {
				if val, ok := rt.Env().LookupEnv(canon); ok {
					return val
				}
			}
		}
		return match
	})
}

// ExpandOptions controls which substitutions ExpandEnvAndCommands
// performs. The zero value disables everything; use
// DefaultExpandOptions for the historical behavior.
type ExpandOptions struct {
	// ExpandVars enables ${VAR} variable expansion. When false, the
	// input is returned completely unexpanded.
	ExpandVars bool
	// ExecCommands enables $(cmd) command substitution. When false
	// (and ExpandVars is true), ${} still expands but $() is left
	// literal instead of being executed.
	ExecCommands bool
}

// DefaultExpandOptions returns the historical behavior: both ${}
// expansion and $() execution enabled.
func DefaultExpandOptions() ExpandOptions {
	return ExpandOptions{ExpandVars: true, ExecCommands: true}
}

func ExpandEnvAndCommands(rt interfaces.Runtime, input string, envMap any) string {
	return ExpandEnvAndCommandsWithOptions(rt, input, envMap, DefaultExpandOptions())
}

func ExpandEnvAndCommandsWithOptions(rt interfaces.Runtime, input string, envMap any, opts ExpandOptions) string {
	if !opts.ExpandVars {
		return input
	}
	result := replaceVariables(rt, input, envMap)
	// --------------------------------------------------
	// 1. 環境変数の展開: ${VAR} (Regex で OK)
	// --------------------------------------------------

	// --------------------------------------------------
	// 2. コマンド展開: $(cmd) (スタック解析でネスト対応)
	// --------------------------------------------------
	runes := []rune(result)
	var sb strings.Builder

	for i := 0; i < len(runes); i++ {
		// "$(" の開始を検知
		if i+1 < len(runes) && runes[i] == '$' && runes[i+1] == '(' {
			start := i + 2
			content, nextIdx := findMatchingParen(runes, start)

			if nextIdx != -1 {
				// 抽出したコマンドラインを Executor 経由で実行
				cmdLine := strings.TrimSpace(content)
				if !opts.ExecCommands {
					// 実行せず、$(...) をリテラルのまま残す
					rt.Logger().Warn().Str("command", cmdLine).
						Msg("command substitution skipped: exec_commands is disabled")
					sb.WriteString(string(runes[i : nextIdx+1]))
					i = nextIdx // 閉じ括弧までスキップ
					continue
				}
				// 再帰的に中身を展開 (例: $(echo $(date)) の内側を先に解決)
				//expandedCmdLine := ExpandEnvAndCommands(env, fs, exec, console, cmdLine, envMap)
				expandedCmdLine := replaceVariables(rt, cmdLine, envMap)
				// 実行
				out := executeCommand(rt, expandedCmdLine, envMap)
				sb.WriteString(out)

				i = nextIdx // 閉じ括弧までスキップ
				continue
			}
		}
		sb.WriteRune(runes[i])
	}

	return sb.String()
}

// 内部用：Executor を使ってコマンドを実行するヘルパー
func executeCommand(rt interfaces.Runtime, cmdLine string, envMap any) string {
	var name string
	var args []string

	// OSごとのシェル呼び出しを定義
	if rt.Env().GOOS() == "windows" {
		name = "cmd"
		args = []string{"/c", cmdLine}
	} else {
		name = "sh"
		args = []string{"-c", cmdLine}
	}

	// 環境変数の組み立て
	currentEnv := rt.Env().Environ()
	currentEnv = append(currentEnv, getAllEnvsFromAny(rt, envMap)...)

	// Executor.Command を呼び出す (exec.Cmd 型を直接宣言せずに実行)
	// stdout を nil にすることで Output() 相当のキャプチャを可能にする設計を想定
	cmd, err := rt.Executor().Command(
		name,
		args,
		currentEnv,
		rt.Console().Stdin(),
		nil, // stdout: nil を渡して Output() で取得
		rt.Console().Stderr(),
		true, // suppress window (Windows)
	)
	if err != nil {
		rt.Logger().Warn().Err(err).Str("command", cmdLine).
			Msg("command substitution failed: could not create command")
		return ""
	}
	if rt.Env().GOOS() == "windows" {
		// Hand the line to cmd.exe verbatim: Go's default escaping
		// turns embedded quotes into \" which cmd cannot parse.
		UseRawCmdExeLine(cmd, cmdLine)
	}

	out, err := cmd.Output()
	if err != nil {
		rt.Logger().Warn().Err(err).Str("command", cmdLine).
			Msg("command substitution failed: command exited with an error")
		return ""
	}
	res := strings.TrimSpace(string(out))
	if rt.Env().GOOS() == "windows" {
		res = strings.Trim(res, "\"")
	}
	return res
}

func ExpandEnvAndCommandsSlice(rt interfaces.Runtime, inputs []string, envMap any) []string {
	result := make([]string, len(inputs))
	for i, s := range inputs {
		result[i] = ExpandEnvAndCommands(rt, s, envMap)
	}
	return result
}
