package types

import "strings"

// CanonicalEnvKey returns the canonical form of an environment variable
// key for the given OS. Windows environment variables are
// case-insensitive, so keys are canonicalized to upper case there (the
// historical eec behavior). On Linux/macOS variable names are
// case-sensitive, so the key is returned exactly as written — a config
// defining "http_proxy" must not silently become "HTTP_PROXY".
//
// This is the single canonicalization point for env keys: BuildEnvs,
// dependency sorting, expansion lookup and override tracking must all go
// through here (or through EnvKeyEqual) instead of calling
// strings.ToUpper directly. The general package keeps a small mirror of
// this rule because it cannot import types (types imports general).
func CanonicalEnvKey(goos, key string) string {
	if goos == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

// EnvKeysCaseInsensitive reports whether env keys are case-insensitive
// on the given OS (Windows only).
func EnvKeysCaseInsensitive(goos string) bool {
	return goos == "windows"
}

// EnvKeyEqual compares two env keys using the case rules of the given
// OS: case-insensitive on Windows, exact elsewhere.
func EnvKeyEqual(goos, a, b string) bool {
	if EnvKeysCaseInsensitive(goos) {
		return strings.EqualFold(a, b)
	}
	return a == b
}
