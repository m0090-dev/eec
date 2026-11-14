package domain_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// =============================================================
// Mock implementations — 実FSには一切触れない
// =============================================================

// --- LogEvent / Logger ---

type mockLogEvent struct{}

func (e *mockLogEvent) Str(k, v string) interfaces.Event                   { return e }
func (e *mockLogEvent) Strs(k string, v []string) interfaces.Event         { return e }
func (e *mockLogEvent) Int(k string, v int) interfaces.Event               { return e }
func (e *mockLogEvent) Bool(k string, v bool) interfaces.Event             { return e }
func (e *mockLogEvent) Err(err error) interfaces.Event                     { return e }
func (e *mockLogEvent) Interface(k string, v interface{}) interfaces.Event { return e }
func (e *mockLogEvent) Msg(msg string)                                     {}
func (e *mockLogEvent) Msgf(format string, args ...interface{})            {}

type mockLogger struct{}

func (l *mockLogger) Debug() interfaces.Event                                    { return &mockLogEvent{} }
func (l *mockLogger) Info() interfaces.Event                                     { return &mockLogEvent{} }
func (l *mockLogger) Warn() interfaces.Event                                     { return &mockLogEvent{} }
func (l *mockLogger) Error() interfaces.Event                                    { return &mockLogEvent{} }
func (l *mockLogger) Fatal() interfaces.Event                                    { return &mockLogEvent{} }
func (l *mockLogger) Panic() interfaces.Event                                    { return &mockLogEvent{} }
func (l *mockLogger) EnableDebug() interfaces.Logger                             { return l }
func (l *mockLogger) WithField(key string, value interface{}) interfaces.Logger  { return l }
func (l *mockLogger) WithFields(fields map[string]interface{}) interfaces.Logger { return l }
func (l *mockLogger) Level(lv interfaces.Level) interfaces.Logger                { return l }
func (l *mockLogger) Output(w io.Writer) interfaces.Logger                       { return l }

// --- Env ---

type mockEnv struct {
	environ   []string
	homeDir   string
	goos      string
	lookupMap map[string]string
}

func (e *mockEnv) Environ() []string { return e.environ }
func (e *mockEnv) LookupEnv(key string) (string, bool) {
	v, ok := e.lookupMap[key]
	return v, ok
}
func (e *mockEnv) Unsetenv(key string) error    { delete(e.lookupMap, key); return nil }
func (e *mockEnv) Setenv(key, val string) error { e.lookupMap[key] = val; return nil }
func (e *mockEnv) UserHomeDir() (string, error) { return e.homeDir, nil }
func (e *mockEnv) GOOS() string                 { return e.goos }
func (e *mockEnv) PathListSeparator() string {
	if e.goos == "windows" {
		return ";"
	}
	return ":"
}

// --- FS (in-memory のみ。os.* を一切呼ばない) ---

type mockFS struct {
	existingFiles map[string]bool
	tempDir       string
	fileData      map[string][]byte
	removedFiles  []string
	mkdirCalls    []string
	mkdirErr      error
	createErr     error
}

func newMockFS() *mockFS {
	return &mockFS{
		existingFiles: make(map[string]bool),
		tempDir:       "/mock/tmp",
		fileData:      make(map[string][]byte),
	}
}

func (fs *mockFS) FileExists(path string) bool { return fs.existingFiles[path] }
func (fs *mockFS) TempDir() string             { return fs.tempDir }
func (fs *mockFS) MkdirAll(path string, _ uint32) error {
	fs.mkdirCalls = append(fs.mkdirCalls, path)
	return fs.mkdirErr
}
func (fs *mockFS) Remove(path string) error {
	fs.removedFiles = append(fs.removedFiles, path)
	return nil
}
func (fs *mockFS) Create(path string) (*os.File, error) {
	if fs.createErr != nil {
		return nil, fs.createErr
	}
	f, err := os.CreateTemp("", "mock-*")
	if err != nil {
		return nil, err
	}
	fs.existingFiles[path] = true
	return f, nil
}
func (fs *mockFS) ReadFile(path string) ([]byte, error) {
	if d, ok := fs.fileData[path]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("mock: not found: %s", path)
}
func (fs *mockFS) WriteFile(path string, data []byte, _ uint32) error {
	fs.fileData[path] = data
	return nil
}
func (fs *mockFS) FileExt(p string) string  { return "" }
func (fs *mockFS) FileBase(p string) string { return p }
func (fs *mockFS) Open(p string) (*os.File, error) {
	return nil, fmt.Errorf("mock: not found: %s", p)
}
func (fs *mockFS) Stat(p string) (os.FileInfo, error) {
	if fs.existingFiles[p] {
		return nil, nil
	}
	return nil, fmt.Errorf("mock: not found: %s", p)
}
func (fs *mockFS) IsNotExist(err error) bool { return err != nil }
func (fs *mockFS) OpenFile(p string, flag int, perm uint32) (*os.File, error) {
	return nil, fmt.Errorf("mock: not found: %s", p)
}
func (fs *mockFS) O_APPEND() int { return os.O_APPEND }
func (fs *mockFS) O_CREATE() int { return os.O_CREATE }
func (fs *mockFS) O_WRONLY() int { return os.O_WRONLY }

