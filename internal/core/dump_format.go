package core

import (
	"fmt"
	"strings"
)

// FormatDumpLine formats a single resolved "KEY=VALUE" pair for the
// shell requested via `eec dump --shell`.
//
//   - "unix": export KEY='value' — the value is single-quoted, with
//     embedded single quotes escaped as '\” so values containing
//     spaces, semicolons, $ or backticks survive being sourced.
//   - "win": set "KEY=value" — the quoted form keeps cmd.exe from
//     splitting or interpreting spaces and separators in the value.
//   - anything else (including ""): the raw KEY=VALUE pair, unchanged.
//
// The second return value is false only for the "win" form when the
// pair has no '=' and therefore cannot be emitted as a set statement
// (the previous behavior silently skipped such lines).
func FormatDumpLine(kv, shell string) (string, bool) {
	switch shell {
	case "unix":
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return "export " + kv, true
		}
		escaped := strings.ReplaceAll(value, "'", `'\''`)
		return fmt.Sprintf("export %s='%s'", key, escaped), true
	case "win":
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return "", false
		}
		return fmt.Sprintf("set \"%s=%s\"", key, value), true
	default:
		return kv, true
	}
}
