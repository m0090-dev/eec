package types

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
)

// ExecutableDirFunc locates the directory containing the running eec
// executable. It is a variable so tests can substitute a fake location
// for portable-mode detection.
var ExecutableDirFunc = func() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

// ResolveDataDir returns the directory eec uses for persistent data
// (currently the tag store). Resolution order:
//
//  1. The EEC_HOME environment variable. Its value IS the data
//     directory itself (the equivalent of ~/.eec), not a parent of it.
//  2. Portable mode: if a ".eec" directory exists next to the running
//     executable, that directory is used. This lets a USB drive /
//     portable folder carry both the binary and its tags, with no
//     state left behind on the host machine.
//  3. The historical default: ".eec" under the user's home directory.
//
// An empty string is returned when no home directory is available and
// neither of the first two rules applies.
func ResolveDataDir(rt interfaces.Runtime) string {
	if v, ok := rt.Env().LookupEnv("EEC_HOME"); ok && v != "" {
		return v
	}
	if exeDir, err := ExecutableDirFunc(); err == nil && exeDir != "" {
		candidate := filepath.Join(exeDir, DEFAULT_TAG_DIR)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	homeDir, _ := rt.Env().UserHomeDir()
	if homeDir == "" {
		return ""
	}
	return filepath.Join(homeDir, DEFAULT_TAG_DIR)
}

// RelativizeDataDirPath expresses p relative to the resolved data
// directory when p lies inside it, so stored paths survive the data
// directory moving (e.g. a portable drive changing drive letters).
// Paths outside the data directory, relative paths, and paths on a
// different volume are returned unchanged.
func RelativizeDataDirPath(rt interfaces.Runtime, p string) string {
	if p == "" || !filepath.IsAbs(p) {
		return p
	}
	dataDir := ResolveDataDir(rt)
	if dataDir == "" {
		return p
	}
	rel, err := filepath.Rel(dataDir, p)
	if err != nil {
		return p
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return p
	}
	return rel
}
