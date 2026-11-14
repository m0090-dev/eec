package core_deleter_test

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	deleter "github.com/m0090-dev/eec/internal/core-deleter"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
)

// =============================================================
// Mock implementations (deleter の挙動に合わせて調整)
// =============================================================

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

func (l *mockLogger) Debug() interfaces.Event                               { return &mockLogEvent{} }
func (l *mockLogger) Info() interfaces.Event                                { return &mockLogEvent{} }
func (l *mockLogger) Warn() interfaces.Event                                { return &mockLogEvent{} }
func (l *mockLogger) Error() interfaces.Event                               { return &mockLogEvent{} }
func (l *mockLogger) Fatal() interfaces.Event                               { return &mockLogEvent{} }
func (l *mockLogger) Panic() interfaces.Event                               { return &mockLogEvent{} }
func (l *mockLogger) EnableDebug() interfaces.Logger                        { return l }
func (l *mockLogger) WithField(k string, v interface{}) interfaces.Logger   { return l }
func (l *mockLogger) WithFields(f map[string]interface{}) interfaces.Logger { return l }
func (l *mockLogger) Level(lv interfaces.Level) interfaces.Logger           { return l }
func (l *mockLogger) Output(w io.Writer) interfaces.Logger                  { return l }

type mockEnv struct {
	goos    string
	environ []string
}

func (e *mockEnv) Environ() []string                 { return e.environ }
func (e *mockEnv) LookupEnv(k string) (string, bool) { return "", false }
func (e *mockEnv) Unsetenv(k string) error           { return nil }
func (e *mockEnv) Setenv(k, v string) error          { return nil }
func (e *mockEnv) UserHomeDir() (string, error)      { return os.TempDir(), nil }
func (e *mockEnv) GOOS() string                      { return e.goos }
func (e *mockEnv) PathListSeparator() string         { return string(os.PathListSeparator) }

type mockFS struct {
	existFiles   map[string]bool
	fileData     map[string][]byte
	removedFiles []string
}

func (fs *mockFS) FileExists(p string) bool          { return fs.existFiles[p] }
func (fs *mockFS) TempDir() string                   { return os.TempDir() }
func (fs *mockFS) MkdirAll(p string, _ uint32) error { return nil }
func (fs *mockFS) Remove(p string) error {
	fs.removedFiles = append(fs.removedFiles, p)
	delete(fs.existFiles, p)
	return nil
}
func (fs *mockFS) ReadFile(p string) ([]byte, error) {
	if d, ok := fs.fileData[p]; ok {
		return d, nil
	}
	return nil, os.ErrNotExist
}
func (fs *mockFS) WriteFile(p string, data []byte, _ uint32) error {
	fs.fileData[p] = data
	fs.existFiles[p] = true
	return nil
}
func (fs *mockFS) Create(p string) (*os.File, error) { return os.CreateTemp("", "mock-*") }
func (fs *mockFS) FileExt(p string) string           { return filepath.Ext(p) }
func (fs *mockFS) FileBase(p string) string          { return filepath.Base(p) }
func (fs *mockFS) IsNotExist(err error) bool         { return os.IsNotExist(err) }
func (fs *mockFS) Stat(p string) (os.FileInfo, error) {
	if fs.existFiles[p] {
		return os.Stat(os.Args[0])
	} // ダミーのFileInfo
	return nil, os.ErrNotExist
}
func (fs *mockFS) Open(p string) (*os.File, error) {
	// bufio.Scanner が読めるように実ファイルを貸し出す
	if d, ok := fs.fileData[p]; ok {
		tmp, _ := os.CreateTemp("", "eec-deleter-read-*")
		tmp.Write(d)
		tmp.Seek(0, 0)
		return tmp, nil
	}
	return nil, os.ErrNotExist
}
func (fs *mockFS) OpenFile(p string, flag int, perm uint32) (*os.File, error) { return nil, nil }
func (fs *mockFS) O_APPEND() int                                              { return os.O_APPEND }
func (fs *mockFS) O_CREATE() int                                              { return os.O_CREATE }
func (fs *mockFS) O_WRONLY() int                                              { return os.O_WRONLY }

