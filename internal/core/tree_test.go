package core_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	core "github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// Phase 6-2: the tree overhaul renders values, badges, capture
// lines, override chains and a summary through Engine.Out.

var longValue = strings.Repeat("x", 100)

// setupTreeFixture builds a tag "dev" in a temp EEC_HOME with two
// config files: base.toml (plain list + prepend + OS section + a long
// value) imported by extra.toml (an override + a capture command).
func setupTreeFixture(t *testing.T) (basePath, extraPath string) {
	t.Helper()
	root := t.TempDir()
	dataDir := filepath.Join(root, "eec-home")
	cfgDir := filepath.Join(dataDir, "configs")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EEC_HOME", dataDir)

	basePath = filepath.Join(cfgDir, "base.toml")
	baseBody := fmt.Sprintf(`
[configs]
description = "base environment"

[envs]
PATH = ["/opt/tools/bin"]
GREETING = "hello"
SHARED = "from-base"
LONGVAL = %q

[envs.prepend]
PATH = ["/opt/priority"]

[envs.%s]
OSKEY = "osval"
`, longValue, runtime.GOOS)
	if err := os.WriteFile(basePath, []byte(baseBody), 0644); err != nil {
		t.Fatal(err)
	}

	extraPath = filepath.Join(cfgDir, "extra.toml")
	extraBody := `
[envs]
SHARED = "from-extra"
EXTRA = "x"

[capture]
commands = ["echo setup"]
`
	if err := os.WriteFile(extraPath, []byte(extraBody), 0644); err != nil {
		t.Fatal(err)
	}

	rt := &impl.DefaultRuntime{}
	tag := types.TagData{
		ConfigFile:        basePath,
		Program:           "myprog",
		ProgramArgs:       []string{"--flag"},
		ImportConfigFiles: []string{extraPath},
	}
	if err := tag.Write(rt, "dev"); err != nil {
		t.Fatalf("tag write: %v", err)
	}
	return basePath, extraPath
}

func newTreeEngine(buf *bytes.Buffer) *core.Engine {
	e := core.NewEngine(&impl.DefaultRuntime{})
	e.Out = buf
	return e
}

func TestTree_RendersValuesBadgesOverridesAndSummary(t *testing.T) {
	basePath, extraPath := setupTreeFixture(t)
	var buf bytes.Buffer
	e := newTreeEngine(&buf)
	if err := e.TreeWithOptions("dev", core.TreeOptions{}); err != nil {
		t.Fatalf("TreeWithOptions: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"Dependency tree for tag: dev",
		"Program: myprog --flag",
		basePath,  // full path, not just the basename
		extraPath, // imported file, full path
		"Imported file: ",
		"Env: GREETING = hello",
		"Env: PATH = /opt/tools/bin [list]",
		"Env: PATH = /opt/priority [prepend]",
		"Env: OSKEY = osval [os: " + runtime.GOOS + "]",
		"Capture: echo setup",
		// Overrides: the SHARED chain in resolution order + winner.
		"Overrides:",
		"SHARED",
		basePath + ": from-base",
		extraPath + ": from-extra",
		"final: from-extra",
		// 2 configs; base has 6 env entries (4 plain + prepend + OS),
		// extra has 2; PATH and SHARED are overridden.
		"Summary: 2 config(s), 8 env definition(s), 2 override(s),",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("tree output missing %q\n--- output ---\n%s", want, out)
		}
	}
	if !strings.Contains(out, "final variable(s)") {
		t.Errorf("summary missing final variable count\n%s", out)
	}
	// A bytes.Buffer is not a terminal: no ANSI codes may appear.
	if strings.Contains(out, "\x1b[") {
		t.Errorf("non-TTY output contains ANSI escapes\n%q", out)
	}
	// The legacy Tree entrypoint still works.
	var buf2 bytes.Buffer
	if err := newTreeEngine(&buf2).Tree("dev"); err != nil {
		t.Errorf("Tree (legacy): %v", err)
	}
	if !strings.Contains(buf2.String(), "Dependency tree for tag: dev") {
		t.Errorf("legacy Tree output missing header\n%s", buf2.String())
	}
}

