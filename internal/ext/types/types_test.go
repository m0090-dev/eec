package types_test

import (
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
		// Windows/Linux 両対応のため OS の TempDir を利用
		tempDir:  os.TempDir(),
		fileData: make(map[string][]byte),
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
	f, err := os.CreateTemp("", "mock-create-*")
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
	// 実ファイルに書き出すケースを考慮
	return os.ReadFile(path)
}
func (fs *mockFS) WriteFile(path string, data []byte, _ uint32) error {
	fs.fileData[path] = data
	fs.existingFiles[path] = true
	return nil
}
func (fs *mockFS) FileExt(p string) string  { return filepath.Ext(p) }
func (fs *mockFS) FileBase(p string) string { return filepath.Base(p) }

// Open / OpenFile をエラーにせず、実体（Temp）を返すように修正
func (fs *mockFS) Open(p string) (*os.File, error) {
	return os.Open(p)
}
func (fs *mockFS) OpenFile(p string, flag int, perm uint32) (*os.File, error) {
	// ディレクトリがないと言われないよう、親ディレクトリを作成
	_ = os.MkdirAll(filepath.Dir(p), 0755)
	fs.existingFiles[p] = true
	return os.OpenFile(p, flag, os.FileMode(perm))
}

func (fs *mockFS) Stat(p string) (os.FileInfo, error) { return os.Stat(p) }
func (fs *mockFS) IsNotExist(err error) bool          { return os.IsNotExist(err) }
func (fs *mockFS) O_APPEND() int                      { return os.O_APPEND }
func (fs *mockFS) O_CREATE() int                      { return os.O_CREATE }
func (fs *mockFS) O_WRONLY() int                      { return os.O_WRONLY }

type mockExecutor struct {
	pid          int
	startErr     error
	startedNames []string
}

func (ex *mockExecutor) Getpid() int                              { return ex.pid }
func (ex *mockExecutor) FindProcess(pid int) (*os.Process, error) { return os.FindProcess(pid) }
func (ex *mockExecutor) StartProcess(name string, args, env []string, stdin, stdout, stderr *os.File, hide bool) (*exec.Cmd, error) {
	if ex.startErr != nil {
		return nil, ex.startErr
	}
	ex.startedNames = append(ex.startedNames, name)
	return &exec.Cmd{}, nil
}
func (ex *mockExecutor) WaitProcess(_ *exec.Cmd, _ time.Duration) error { return nil }
func (ex *mockExecutor) Command(name string, args, env []string, stdin, stdout, stderr *os.File, hide bool) (*exec.Cmd, error) {
	return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}, nil
}

type mockCommandLine struct{ args []string }

func (c *mockCommandLine) Args() []string { return c.args }

type mockConsole struct{}

func (c *mockConsole) Stdin() *os.File  { return nil }
func (c *mockConsole) Stdout() *os.File { return nil }
func (c *mockConsole) Stderr() *os.File { return nil }

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
		env:     &mockEnv{environ: []string{}, homeDir: os.TempDir(), goos: "linux", lookupMap: map[string]string{}},
		fs:      newMockFS(),
		exec:    &mockExecutor{pid: 9999},
		logger:  &mockLogger{},
		cmdLine: &mockCommandLine{args: []string{"eec"}},
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
// types 固有のテストケース
// =============================================================

// --- tag.go のテスト ---

func TestTagData_SaveAndRead(t *testing.T) {
	rt := newMockRuntime()
	tag := &types.TagData{
		Program:     "python",
		ProgramArgs: []string{"script.py"},
	}

	// 保存[cite: 8]
	err := tag.Write(rt, "my-tag")
	if err != nil {
		t.Fatalf("Tag Write failed: %v", err)
	}

	// 読み込み[cite: 8]
	readTag, err := types.ReadTagData(rt, "my-tag")
	if err != nil {
		t.Fatalf("Tag Read failed: %v", err)
	}

	if readTag.Program != "python" {
		t.Errorf("expected python, got %s", readTag.Program)
	}
}

// --- config.go / environ.go のテスト ---

func TestConfig_SortEnvs(t *testing.T) {
	cfg := types.Config{
		Envs: []types.Environ{
			{Key: "PATH", Value: "${ROOT}/bin"}, // ROOT に依存
			{Key: "ROOT", Value: "/usr/local"},
		},
	}

	// mock runtime は linux (ケースセンシティブ) なので false。
	err := cfg.SortEnvsByDependency(false)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Envs[0].Key != "ROOT" {
		t.Errorf("Sort failed, expected ROOT first, got %s", cfg.Envs[0].Key)
	}
}

func TestConfig_BuildEnvs(t *testing.T) {
	rt := newMockRuntime()
	rt.env.lookupMap["USER"] = "tester"

	cfg := types.Config{
		Envs: []types.Environ{
			{Key: "APP_HOME", Value: "/home/${USER}/app"},          // 環境変数展開[cite: 11]
			{Key: "PATH", Value: []interface{}{"${APP_HOME}/bin"}}, // スライス形式のマージ
		},
	}

	// Linux 環境を想定
	newEnv := cfg.BuildEnvs(rt, []string{"EXISTING=val"}, ":")

	res := make(map[string]string)
	for _, e := range newEnv {
		parts := strings.SplitN(e, "=", 2)
		res[parts[0]] = parts[1]
	}

	if res["APP_HOME"] != "/home/tester/app" {
		t.Errorf("Expansion failed: %s", res["APP_HOME"])
	}
	if !strings.Contains(res["PATH"], "/home/tester/app/bin") {
		t.Errorf("Path merge failed: %s", res["PATH"])
	}
}

func TestTrackEnvOverrides(t *testing.T) {
	configs := []types.Config{
		{SourcePath: "file1.toml", Envs: []types.Environ{{Key: "VAR", Value: "old"}}},
		{SourcePath: "file2.toml", Envs: []types.Environ{{Key: "VAR", Value: "new"}}},
	}

	// 上書きの追跡[cite: 10] (linux: ケースセンシティブ)
	tracker := types.TrackEnvOverrides(configs, "linux")

	if tracker.Vars["VAR"].OverriddenBy != "file2.toml" {
		t.Errorf("Override tracking failed: %+v", tracker.Vars["VAR"])
	}
}
