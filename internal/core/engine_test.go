package core_test

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	core "github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// =============================================================
// Mock implementations
// =============================================================

// --- LogEvent / Logger ---

type coreLogEvent struct{}

func (e *coreLogEvent) Str(k, v string) interfaces.Event                   { return e }
func (e *coreLogEvent) Strs(k string, v []string) interfaces.Event         { return e }
func (e *coreLogEvent) Int(k string, v int) interfaces.Event               { return e }
func (e *coreLogEvent) Bool(k string, v bool) interfaces.Event             { return e }
func (e *coreLogEvent) Err(err error) interfaces.Event                     { return e }
func (e *coreLogEvent) Interface(k string, v interface{}) interfaces.Event { return e }
func (e *coreLogEvent) Msg(msg string)                                     {}
func (e *coreLogEvent) Msgf(format string, args ...interface{})            {}

type coreLogger struct{}

func (l *coreLogger) Debug() interfaces.Event                                    { return &coreLogEvent{} }
func (l *coreLogger) Info() interfaces.Event                                     { return &coreLogEvent{} }
func (l *coreLogger) Warn() interfaces.Event                                     { return &coreLogEvent{} }
func (l *coreLogger) Error() interfaces.Event                                    { return &coreLogEvent{} }
func (l *coreLogger) Fatal() interfaces.Event                                    { return &coreLogEvent{} }
func (l *coreLogger) Panic() interfaces.Event                                    { return &coreLogEvent{} }
func (l *coreLogger) EnableDebug() interfaces.Logger                             { return l }
func (l *coreLogger) WithField(key string, value interface{}) interfaces.Logger  { return l }
func (l *coreLogger) WithFields(fields map[string]interface{}) interfaces.Logger { return l }
func (l *coreLogger) Level(lv interfaces.Level) interfaces.Logger                { return l }
func (l *coreLogger) Output(w io.Writer) interfaces.Logger                       { return l }

// --- Env ---

type coreEnv struct {
	environ   []string
	homeDir   string
	goos      string
	lookupMap map[string]string
}

func (e *coreEnv) Environ() []string { return e.environ }
func (e *coreEnv) LookupEnv(k string) (string, bool) {
	v, ok := e.lookupMap[k]
	return v, ok
}
func (e *coreEnv) Unsetenv(k string) error      { delete(e.lookupMap, k); return nil }
func (e *coreEnv) Setenv(k, v string) error     { e.lookupMap[k] = v; return nil }
func (e *coreEnv) UserHomeDir() (string, error) { return e.homeDir, nil }
func (e *coreEnv) GOOS() string                 { return e.goos }
func (e *coreEnv) PathListSeparator() string    { return string(os.PathListSeparator) }

// --- FS ---

type coreFS struct {
	existFiles   map[string]bool
	tempDir      string
	fileData     map[string][]byte
	removedFiles []string
	mkdirCalls   []string
	createErr    error
	mkdirErr     error
}

func newCoreFS() *coreFS {
	return &coreFS{
		existFiles: make(map[string]bool),
		tempDir:    os.TempDir(),
		fileData:   make(map[string][]byte),
	}
}

func (fs *coreFS) FileExists(p string) bool { return fs.existFiles[p] }
func (fs *coreFS) TempDir() string          { return fs.tempDir }
func (fs *coreFS) MkdirAll(p string, _ uint32) error {
	fs.mkdirCalls = append(fs.mkdirCalls, p)
	return fs.mkdirErr
}
func (fs *coreFS) Remove(p string) error {
	fs.removedFiles = append(fs.removedFiles, p)
	return nil
}
func (fs *coreFS) ReadFile(p string) ([]byte, error) {
	if d, ok := fs.fileData[p]; ok {
		return d, nil
	}
	return nil, os.ErrNotExist
}
func (fs *coreFS) WriteFile(p string, data []byte, _ uint32) error {
	fs.fileData[p] = data
	return nil
}

func (fs *coreFS) Create(p string) (*os.File, error) {
	if fs.createErr != nil {
		return nil, fs.createErr
	}
	// パニック回避のため実ファイルを作成
	f, err := os.CreateTemp("", "eec-mock-*")
	if err == nil {
		fs.existFiles[p] = true
	}
	return f, err
}

func (fs *coreFS) FileExt(p string) string  { return "" }
func (fs *coreFS) FileBase(p string) string { return p }
func (fs *coreFS) Open(p string) (*os.File, error) {
	return os.Open(os.DevNull)
}

func (fs *coreFS) Stat(p string) (os.FileInfo, error) {
	if fs.existFiles[p] {
		// 適当な実ファイルのStatを返してインターフェースを満たす
		return os.Stat(os.Args[0])
	}
	return nil, os.ErrNotExist
}

func (fs *coreFS) IsNotExist(err error) bool { return os.IsNotExist(err) }
func (fs *coreFS) OpenFile(p string, flag int, perm uint32) (*os.File, error) {
	// Manifest writes must go to a real writable file, otherwise
	// Engine.Run can never get past WriteToManifest in tests.
	return os.OpenFile(p, flag, os.FileMode(perm))
}
func (fs *coreFS) O_APPEND() int { return os.O_APPEND }
func (fs *coreFS) O_CREATE() int { return os.O_CREATE }
func (fs *coreFS) O_WRONLY() int { return os.O_WRONLY }

