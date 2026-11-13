package domain_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// syncBuffer is a goroutine-safe output sink: the REPL under test
// runs in its own goroutine while the test polls the output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func waitForOutput(t *testing.T, b *syncBuffer, substr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(b.String(), substr) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("output did not contain %q within %s; output so far:\n%s", substr, timeout, b.String())
}

func waitForCount(t *testing.T, b *syncBuffer, substr string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Count(b.String(), substr) >= n {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("output did not contain %q x%d within %s; output so far:\n%s", substr, n, timeout, b.String())
}

func TestRunREPL_BuiltinsAndExternalCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based external command tests")
	}
	dir := t.TempDir()
	cfg := writeTempFile(t, dir, "repl.toml", "[envs]\nGREETING = \"from-config\"\n")

	lines := []string{
		"env GREETING", // GREETING=from-config
		"set GREETING=hello",
		"env GREETING",   // GREETING=hello
		"echo $GREETING", // external: prints a bare "hello" line
		"exit 42",        // external: prints [exit 42], REPL stays alive
		"env GREETING",   // still GREETING=hello
		"unset GREETING",
		"env GREETING", // GREETING is not set
		"help",
		"exit",
	}
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	out := &syncBuffer{}
	rt := &impl.DefaultRuntime{}

	if err := domain.RunREPL(context.Background(), rt, types.RunOptions{ConfigFile: cfg}, in, out, false); err != nil {
		t.Fatalf("RunREPL: %v", err)
	}
	s := out.String()

	if !strings.Contains(s, "GREETING=from-config") {
		t.Errorf("config value not shown:\n%s", s)
	}
	if n := strings.Count(s, "GREETING=hello"); n != 2 {
		t.Errorf("GREETING=hello appears %d times, want 2 (before and after 'exit 42'):\n%s", n, s)
	}
	if !strings.Contains(s, "\nhello\n") {
		t.Errorf("external echo did not print the session value:\n%s", s)
	}
	if !strings.Contains(s, "[exit 42]") {
		t.Errorf("non-zero external exit not reported:\n%s", s)
	}
	if !strings.Contains(s, "GREETING is not set") {
		t.Errorf("unset did not take effect:\n%s", s)
	}
	if !strings.Contains(s, "reload") {
		t.Errorf("help output missing:\n%s", s)
	}
	if strings.Contains(s, "eec> ") {
		t.Errorf("prompt must not appear in non-interactive mode:\n%s", s)
	}
}

func TestRunREPL_PromptOnlyWhenInteractive(t *testing.T) {
	out := &syncBuffer{}
	rt := &impl.DefaultRuntime{}
	in := strings.NewReader("exit\n")
	if err := domain.RunREPL(context.Background(), rt, types.RunOptions{}, in, out, true); err != nil {
		t.Fatalf("RunREPL: %v", err)
	}
	if !strings.Contains(out.String(), "eec> ") {
		t.Errorf("interactive mode must show the prompt:\n%s", out.String())
	}
}

func TestRunREPL_CdBuiltin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based external command tests")
	}
	dir := t.TempDir()
	lines := []string{
		"cd /nonexistent-eec-dir",
		"cd " + dir,
		"pwd", // external: prints the session cwd
		"cd",  // prints the session cwd
		"exit",
	}
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	out := &syncBuffer{}
	rt := &impl.DefaultRuntime{}
	if err := domain.RunREPL(context.Background(), rt, types.RunOptions{}, in, out, false); err != nil {
		t.Fatalf("RunREPL: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "cd: no such directory: /nonexistent-eec-dir") {
		t.Errorf("bad cd not reported:\n%s", s)
	}
	if n := strings.Count(s, dir); n < 2 {
		t.Errorf("session cwd %q appears %d times, want >= 2 (pwd + cd):\n%s", dir, n, s)
	}
}

// The reload test drives the REPL line by line through an io.Pipe so
// it can rewrite the config file mid-session — the exact workflow
// reload exists for.
func TestRunREPL_ReloadPicksUpConfigEdits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh-based external command tests")
	}
	dir := t.TempDir()
	cfg := writeTempFile(t, dir, "repl.toml", "[envs]\nV = \"one\"\n")

	pr, pw := io.Pipe()
	out := &syncBuffer{}
	rt := &impl.DefaultRuntime{}
	done := make(chan error, 1)
	go func() {
		done <- domain.RunREPL(context.Background(), rt, types.RunOptions{ConfigFile: cfg}, pr, out, false)
	}()
	defer pw.Close()

	send := func(line string) {
		t.Helper()
		if _, err := fmt.Fprintln(pw, line); err != nil {
			t.Fatalf("send %q: %v", line, err)
		}
	}

	send("env V")
	waitForOutput(t, out, "V=one", 3*time.Second)

	send("set KEEP=me")
	send("env KEEP")
	waitForOutput(t, out, "KEEP=me", 3*time.Second)

	// Edit the config while the session is alive, then reload.
	if err := os.WriteFile(cfg, []byte("[envs]\nV = \"two\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	send("reload")
	waitForOutput(t, out, "Reloaded", 3*time.Second)

	send("env V")
	waitForOutput(t, out, "V=two", 3*time.Second)

	// Session overrides survive the reload.
	send("env KEEP")
	waitForCount(t, out, "KEEP=me", 2, 3*time.Second)

	// An unset tombstone survives too: remove V from the session,
	// reload again, and it must stay gone even though the config
	// still defines it.
	send("unset V")
	send("reload")
	waitForCount(t, out, "Reloaded", 2, 3*time.Second)
	send("env V")
	waitForOutput(t, out, "V is not set", 3*time.Second)

	send("exit")
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RunREPL returned %v, want nil on exit", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunREPL did not return after exit")
	}
}
