package domain

import (
	"strings"

	"github.com/m0090-dev/eec/internal/ext/types"
)

// GetEnvValue looks up key in a "KEY=VALUE" list using the key case
// rules of goos (case-insensitive on Windows, exact elsewhere). The
// second return value reports whether the key was present at all.
func GetEnvValue(env []string, goos, key string) (string, bool) {
	for _, kv := range env {
		k, v, ok := strings.Cut(kv, "=")
		if ok && types.EnvKeyEqual(goos, k, key) {
			return v, true
		}
	}
	return "", false
}

// SetEnvValue replaces the first entry whose key matches key (under
// the case rules of goos), preserving that entry's original key
// spelling, or appends a new entry when the key is absent.
func SetEnvValue(env []string, goos, key, value string) []string {
	for i, kv := range env {
		k, _, ok := strings.Cut(kv, "=")
		if ok && types.EnvKeyEqual(goos, k, key) {
			env[i] = k + "=" + value
			return env
		}
	}
	return append(env, key+"="+value)
}

// RemoveEnvValue returns a copy of env with every entry whose key
// matches key (under the case rules of goos) removed.
func RemoveEnvValue(env []string, goos, key string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		k, _, ok := strings.Cut(kv, "=")
		if ok && types.EnvKeyEqual(goos, k, key) {
			continue
		}
		out = append(out, kv)
	}
	return out
}
