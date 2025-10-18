// engine.go
package core

import (
	//"io"
	//"strconv"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"time"
	//"syscall"
	"github.com/google/uuid"
	"github.com/m0090-dev/eec-go/core/interfaces"
	"github.com/m0090-dev/eec-go/core/interfaces/impl"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/m0090-dev/eec-go/core/utils/domain"
	"github.com/m0090-dev/eec-go/core/utils/general"
	//"github.com/aymanbagabas/go-pty"
	//"github.com/rs/zerolog/log"
	//"os"
	//"os/exec"
	"path/filepath"
	"strings"
)

// Engine is the core library entrypoint. It contains pluggable implementations
// for executing commands and file operations so CLI can inject mocks for tests.
type Engine struct {
	OS types.OS
	//PtyData types.PtyData
	Logger interfaces.Logger
}

func (e *Engine) FS() interfaces.FS                   { return e.OS.FS }
func (e *Engine) Env() interfaces.Env                 { return e.OS.Env }
func (e *Engine) Executor() interfaces.Executor       { return e.OS.Executor }
func (e *Engine) CommandLine() interfaces.CommandLine { return e.OS.CommandLine }
func (e *Engine) Console() interfaces.Console         { return e.OS.Console }

// NewEngine returns an Engine with sensible defaults (os-backed).

func NewEngine(os *types.OS, logger interfaces.Logger) *Engine {
	if os == nil {
		temp := types.NewOS()
		os = &temp
	}
	if logger == nil {
		logger = impl.NewDefaultLogger()
	}
	return &Engine{
		OS:     *os,
		Logger: logger,
	}
}