type mockExecutor struct{}

func (ex *mockExecutor) Getpid() int                              { return 9999 }
func (ex *mockExecutor) FindProcess(pid int) (*os.Process, error) { return nil, nil }
func (ex *mockExecutor) StartProcess(n string, a, e []string, si, so, se *os.File, h bool) (*exec.Cmd, error) {
	return nil, nil
}
func (ex *mockExecutor) WaitProcess(_ *exec.Cmd, _ time.Duration) error { return nil }
func (ex *mockExecutor) Command(n string, a, e []string, si, so, se *os.File, h bool) (*exec.Cmd, error) {
	// Windows では "echo" 単体では実行できないため、cmd /c echo を使用するか、
	// そもそも外部コマンドに依存しない空のコマンドを返します。
	if os.Getenv("GOOS") == "windows" || filepath.Separator == '\\' {
		return exec.Command("cmd", "/c", "exit 0"), nil
	}
	return exec.Command("true"), nil
}

type mockRuntime struct {
	fs   *mockFS
	env  *mockEnv
	exec *mockExecutor
}

func (rt *mockRuntime) FS() interfaces.FS                   { return rt.fs }
func (rt *mockRuntime) Env() interfaces.Env                 { return rt.env }
func (rt *mockRuntime) Executor() interfaces.Executor       { return rt.exec }
func (rt *mockRuntime) Logger() interfaces.Logger           { return &mockLogger{} }
func (rt *mockRuntime) SetLogger(_ interfaces.Logger)       {}
func (rt *mockRuntime) CommandLine() interfaces.CommandLine { return nil }
func (rt *mockRuntime) Console() interfaces.Console         { return &mockConsole{} }

type mockConsole struct{}

func (c *mockConsole) Stdin() *os.File  { return os.Stdin }
func (c *mockConsole) Stdout() *os.File { return os.Stdout }
func (c *mockConsole) Stderr() *os.File { return os.Stderr }

// =============================================================
// Test Cases
// =============================================================

func TestEngine_Run_NormalFlow(t *testing.T) {
	// 1. セットアップ
	rt := &mockRuntime{
		fs:   &mockFS{existFiles: make(map[string]bool), fileData: make(map[string][]byte)},
		env:  &mockEnv{goos: "linux"},
		exec: &mockExecutor{},
	}

	// 削除されるべき一時ファイル
	tempFile := filepath.Join(os.TempDir(), "eec_temp_123.tmp")
	rt.fs.existFiles[tempFile] = true

	// マニフェストファイルの準備[cite: 14]
	manifestPath := filepath.Join(os.TempDir(), types.DEFAULT_MANIFEST_FILE_NAME+".jsonl")
	entry := types.ManifestEntry{
		TempFilePath: tempFile,
		EECPID:       8888,
	}
	line, _ := json.Marshal(entry)
	rt.fs.WriteFile(manifestPath, append(line, '\n'), 0644)

	// 2. 実行
	e := deleter.NewEngine(rt)

	// Run は remaining == 0 になると manifestPath を Remove して終了する[cite: 14]
	err := e.Run()
	if err != nil {
		t.Fatalf("Engine.Run() failed: %v", err)
	}

	// 3. 検証
	// 一時ファイルが削除されたか
	tempRemoved := false
	for _, f := range rt.fs.removedFiles {
		if f == tempFile {
			tempRemoved = true
		}
	}
	if !tempRemoved {
		t.Error("Temporary file was not removed")
	}

	// マニフェストファイルが最後に削除されたか
	manifestRemoved := false
	for _, f := range rt.fs.removedFiles {
		if f == manifestPath {
			manifestRemoved = true
		}
	}
	if !manifestRemoved {
		t.Error("Manifest file was not removed")
	}
}

func TestNewEngine_NilRuntime(t *testing.T) {
	e := deleter.NewEngine(nil)
	if e == nil {
		t.Fatal("NewEngine(nil) is nil")
	}
	if e.Runtime == nil {
		t.Error("Default runtime not set")
	}
}
