package general

import (
    "os"
    "regexp"
)




// --------------------------
// 環境変数を展開して string を返す関数（$(VAR)形式）
// --------------------------
func ExpandEnvVariables(input string) string {
    re := regexp.MustCompile(`\$\(([^)]+)\)`)
    return re.ReplaceAllStringFunc(input, func(match string) string {
        submatches := re.FindStringSubmatch(match)
        if len(submatches) == 2 {
            if val, ok := os.LookupEnv(submatches[1]); ok {
                return val
            }
        }
        return ""
    })
}


// --------------------------
// ExpandEnvVariables の []string 版
// --------------------------
func ExpandEnvVariablesSlice(inputs []string) []string {
    result := make([]string, len(inputs))
    for i, s := range inputs {
        result[i] = ExpandEnvVariables(s)
    }
    return result
}