func (e *Engine) Run(ctx context.Context, opts types.RunOptions) error {
	var err error

	// -----------------------*/
	// 開始時環境変数表示
	// -----------------------*/
	envs := e.Env().Environ()
	e.Logger.Debug().Str("Started envs", strings.Join(envs, ", ")).Msg("")

	e.Logger.Debug().
		Str("config file", opts.ConfigFile).
		Str("program", opts.Program).
		Strs("Program args", opts.ProgramArgs).
		Str("tag", opts.Tag).
		Strs("imports", opts.Imports).
		Int("Wait timeout", int(opts.WaitTimeout)).
		Bool("Hide window", opts.HideWindow).
		Str("Deleter path", opts.DeleterPath).
		Bool("Deleter hide window", opts.DeleterHideWindow).
		Msg("Run called")

	// -----------------------*/
	// deleter起動
	// -----------------------*/
	if err := domain.LaunchDeleter(e.OS, e.Logger, opts); err != nil {
		return err
	}

	// ----------------------*/
	// タグデータ読み込み
	// -----------------------*/
	var tagData types.TagData
	if opts.Tag != "" {
		tagData, err = types.ReadTagData(e.OS, e.Logger, opts.Tag)
		if err != nil {
			e.Logger.Error().Err(err).Str("tag", opts.Tag).Msg("failed to read tag")
			return fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
		}
	}

	// ----------------------*/
	// メイン config 読み込み
	// -----------------------*/
	var config types.Config
	if opts.ConfigFile != "" && e.FS().FileExists(opts.ConfigFile) {
		config, err = types.ReadConfig(e.OS, e.Logger, opts.ConfigFile)
		if err != nil {
			e.Logger.Error().Err(err).Str("configFile", opts.ConfigFile).Msg("failed to read config")
			return fmt.Errorf("failed to read config %s: %w", opts.ConfigFile, err)
		}
	}

	// ----------------------*/
	// ResolveRunOptions 呼び出し
	// -----------------------*/
	configFile, program, pArgs, finalEnv := domain.ResolveRunOptions(opts, tagData, config, e.OS, e.Logger)
	if program == "" {
		return errors.New("no program specified")
	}

	// make configFile absolute if present
	if configFile != "" {
		if abs, err := filepath.Abs(configFile); err == nil {
			configFile = abs
		}
	}

	// ----------------------*/
	// build temp file
	// -----------------------*/
	selfProgram := e.CommandLine().Args()[0]
	tmpDir := e.FS().TempDir()
	tmpPrefix := fmt.Sprintf("%s_%s_%s.tmp",
		general.RemoveExtension(filepath.Base(selfProgram)),
		general.RemoveExtension(filepath.Base(program)),
		uuid.New().String(),
	)
	tmpPath := filepath.Join(tmpDir, tmpPrefix)
	tmpFile, err := e.FS().Create(tmpPath)
	if err != nil {
		e.Logger.Error().Err(err).Str("prefix", tmpPrefix).Msg("failed to create temp file")
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	e.Logger.Info().Str("tempFile", tmpPath).Msg("created temp file")

	// manifest 書き込み
	manifest := types.Manifest{
		TempFilePath: tmpPath,
		EECPID:       e.Executor().Getpid(),
	}
	if _, err := manifest.WriteToManifest(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// ----------------------*/
	// Start process
	// -----------------------*/
	var childPid int
	var cmd *exec.Cmd

	cmd, err = e.Executor().StartProcess(program, pArgs, finalEnv,
		e.Console().Stdin(), e.Console().Stdout(), e.Console().Stderr(), opts.HideWindow)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to start process: %w", err)
	}
	childPid = cmd.Process.Pid

	// ----------------------*/
	// write tempData immediately
	// -----------------------*/
	tempData := types.TempData{
		ParentPID:         e.Executor().Getpid(),
		ChildPID:          childPid,
		ConfigFile:        configFile,
		Program:           program,
		ProgramArgs:       pArgs,
		Tag:               opts.Tag,
		Imports:           opts.Imports,
		WaitTimeout:       int64(opts.WaitTimeout),
		HideWindow:        opts.HideWindow,
		DeleterPath:       opts.DeleterPath,
		DeleterHideWindow: opts.DeleterHideWindow,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(tempData); err != nil {
		tmpFile.Close()
		if !opts.Pty {
			_ = cmd.Process.Kill()
		}
		return fmt.Errorf("failed to encode temp data: %w", err)
	}
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		if !opts.Pty {
			_ = cmd.Process.Kill()
		}
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		if !opts.Pty {
			_ = cmd.Process.Kill()
		}
		return fmt.Errorf("failed to flush temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	e.Logger.Info().
		Int("ParentPID", tempData.ParentPID).
		Int("ChildPID", tempData.ChildPID).
		Str("ConfigFile", tempData.ConfigFile).
		Str("Program", tempData.Program).
		Strs("ProgramArgs", tempData.ProgramArgs).
		Msg("temp file written")

		// ----------------------*/
		// Wait for process
		// -----------------------*/
	if err := e.Executor().WaitProcess(cmd.Process, opts.WaitTimeout); err != nil {
		return fmt.Errorf("process wait error: %w", err)
	}

	// -----------------------*/
	// 終了時環境変数表示
	// -----------------------*/
	envs = e.Env().Environ()
	e.Logger.Debug().Str("Finished envs", strings.Join(envs, ", ")).Msg("")
	e.Logger.Info().Msg("process finished normally")
	return nil
}

// ----------------- Stubs for other command core behaviors ------------------

// Gen performs generator-related core work (placeholder).
func (e *Engine) GenScript() error {
	domain.GenUtilsScript(e.OS, e.Logger)
	domain.GenWrapScript(e.OS, e.Logger)
	return nil
}

// Info returns structured information about the environment or config.
func (e *Engine) Info() error {
	infos := []string{}
	infos = append(infos, fmt.Sprintf("version=%s", types.VERSION))
	infos = append(infos, fmt.Sprintf("pid=%d", e.Executor().Getpid()))
	infos = append(infos, fmt.Sprintf("goOS=%s", runtime.GOOS))
	infos = append(infos, fmt.Sprintf("commitHash=%s", types.BuildHash))
	infos = append(infos, fmt.Sprintf("buildMode=%s", types.BuildMode))
	e.Logger.Info().Strs("infos", infos).Msg("eec Info messages")
	return nil
}

func (e *Engine) Repl() error {
	return nil
}

// Tag-related core functions (create, list, delete).
func (e *Engine) TagAdd(name string, tag types.TagData) error {
	tagName := name

	// make configFile absolute if present
	if tag.ConfigFile != "" {
		if abs, err := filepath.Abs(tag.ConfigFile); err == nil {
			tag.ConfigFile = abs
		}
	}

	// -- デバッグ用 --
	e.Logger.Debug().
		Str("tagName", tagName).
		Msg("")
	e.Logger.Debug().
		Str("configFileFlag", tag.ConfigFile).
		Msg("")
	e.Logger.Debug().
		Str("programFlag", tag.Program).
		Msg("")
	e.Logger.Debug().
		Str("programArgsFlag", strings.Join(tag.ProgramArgs, ", ")).
		Msg("")
	e.Logger.Debug().
		Str("Import config files", strings.Join(tag.ImportConfigFiles, ", ")).
		Msg("")
	//

	if err := tag.Write(e.OS, e.Logger, tagName); err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルの書き込みに失敗しました")
		return fmt.Errorf("Failed to tag file")
	}
	e.Logger.Info().Str("Tag name", tagName).Msg("Tag added")
	return nil
}
func (e *Engine) TagRead(name string) error {
	tagName := name
	data, err := types.ReadTagData(e.OS, e.Logger, tagName)
	if err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルの読み込みに失敗しました")
		return fmt.Errorf("Failed to tag read")
	}

	e.Logger.Info().
		Str("Tag", tagName).
		Str("Config", data.ConfigFile).
		Str("Program", data.Program).
		Strs("Args", data.ProgramArgs).
		Strs("Import config files", data.ImportConfigFiles).
		Msg("Tag information")
	return nil
}
func (e *Engine) TagList() error {
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Logger.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		return fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, types.DEFAULT_TAG_DIR)
	fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")
	if err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルが見つかりませんでした")
		return fmt.Errorf("Failed to tag list")
	}
	e.Logger.Info().Str("message", "-- current tag lists --").Msg("Tag List Header")
	e.Logger.Info().Strs("tags", fileLists).Msg("Current tags")
	return nil
}

