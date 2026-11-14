package impl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// This file resolves bare program names against the PATH of the
// environment that will be handed to the child process.
//
// It exists because of a regression from the Phase 1 executor rework:
// before it, Windows always launched through cmd.exe, which resolved
// the program name using the child's environment. The direct-exec
// code instead relies on exec.Command, and Go resolves the name at
// Command-construction time using the PATH of the eec process itself,
// ignoring Cmd.Env. A program that exists only on the configured
// PATH (the normal eec setup, e.g. a tool on a portable dev drive)
// therefore stopped being found: `exec: "nvim": executable file not
// found in %PATH%`. Resolving the name against env ourselves, before
// calling exec.Command, restores the pre-Phase 1 behavior while
// keeping direct execution (and its quoting benefits).
//
// The search rules deliberately mirror domain.FindInPath (which
// powers `eec which`): bare names never search the current directory,
// Windows tries PATHEXT extensions from env, and Unix requires the
// executable bit. The helper is duplicated here instead of imported
// because the domain package's tests import impl, so impl importing
// domain would create an import cycle.

// envValueForOS returns the value of key in env ("K=V" entries). Key
// comparison follows the OS rule: case-insensitive on Windows,
// exact elsewhere.
func envValueForOS(env []string, key string) (string, bool) {
	for _, e := range env {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if runtime.GOOS == "windows" {
			if strings.EqualFold(k, key) {
				return v, true
			}
		} else if k == key {
			return v, true
		}
	}
	return "", false
}

// pathExtCandidates returns the executable extensions to try for a
// bare name on Windows: the env's PATHEXT when defined, else the
// standard fallback. On other OSes it returns nil.
func pathExtCandidates(env []string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	if v, ok := envValueForOS(env, "PATHEXT"); ok {
		var exts []string
		for _, e := range strings.Split(v, ";") {
			e = strings.TrimSpace(e)
			if e == "" {
				continue
			}
			if !strings.HasPrefix(e, ".") {
				e = "." + e
			}
			exts = append(exts, e)
		}
		if len(exts) > 0 {
			return exts
		}
	}
	return []string{".COM", ".EXE", ".BAT", ".CMD"}
}

// isExecutableCandidate reports whether path is an existing regular
// file that can be executed: on Windows existence is enough (the
// extension rules are applied by the caller), on Unix at least one
// executable bit must be set.
func isExecutableCandidate(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

// resolveProgramPath searches the PATH of env for a bare program
// name and returns the absolute path of the first match. A name that
// contains a path separator (or is absolute) is returned unchanged,
// and so is a name that env's PATH cannot resolve: exec.Command then
// falls back to the process PATH and, failing that, produces its
// usual "executable file not found" error.
func resolveProgramPath(path string, env []string) string {
	if path == "" || filepath.IsAbs(path) || strings.ContainsAny(path, `/\`) {
		return path
	}
	pathVal, ok := envValueForOS(env, "PATH")
	if !ok || pathVal == "" {
		return path
	}

	var candidates []string
	if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
		for _, ext := range pathExtCandidates(env) {
			candidates = append(candidates, path+ext)
		}
	} else {
		candidates = []string{path}
	}

	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			// Never search the current directory for bare names.
			continue
		}
		for _, cand := range candidates {
			full := filepath.Join(dir, cand)
			if isExecutableCandidate(full) {
				if abs, err := filepath.Abs(full); err == nil {
					return abs
				}
				return full
			}
		}
	}
	return path
}