// --- Executor ---

type coreExecutor struct {
	pid      int
	startErr error
	waitErr  error
}

func (ex *coreExecutor) Getpid() int { return ex.pid }
func (ex *coreExecutor) FindProcess(pid int) (*os.Process, error) {
	return os.FindProcess(pid)
}

func (ex *coreExecutor) StartProcess(
	name string, args, env []string,
	stdin, stdout, stderr *os.File, hide bool,
) (*exec.Cmd, error) {
	if ex.startErr != nil {
		// エラー時でも .Process.Pid へのアクセスで落ちないよう Cmd を構成
		cmd := exec.Command("echo")
		return cmd, ex.startErr
	}
	// 実際にプロセスは動かさないが、構造体として正しく初期化されたものを返す
	cmd := exec.Command("echo")
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func (ex *coreExecutor) WaitProcess(_ *exec.Cmd, _ time.Duration) error { return ex.waitErr }

func (ex *coreExecutor) Command(
	name string, args, env []string,
	stdin, stdout, stderr *os.File, hide bool,
) (*exec.Cmd, error) {
	// 実際に tasklist 等が呼ばれてもいいように exec.Command を使う
	// (tasklist /FI ... は Windows 以外ではエラーになるが、パニックはしない)
	cmd := exec.Command(name, args...)
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd, nil
}

// --- CommandLine / Console ---

type coreCmdLine struct{ args []string }

func (c *coreCmdLine) Args() []string { return c.args }

type coreConsole struct{}

func (c *coreConsole) Stdin() *os.File  { return os.Stdin }
func (c *coreConsole) Stdout() *os.File { return os.Stdout }
func (c *coreConsole) Stderr() *os.File { return os.Stderr }

// --- Runtime ---

type coreRuntime struct {
	env     *coreEnv
	fs      *coreFS
	exec    *coreExecutor
	logger  *coreLogger
	cmdLine *coreCmdLine
	console *coreConsole
}

func newCoreRuntime() *coreRuntime {
	return &coreRuntime{
		env:     &coreEnv{environ: []string{}, homeDir: os.TempDir(), goos: "linux", lookupMap: map[string]string{}},
		fs:      newCoreFS(),
		exec:    &coreExecutor{pid: os.Getpid()},
		logger:  &coreLogger{},
		cmdLine: &coreCmdLine{args: []string{"eec"}},
		console: &coreConsole{},
	}
}

func (rt *coreRuntime) FS() interfaces.FS                   { return rt.fs }
func (rt *coreRuntime) Env() interfaces.Env                 { return rt.env }
func (rt *coreRuntime) Executor() interfaces.Executor       { return rt.exec }
func (rt *coreRuntime) Logger() interfaces.Logger           { return rt.logger }
func (rt *coreRuntime) SetLogger(_ interfaces.Logger)       {}
func (rt *coreRuntime) CommandLine() interfaces.CommandLine { return rt.cmdLine }
func (rt *coreRuntime) Console() interfaces.Console         { return rt.console }

// =============================================================
// Tests
// =============================================================

func TestNewEngine_NilRuntime_UsesDefault(t *testing.T) {
	e := core.NewEngine(nil)
	if e == nil {
		t.Error("NewEngine(nil) returned nil")
	}
}

func TestEngine_Run_NoProgramReturnsError(t *testing.T) {
	rt := newCoreRuntime()
	e := core.NewEngine(rt)
	err := e.Run(context.Background(), types.RunOptions{})
	if err == nil {
		t.Error("expected error when no program specified, got nil")
	}
}

func TestEngine_Run_StartProcessError_ReturnsError(t *testing.T) {
	rt := newCoreRuntime()
	rt.exec.startErr = os.ErrPermission
	e := core.NewEngine(rt)
	err := e.Run(context.Background(), types.RunOptions{Program: "prog"})
	if err == nil {
		t.Error("expected error when StartProcess fails, got nil")
	}
}

func TestEngine_Run_ChildExitCode_Propagated(t *testing.T) {
	rt := newCoreRuntime()
	rt.exec.waitErr = &types.ProcessExitError{ExitCode: 7}
	e := core.NewEngine(rt)
	err := e.Run(context.Background(), types.RunOptions{Program: "prog"})
	var exitErr *types.ProcessExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *types.ProcessExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.ExitCode)
	}
}

func TestEngine_Run_VerboseFlag_NoPanic(t *testing.T) {
	rt := newCoreRuntime()
	e := core.NewEngine(rt)
	_ = e.Run(context.Background(), types.RunOptions{Verbose: true, Program: "echo"})
}

func TestEngine_TagAdd_NoConfigFile_WritesTag(t *testing.T) {
	rt := newCoreRuntime()
	e := core.NewEngine(rt)
	tag := types.TagData{Program: "prog"}
	if err := e.TagAdd("testtag", tag); err != nil {
		t.Errorf("TagAdd returned error: %v", err)
	}
}

func TestEngine_Dump_WinFormat_ReturnsNil(t *testing.T) {
	rt := newCoreRuntime()
	rt.env.environ = []string{"MY_VAR=hello"}
	e := core.NewEngine(rt)
	if err := e.Dump(types.RunOptions{}, "win"); err != nil {
		t.Errorf("Dump(win) returned error: %v", err)
	}
}