func (e *Engine) loadTempData() (types.TempData, string, error) {
	var td types.TempData

	// 1. OS の Temp にある manifest ファイルパスを取得
	tmpDir := e.FS().TempDir()
	manifestPath := filepath.Join(tmpDir, "eec_manifest.txt")

	// 2. manifest ファイルを読み込む
	content, err := e.FS().ReadFile(manifestPath)
	if err != nil {
		e.Logger.Error().Err(err).Str("manifestPath", manifestPath).Msg("failed to read manifest")
		return td, "", fmt.Errorf("failed to read manifest: %w", err)
	}

	// 3. 先頭のファイルパスだけを取り出す
	tmpFilePath := strings.TrimSpace(string(content))
	if idx := strings.Index(tmpFilePath, " "); idx != -1 {
		tmpFilePath = tmpFilePath[:idx]
	}
	if tmpFilePath == "" {
		e.Logger.Error().Str("manifestPath", manifestPath).Msg("manifest file is empty")
		return td, "", fmt.Errorf("manifest file is empty")
	}

	// 4. tempFile を開いて TempData をデコード
	f, err := e.FS().Open(tmpFilePath)
	if err != nil {
		e.Logger.Debug().Err(err).Str("tempFile", tmpFilePath).Msg("cannot open temp file")
		return td, tmpFilePath, fmt.Errorf("no temp file found for current ChildPID")
	}
	defer f.Close()

	if err := gob.NewDecoder(f).Decode(&td); err != nil {
		e.Logger.Error().Err(err).Str("tempFile", tmpFilePath).Msg("failed to decode temp data")
		return td, tmpFilePath, fmt.Errorf("failed to decode temp data: %w", err)
	}

	return td, tmpFilePath, nil
}

