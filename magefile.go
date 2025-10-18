//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// デフォルトターゲット（引数なし）
var Default = BuildCLIRelease

// --- Common utilities ---
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func hash() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func buildModeArg(mode string) (ldflags, gcflags []string) {
	h := hash()
	if strings.ToLower(mode) == "debug" {
		gcflags = []string{"all=-N -l"}
		ldflags = []string{fmt.Sprintf(`-X github.com/m0090-dev/eec-go/core/types.BuildMode=debug -X github.com/m0090-dev/eec-go/core/types.BuildHash=%s`, h)}
	} else {
		ldflags = []string{fmt.Sprintf(`-s -w -X github.com/m0090-dev/eec-go/core/types.BuildMode=release -X github.com/m0090-dev/eec-go/core/types.BuildHash=%s`, h)}
	}
	return
}

// --- CLI build (引数付き本体) ---
func BuildCLI(mode string) error {
	if mode == "" {
		mode = "release"
	}
	fmt.Printf("Building CLI (%s)...\n", mode)
	os.Chdir("cli")
	defer os.Chdir("..")

	ldflags, gcflags := buildModeArg(mode)
	args := []string{"build", "-ldflags", strings.Join(ldflags, " "), "-o", "../build/eec.exe"}
	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}
	return run("go", args...)
}

// --- CLI: release/debug ラッパー ---
func BuildCLIRelease() error {
	return BuildCLI("release")
}

func BuildCLIDebug() error {
	return BuildCLI("debug")
}

// --- GUI build ---
func BuildGUI(mode string) error {
	if mode == "" {
		mode = "release"
	}
	fmt.Printf("Building GUI (%s)...\n", mode)
	os.Chdir("gui/csharp/GEEC")
	defer os.Chdir("../../..")

	conf := "Release"
	if strings.ToLower(mode) == "debug" {
		conf = "Debug"
	}
	return run("dotnet", "publish", "-c", conf, "-r", "win-x64", "--self-contained", "true", "-o", "../../../build/gui/")
}

// --- GUI: release/debug ラッパー ---
func BuildGUIRelease() error {
	return BuildGUI("release")
}

func BuildGUIDebug() error {
	return BuildGUI("debug")
}

// --- Shared library build ---
func BuildLib(mode string) error {
	if mode == "" {
		mode = "release"
	}
	fmt.Printf("Building shared lib (%s)...\n", mode)
	os.Chdir("core/cexport")
	defer os.Chdir("../..")

	ldflags, gcflags := buildModeArg(mode)
	args := []string{"build", "-buildmode=c-shared", "-ldflags", strings.Join(ldflags, " "), "-o", "../../build/libcengine.dll"}
	if len(gcflags) > 0 {
		args = append(args, "-gcflags", strings.Join(gcflags, " "))
	}
	return run("go", args...)
}

// --- Lib: release/debug ラッパー ---
func BuildLibRelease() error {
	return BuildLib("release")
}

func BuildLibDebug() error {
	return BuildLib("debug")
}