func TestTree_ImportedTagUnknownAndAlreadyVisited(t *testing.T) {
	basePath, extraPath := setupTreeFixture(t)
	rt := &impl.DefaultRuntime{}

	sub := types.TagData{ConfigFile: extraPath, Program: "subprog"}
	if err := sub.Write(rt, "sub"); err != nil {
		t.Fatalf("sub tag write: %v", err)
	}
	dev2 := types.TagData{
		ConfigFile:        basePath,
		ImportConfigFiles: []string{"sub", basePath, "mystery-thing"},
	}
	if err := dev2.Write(rt, "dev2"); err != nil {
		t.Fatalf("dev2 tag write: %v", err)
	}

	var buf bytes.Buffer
	if err := newTreeEngine(&buf).TreeWithOptions("dev2", core.TreeOptions{}); err != nil {
		t.Fatalf("TreeWithOptions: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"Imported tag: sub",
		"Program: subprog",
		basePath + " (already visited)",
		"Unknown import: mystery-thing",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("tree output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestTree_TraceKeyShowsChainAndFinalValue(t *testing.T) {
	setupTreeFixture(t)
	var buf bytes.Buffer
	e := newTreeEngine(&buf)
	err := e.TreeWithOptions("dev", core.TreeOptions{Key: "PATH", Full: true})
	if err != nil {
		t.Fatalf("trace PATH: %v", err)
	}
	out := buf.String()

	sep := string(os.PathListSeparator)
	_ = sep
	for _, want := range []string{
		"PATH",
		"/opt/tools/bin [list]",
		"/opt/priority [prepend]",
		"final: ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("trace output missing %q\n--- output ---\n%s", want, out)
		}
	}
	// The final value must show the prepend part BEFORE the base /
	// appended parts.
	var finalLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "  final: ") {
			finalLine = strings.TrimPrefix(line, "  final: ")
		}
	}
	if finalLine == "" {
		t.Fatalf("no final line in trace output\n%s", out)
	}
	iPre := strings.Index(finalLine, "/opt/priority")
	iTools := strings.Index(finalLine, "/opt/tools/bin")
	if iPre < 0 || iTools < 0 || iPre > iTools {
		t.Errorf("final PATH = %q, want /opt/priority before /opt/tools/bin", finalLine)
	}
	// Trace mode prints ONLY the traced variable.
	if strings.Contains(out, "GREETING") || strings.Contains(out, "Dependency tree") {
		t.Errorf("trace output should contain only the traced variable\n%s", out)
	}
}

func TestTree_TraceMissingKeyErrors(t *testing.T) {
	setupTreeFixture(t)
	var buf bytes.Buffer
	e := newTreeEngine(&buf)
	err := e.TreeWithOptions("dev", core.TreeOptions{Key: "NOPE_NOT_DEFINED_XYZ"})
	if err == nil {
		t.Fatal("tracing an undefined key must return an error")
	}
	if !strings.Contains(err.Error(), "NOPE_NOT_DEFINED_XYZ") {
		t.Errorf("error = %q, want it to name the key", err)
	}
}

func TestTree_TruncationAndFull(t *testing.T) {
	setupTreeFixture(t)

	var buf bytes.Buffer
	if err := newTreeEngine(&buf).TreeWithOptions("dev", core.TreeOptions{}); err != nil {
		t.Fatalf("TreeWithOptions: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, longValue) {
		t.Errorf("default output must truncate the 100-char value\n%s", out)
	}
	if !strings.Contains(out, strings.Repeat("x", 59)+"…") {
		t.Errorf("default output should contain the value truncated at 60 chars with …\n%s", out)
	}

	buf.Reset()
	if err := newTreeEngine(&buf).TreeWithOptions("dev", core.TreeOptions{Full: true}); err != nil {
		t.Fatalf("TreeWithOptions --full: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, longValue) {
		t.Errorf("--full output must contain the complete value\n%s", out)
	}
	if strings.Contains(out, "…") {
		t.Errorf("--full output must not contain truncation marks\n%s", out)
	}
}