func (e *Engine) Restart() error {
	// 1. manifest ファイルパス取得
	tmpDir := e.FS().TempDir()
	manifestPath := filepath.Join(tmpDir, "eec_manifest.txt")

	// 2. manifest 読み込み
	content, err := e.FS().ReadFile(manifestPath)
	if err != nil {
		e.Logger.Error().Err(err).Msg("failed to read manifest")
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	// 3. 先頭のファイルパス抽出
	tmpFilePath := strings.TrimSpace(string(content))
	if idx := strings.Index(tmpFilePath, " "); idx != -1 {
		tmpFilePath = tmpFilePath[:idx]
	}
	if tmpFilePath == "" {
		e.Logger.Error().Msg("manifest file is empty")
		return fmt.Errorf("manifest file is empty")
	}

	// 4. TempData デコード
	f, err := e.FS().Open(tmpFilePath)
	if err != nil {
		e.Logger.Debug().Err(err).Str("tempFile", tmpFilePath).Msg("cannot open temp file")
		return fmt.Errorf("no temp file found for current ChildPID")
	}
	defer f.Close()

	var td types.TempData
	if err := gob.NewDecoder(f).Decode(&td); err != nil {
		e.Logger.Error().Err(err).Str("tempFile", tmpFilePath).Msg("failed to decode temp data")
		return fmt.Errorf("failed to decode temp data: %w", err)
	}

	// 5. 既存プロセス終了（PTY の場合は Kill をスキップ）
	if td.ChildPID != 0 {
		running, err := domain.IsPIDRunning(e.OS, e.Logger, td.ChildPID)
		if err != nil {
			e.Logger.Warn().Err(err).Int("ChildPID", td.ChildPID).Msg("failed to check if child PID, continuing")
		}
		if running {
			proc, err := e.Executor().FindProcess(td.ChildPID)
			if err == nil && proc != nil {
				if killErr := proc.Kill(); killErr != nil {
					e.Logger.Warn().Err(killErr).Int("ChildPID", td.ChildPID).Msg("failed to kill old child process, continuing")
				} else {
					e.Logger.Info().Int("ChildPID", td.ChildPID).Msg("old child process killed")
				}
			}
		} else {
			e.Logger.Debug().Int("ChildPID", td.ChildPID).Msg("no existing child process found")
		}
	}

	// 6. RunOptions 復元
	opts := types.RunOptions{
		ConfigFile:        td.ConfigFile,
		Program:           td.Program,
		ProgramArgs:       td.ProgramArgs,
		Tag:               td.Tag,
		Imports:           td.Imports,
		WaitTimeout:       time.Duration(td.WaitTimeout) * time.Millisecond,
		HideWindow:        td.HideWindow,
		DeleterPath:       td.DeleterPath,
		DeleterHideWindow: td.DeleterHideWindow,
	}

	// 7. 再起動処理

	// 通常プロセスは Run で再実行
	if err := e.Run(context.Background(), opts); err != nil {
		e.Logger.Error().Err(err).Msg("failed to restart process")
		return fmt.Errorf("failed to restart process: %w", err)
	}

	e.Logger.Info().Msg("process restarted successfully")
	return nil
}

func (e *Engine) TagRemove(name string) error {
	tagName := name
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Logger.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		return fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, types.DEFAULT_TAG_DIR)
	tagPath := filepath.Join(tagDir, fmt.Sprintf("%s.tag", tagName))
	err = e.FS().Remove(tagPath)
	if err != nil {
		e.Logger.Error().
			Err(err).
			Str("tagName", tagName).
			Msg("Failed to remove tag file")
		return fmt.Errorf("failed to remove tag %s: %w", tagName, err)
	}
	e.Logger.Info().Str("deletedTag", tagName).Msg("タグを削除しました")
	e.TagList()
	return nil
}
