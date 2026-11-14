//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// デフォルトターゲット
var Default = BuildCLIRelease
var defaultTargetBinaryName = "eec"
var targetExt = ""

// --- Common utilities ---
func run(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd.Run()
}

func hash() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

// buildModeArg: パッケージごとに ldflags / gcflags を組み立てる
func buildModeArg(mode, pkg string) (ldflags, gcflags []string) {
	h := hash()
	if strings.ToLower(mode) == "debug" {
		gcflags = []string{"all=-N -l"}
		ldflags = []string{
			fmt.Sprintf(`-X %s.LogMode=debug -X %s.BuildHash=%s`, pkg, pkg, h),
		}
	} else {
		ldflags = []string{
			fmt.Sprintf(`-s -w -X %s.LogMode=release -X %s.BuildHash=%s`, pkg, pkg, h),
		}
	}
	return
}

func projectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}

// --- CLI build ---
func BuildCLI(mode string) error {
	if mode == "" {
		mode = "release"
	}

	if os.Getenv("GOOS") == "windows" {
		targetExt = ".exe"
	} else {
		targetExt = ""
	}

	fmt.Printf("Building CLI (%s)...\n", mode)

	root := projectRoot()
	cliDir := filepath.Join(root, "cmd/eec/")
	buildFile := filepath.Join(root, "build", defaultTargetBinaryName+targetExt)

	ldflags, gcflags := buildModeArg(mode, "github.com/m0090-dev/eec/internal/ext/types")
	args := []string{"build", "-ldflags", strings.Join(ldflags, " "), "-o", buildFile}
	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}

	return run(cliDir, "go", args...)
}

func BuildCLIRelease() error { return BuildCLI("release") }
func BuildCLIDebug() error   { return BuildCLI("debug") }

// --- Shared library build ---
func BuildLib(mode string) error {
	if mode == "" {
		mode = "release"
	}

	if os.Getenv("GOOS") == "windows" {
		targetExt = ".dll"
	} else {
		targetExt = ".so"
	}

	fmt.Printf("Building shared lib (%s)...\n", mode)

	root := projectRoot()
	libDir := filepath.Join(root, "pkg/cexport")
	buildFile := filepath.Join(root, "build", "libcengine"+targetExt)

	ldflags, gcflags := buildModeArg(mode, "github.com/m0090-dev/eec/pkg/cexport/")
	args := []string{"build", "-buildmode=c-shared", "-ldflags", strings.Join(ldflags, " "), "-o", buildFile}
	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}

	return run(libDir, "go", args...)
}

func BuildLibRelease() error { return BuildLib("release") }
func BuildLibDebug() error   { return BuildLib("debug") }