// --- Executor (実プロセスを起動しない) ---

type mockExecutor struct {
	pid          int
	startErr     error
	startedNames []string
}

func (ex *mockExecutor) Getpid() int { return ex.pid }
func (ex *mockExecutor) FindProcess(pid int) (*os.Process, error) {
	return os.FindProcess(pid)
}
func (ex *mockExecutor) StartProcess(
	name string, args, env []string,
	stdin, stdout, stderr *os.File, hide bool,
) (*exec.Cmd, error) {
	if ex.startErr != nil {
		return nil, ex.startErr
	}
	ex.startedNames = append(ex.startedNames, name)
	return &exec.Cmd{}, nil
}
func (ex *mockExecutor) WaitProcess(_ *exec.Cmd, _ time.Duration) error { return nil }
func (ex *mockExecutor) Command(
	name string, args, env []string,
	stdin, stdout, stderr *os.File, hide bool,
) (*exec.Cmd, error) {
	return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}, nil
}

// --- CommandLine / Console ---

type mockCommandLine struct{ args []string }

func (c *mockCommandLine) Args() []string { return c.args }

type mockConsole struct{}

func (c *mockConsole) Stdin() *os.File  { return nil }
func (c *mockConsole) Stdout() *os.File { return nil }
func (c *mockConsole) Stderr() *os.File { return nil }

// --- Runtime ---

type mockRuntime struct {
	env     *mockEnv
	fs      *mockFS
	exec    *mockExecutor
	logger  *mockLogger
	cmdLine *mockCommandLine
	console *mockConsole
}

func newMockRuntime() *mockRuntime {
	return &mockRuntime{
		env:     &mockEnv{environ: []string{}, homeDir: "/mock/home", goos: "linux", lookupMap: map[string]string{}},
		fs:      newMockFS(),
		exec:    &mockExecutor{pid: 9999},
		logger:  &mockLogger{},
		cmdLine: &mockCommandLine{args: []string{"/usr/bin/eec"}},
		console: &mockConsole{},
	}
}

func (rt *mockRuntime) FS() interfaces.FS                   { return rt.fs }
func (rt *mockRuntime) Env() interfaces.Env                 { return rt.env }
func (rt *mockRuntime) Executor() interfaces.Executor       { return rt.exec }
func (rt *mockRuntime) Logger() interfaces.Logger           { return rt.logger }
func (rt *mockRuntime) SetLogger(_ interfaces.Logger)       {}
func (rt *mockRuntime) CommandLine() interfaces.CommandLine { return rt.cmdLine }
func (rt *mockRuntime) Console() interfaces.Console         { return rt.console }

// =============================================================
// order.go — ResolveRunOptions
// =============================================================

func TestResolveRunOptions_ProgramFromOpts(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{Program: "myapp"}
	_, program, _, _, allConfigs, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program != "myapp" {
		t.Errorf("program = %q, want myapp", program)
	}
	if len(allConfigs) != 0 {
		t.Errorf("allConfigs len = %d, want 0", len(allConfigs))
	}
}

