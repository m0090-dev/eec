package domain

import (
	"path/filepath"
	"strings"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
)

// DefaultPathExt is the PATHEXT fallback used on Windows when the
// environment being searched does not define PATHEXT itself.
var DefaultPathExt = []string{".COM", ".EXE", ".BAT", ".CMD"}

// PathExtList returns the executable extensions to try on Windows,
// taken from the PATHEXT variable of env when present (entries are
// normalized to carry a leading dot) and DefaultPathExt otherwise.
// On non-Windows systems it returns nil: any file with the executable
// bit set qualifies there, regardless of its name.
func PathExtList(goos string, env []string) []string {
	if goos != "windows" {
		return nil
	}
	if v, ok := GetEnvValue(env, goos, "PATHEXT"); ok {
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
	return append([]string(nil), DefaultPathExt...)
}

// ExeCandidates returns the file names to try for a bare program
// name, in order of preference. On Unix it is just the name itself.
// On Windows a name that already carries an extension is used as-is
// (only PATHEXT extensions are executable, and cmd.exe semantics try
// the literal name first); a bare name is combined with every
// PATHEXT extension.
func ExeCandidates(goos string, env []string, name string) []string {
	if goos != "windows" {
		return []string{name}
	}
	if filepath.Ext(name) != "" {
		return []string{name}
	}
	exts := PathExtList(goos, env)
	cands := make([]string, 0, len(exts))
	for _, e := range exts {
		cands = append(cands, name+e)
	}
	return cands
}

// isExecutableFile reports whether path names an existing regular
// file that can be executed on goos: on Windows existence is enough
// (the extension rules live in ExeCandidates); on Unix the file must
// carry at least one executable bit.
func isExecutableFile(rt interfaces.Runtime, goos, path string) bool {
	info, err := rt.FS().Stat(path)
	if err != nil || info == nil || info.IsDir() {
		return false
	}
	if goos == "windows" {
		return true
	}
	return info.Mode().Perm()&0o111 != 0
}

// FindInPath searches the PATH of env for an executable called name
// and returns its absolute path. A name containing a path separator
// is checked directly (relative to the current directory) instead of
// being searched. The current directory is NOT searched for bare
// names, on any OS.
func FindInPath(rt interfaces.Runtime, env []string, name string) (string, bool) {
	goos := rt.Env().GOOS()
	if strings.ContainsAny(name, `/\`) {
		if !isExecutableFile(rt, goos, name) {
			return "", false
		}
		if abs, err := filepath.Abs(name); err == nil {
			return abs, true
		}
		return name, true
	}

	pathVal, _ := GetEnvValue(env, goos, "PATH")
	sep := ":"
	if goos == "windows" {
		sep = ";"
	}
	for _, dir := range strings.Split(pathVal, sep) {
		if dir == "" {
			continue
		}
		for _, cand := range ExeCandidates(goos, env, name) {
			full := filepath.Join(dir, cand)
			if isExecutableFile(rt, goos, full) {
				if abs, err := filepath.Abs(full); err == nil {
					return abs, true
				}
				return full, true
			}
		}
	}
	return "", false
}
