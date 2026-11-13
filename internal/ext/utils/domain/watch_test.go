package domain_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// --- Pure change detection ---

func TestChangedPaths(t *testing.T) {
	t0 := time.Unix(1000, 0)
	t1 := time.Unix(2000, 0)
	base := map[string]domain.FileState{
		"a.toml": {Exists: true, ModTime: t0},
		"b.toml": {Exists: true, ModTime: t0},
		"c.toml": {Exists: false},
	}

	if got := domain.ChangedPaths(base, base); len(got) != 0 {
		t.Errorf("identical snapshots: changed = %v, want none", got)
	}

	mod := map[string]domain.FileState{
		"a.toml": {Exists: true, ModTime: t1}, // modified
		"b.toml": {Exists: false},             // deleted
		"c.toml": {Exists: true, ModTime: t1}, // appeared
	}
	got := domain.ChangedPaths(base, mod)
	if strings.Join(got, ",") != "a.toml,b.toml,c.toml" {
		t.Errorf("changed = %v, want [a.toml b.toml c.toml]", got)
	}

	partial := map[string]domain.FileState{
		"a.toml": {Exists: true, ModTime: t0},
		"b.toml": {Exists: true, ModTime: t0},
		"c.toml": {Exists: false},
		"d.toml": {Exists: true, ModTime: t1}, // new key
	}
	if got := domain.ChangedPaths(base, partial); strings.Join(got, ",") != "d.toml" {
		t.Errorf("changed = %v, want [d.toml]", got)
	}
}

func TestCollectWatchPaths(t *testing.T) {
	cfgs := []types.Config{
		{SourcePath: "/x/a.toml"},
		{SourcePath: ""}, // inline: skipped
		{SourcePath: "/x/b.toml"},
		{SourcePath: "/x/a.toml"}, // duplicate
	}
	got := domain.CollectWatchPaths(cfgs)
	if strings.Join(got, ",") != "/x/a.toml,/x/b.toml" {
		t.Errorf("CollectWatchPaths = %v", got)
	}
}

func TestRunWatch_NoFileConfigsIsError(t *testing.T) {
	rt := &impl.DefaultRuntime{}
	opts := types.RunOptions{
		InlineConfigToml: "[envs]\nFOO = \"bar\"\n",
		Program:          "sh",
	}
	err := domain.RunWatch(context.Background(), rt, opts, types.TagData{}, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "watch requires file-based configs") {
		t.Errorf("err = %v, want a 'watch requires file-based configs' error", err)
	}
}

// --- Integration: real child, real config file ---

func countLines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(data), "\n")
}

func waitForLines(t *testing.T, path string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if countLines(t, path) >= n {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("counter file %s did not reach %d lines within %s (got %d)", path, n, timeout, countLines(t, path))
}

func TestRunWatch_RestartsOnConfigChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script integration test")
	}
	oldPoll, oldSettle := domain.WatchPollInterval, domain.WatchSettleDelay
	domain.WatchPollInterval = 50 * time.Millisecond
	domain.WatchSettleDelay = 50 * time.Millisecond
	defer func() { domain.WatchPollInterval, domain.WatchSettleDelay = oldPoll, oldSettle }()

	dir := t.TempDir()
	counter := filepath.Join(dir, "counter.txt")
	script := filepath.Join(dir, "prog.sh")
	body := "#!/bin/sh\necho run >> \"" + counter + "\"\nexec sleep 30\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
	cfgPath := writeTempFile(t, dir, "watch.toml", "[envs]\nV = \"one\"\n")

	rt := &impl.DefaultRuntime{}
	opts := types.RunOptions{ConfigFile: cfgPath, Program: "sh", ProgramArgs: []string{script}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- domain.RunWatch(ctx, rt, opts, types.TagData{}, nil, nil, nil) }()

	waitForLines(t, counter, 1, 3*time.Second)

	// Edit the config; bump the mtime explicitly so the change is
	// visible regardless of filesystem timestamp granularity.
	if err := os.WriteFile(cfgPath, []byte("[envs]\nV = \"two\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(cfgPath, future, future); err != nil {
		t.Fatal(err)
	}

	waitForLines(t, counter, 2, 3*time.Second)

	cancel()
	select {
	case err := <-done:
		var exitErr *types.ProcessExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode != 130 {
			t.Errorf("after cancel, err = %v, want ProcessExitError{130}", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunWatch did not return after context cancellation")
	}
}

func TestRunWatch_ChildSelfExitPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-script integration test")
	}
	oldPoll := domain.WatchPollInterval
	domain.WatchPollInterval = 50 * time.Millisecond
	defer func() { domain.WatchPollInterval = oldPoll }()

	dir := t.TempDir()
	cfgPath := writeTempFile(t, dir, "watch.toml", "[envs]\nV = \"one\"\n")
	rt := &impl.DefaultRuntime{}
	opts := types.RunOptions{
		ConfigFile:  cfgPath,
		Program:     "sh",
		ProgramArgs: []string{"-c", "exit 7"},
	}
	err := domain.RunWatch(context.Background(), rt, opts, types.TagData{}, nil, nil, nil)
	var exitErr *types.ProcessExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode != 7 {
		t.Errorf("err = %v, want ProcessExitError{7}", err)
	}
}