func TestResolveRunOptions_ProgramFromTagData(t *testing.T) {
	rt := newMockRuntime()
	_, program, _, _, _, err := domain.ResolveRunOptions(types.RunOptions{}, types.TagData{Program: "tagged-prog"}, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program != "tagged-prog" {
		t.Errorf("program = %q, want tagged-prog", program)
	}
}

// opts.Program が tagData.Program より優先される
func TestResolveRunOptions_OptsProgramOverridesTag(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{Program: "from-opts"}
	tagData := types.TagData{Program: "from-tag"}
	_, program, _, _, _, err := domain.ResolveRunOptions(opts, tagData, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program != "from-opts" {
		t.Errorf("program = %q, want from-opts", program)
	}
}

func TestResolveRunOptions_ProgramArgsFromOpts(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{Program: "prog", ProgramArgs: []string{"-a", "-b"}}
	_, _, pArgs, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pArgs) != 2 || pArgs[0] != "-a" || pArgs[1] != "-b" {
		t.Errorf("pArgs = %v, want [-a -b]", pArgs)
	}
}

func TestResolveRunOptions_ProgramArgsFromTagData(t *testing.T) {
	rt := newMockRuntime()
	tagData := types.TagData{Program: "prog", ProgramArgs: []string{"--foo"}}
	_, _, pArgs, _, _, err := domain.ResolveRunOptions(types.RunOptions{}, tagData, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pArgs) != 1 || pArgs[0] != "--foo" {
		t.Errorf("pArgs = %v, want [--foo]", pArgs)
	}
}

// opts.ProgramArgs が tagData.ProgramArgs より優先される
func TestResolveRunOptions_OptsProgramArgsOverridesTag(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{ProgramArgs: []string{"--from-opts"}}
	tagData := types.TagData{ProgramArgs: []string{"--from-tag"}}
	_, _, pArgs, _, _, err := domain.ResolveRunOptions(opts, tagData, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pArgs) != 1 || pArgs[0] != "--from-opts" {
		t.Errorf("pArgs = %v, want [--from-opts]", pArgs)
	}
}

// finalEnv には mock Environ() の内容が含まれる
func TestResolveRunOptions_FinalEnvContainsOSEnv(t *testing.T) {
	rt := newMockRuntime()
	rt.env.environ = []string{"OS_VAR=hello"}
	opts := types.RunOptions{Program: "prog"}
	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, e := range finalEnv {
		if strings.HasPrefix(e, "OS_VAR=") {
			found = true
		}
	}
	if !found {
		t.Error("finalEnv should contain OS_VAR from mock Environ()")
	}
}

// configFile が FileExists=false のとき config 読み込みをスキップして allConfigs は空
func TestResolveRunOptions_NonExistentConfigFileSkipped(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{Program: "prog", ConfigFile: "/mock/nonexistent.toml"}
	_, _, _, _, allConfigs, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(allConfigs) != 0 {
		t.Errorf("allConfigs = %d, want 0 when config file not found", len(allConfigs))
	}
}

// =============================================================
// run.go — ReadOrFallback / ReadOrFallbackRecursive
// =============================================================

func TestReadOrFallback_NonExistentReturnsError(t *testing.T) {
	rt := newMockRuntime()
	_, err := domain.ReadOrFallback(types.RunOptions{}, rt, "ghost_tag")
	if err == nil {
		t.Error("expected error for nonexistent file/tag, got nil")
	}
}

func TestReadOrFallbackRecursive_NonExistentReturnsError(t *testing.T) {
	rt := newMockRuntime()
	_, err := domain.ReadOrFallbackRecursive(types.RunOptions{}, rt, "ghost_tag")
	if err == nil {
		t.Error("expected error for nonexistent file/tag, got nil")
	}
}

// 両関数は同じ入力に対して同じ振る舞いをする
func TestReadOrFallback_MatchesRecursive(t *testing.T) {
	rt := newMockRuntime()
	_, err1 := domain.ReadOrFallback(types.RunOptions{}, rt, "ghost")
	_, err2 := domain.ReadOrFallbackRecursive(types.RunOptions{}, rt, "ghost")
	if (err1 == nil) != (err2 == nil) {
		t.Errorf("ReadOrFallback(%v) vs ReadOrFallbackRecursive(%v) differ", err1, err2)
	}
}

// =============================================================
// gen.go — スクリプト文字列生成（FS 不使用の純粋関数）
// =============================================================

func TestGenWindowsTagUtilsScript_ContainsTagName(t *testing.T) {
	script := domain.GenWindowsTagUtilsScript("dev")
	if !strings.Contains(script, "dev") {
		t.Error("expected script to contain 'dev'")
	}
	if !strings.Contains(script, "\r\n") {
		t.Error("expected CRLF line endings")
	}
}

func TestGenWindowsTagUtilsScriptProgramFixed_ContainsTagName(t *testing.T) {
	script := domain.GenWindowsTagUtilsScriptProgramFixed("myprog")
	if !strings.Contains(script, "myprog") {
		t.Error("expected script to contain 'myprog'")
	}
	if !strings.Contains(script, "\r\n") {
		t.Error("expected CRLF line endings")
	}
}

func TestGenWindowsTagUtilsScript_DifferentTagsDiffer(t *testing.T) {
	s1 := domain.GenWindowsTagUtilsScript("alpha")
	s2 := domain.GenWindowsTagUtilsScript("beta")
	if s1 == s2 {
		t.Error("scripts for different tag names should differ")
	}
}

func TestGenUnixTagUtilsScript_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenUnixTagUtilsScript panicked: %v", r)
		}
	}()
	_ = domain.GenUnixTagUtilsScript("sometag")
}

// =============================================================
// gen.go — GenUtilsScript (mock FS 経由)
// =============================================================

func TestGenUtilsScript_EmptyHomeDir_NoPanic(t *testing.T) {
	rt := newMockRuntime()
	rt.env.homeDir = ""
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenUtilsScript panicked: %v", r)
		}
	}()
	domain.GenUtilsScript(rt, "", false)
}

func TestGenUtilsScript_NoTagFiles_NoPanic(t *testing.T) {
	rt := newMockRuntime()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenUtilsScript panicked: %v", r)
		}
	}()
	domain.GenUtilsScript(rt, "", false)
}
