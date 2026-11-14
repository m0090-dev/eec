// engine.go
package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Engine is the core library entrypoint. It contains pluggable implementations
// for executing commands and file operations so CLI can inject mocks for tests.
type Engine struct {
	Runtime interfaces.Runtime
	// Out is the writer for human-readable reports (currently the
	// dependency tree). Nil means os.Stdout.
	Out io.Writer
}

// outWriter returns the configured report writer (default os.Stdout).
func (e *Engine) outWriter() io.Writer {
	if e.Out != nil {
		return e.Out
	}
	return os.Stdout
}

func (e *Engine) FS() interfaces.FS                   { return e.Runtime.FS() }
func (e *Engine) Env() interfaces.Env                 { return e.Runtime.Env() }
func (e *Engine) Executor() interfaces.Executor       { return e.Runtime.Executor() }
func (e *Engine) CommandLine() interfaces.CommandLine { return e.Runtime.CommandLine() }
func (e *Engine) Console() interfaces.Console         { return e.Runtime.Console() }

// NewEngine returns an Engine with sensible defaults (os-backed).

func NewEngine(rt interfaces.Runtime) *Engine {
	if rt == nil {
		temp := impl.DefaultRuntime{}
		rt = &temp
	}
	return &Engine{
		Runtime: rt,
	}
}

func (e *Engine) Run(ctx context.Context, opts types.RunOptions) error {
	var err error

	// -----------------------*/
	// 開始時環境変数表示
	// -----------------------*/
	envs := e.Env().Environ()
	e.Runtime.Logger().Debug().Str("Started envs", strings.Join(envs, ", ")).Msg("")

	e.Runtime.Logger().Debug().
		Str("config file", opts.ConfigFile).
		Str("program", opts.Program).
		Strs("Program args", opts.ProgramArgs).
		Str("tag", opts.Tag).
		Strs("imports", opts.Imports).
		Int("Wait timeout", int(opts.WaitTimeout)).
		Bool("Hide window", opts.HideWindow).
		Msg("Run called")

	if opts.Verbose {
		e.Runtime.SetLogger(
			e.Runtime.Logger().EnableDebug(),
		)
	}

	// ----------------------*/
	// タグデータ読み込み
	// -----------------------*/
	var tagData types.TagData
	if opts.Tag != "" {
		tagData, err = types.ReadTagData(e.Runtime, opts.Tag)
		if err != nil {
			e.Runtime.Logger().Error().Err(err).Str("tag", opts.Tag).Msg("failed to read tag")
			return fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
		}
	}
	// ----------------------*/
	// ResolveRunOptions 呼び出し
	// -----------------------*/
	_, program, pArgs, finalEnv, allConfigs, err := domain.ResolveRunOptions(opts, tagData, e.Runtime)
	if err != nil {
		return err // 循環参照などのエラーがあれば、ここで即座に終了
	}
	if program == "" {
		return errors.New("no program specified")
	}

	// #20: 環境変数の上書き追跡・表示
	tracker := types.TrackEnvOverrides(allConfigs, e.Runtime.Env().GOOS())
	if err := tracker.PrintOverrides(e.Runtime.Logger(), opts.DuplicateStrict, opts.DuplicateNoWarn); err != nil {
		return err
	}

	// ----------------------*/
	// Start process
	// -----------------------*/
	cmd, err := e.Executor().StartProcess(program, pArgs, finalEnv,
		e.Console().Stdin(), e.Console().Stdout(), e.Console().Stderr(), opts.HideWindow)
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	e.Runtime.Logger().Info().
		Int("ParentPID", e.Executor().Getpid()).
		Int("ChildPID", cmd.Process.Pid).
		Str("Program", program).
		Strs("ProgramArgs", pArgs).
		Msg("process started")

	// ----------------------*/
	// Wait for process
	// -----------------------*/
	if err := e.Executor().WaitProcess(cmd, opts.WaitTimeout); err != nil {
		// Preserve the child's exit code so the CLI can exit with it.
		var exitErr *types.ProcessExitError
		if errors.As(err, &exitErr) {
			return exitErr
		}
		return fmt.Errorf("process wait error: %w", err)
	}

	// -----------------------*/
	// 終了時環境変数表示
	// -----------------------*/
	envs = e.Env().Environ()
	e.Runtime.Logger().Debug().Str("Finished envs", strings.Join(envs, ", ")).Msg("")
	e.Runtime.Logger().Info().Msg("process finished normally")
	return nil
}

// ----------------- Stubs for other command core behaviors ------------------

// Gen performs generator-related core work (placeholder).
func (e *Engine) GenScript(outputDir string, clean bool) error {
	domain.GenUtilsScript(e.Runtime, outputDir, clean)
	return nil
}

// Info returns structured information about the environment or config.
func (e *Engine) Info() error {
	infos := map[string]interface{}{
		"version":    types.VERSION,
		"pid":        e.Executor().Getpid(),
		"goOS":       e.Env().GOOS(),
		"commitHash": types.BuildHash,
		"logMode":    types.LogMode,
	}

	// general.PrintBlock を呼び出す
	general.PrintBlock("eec Information", infos)
	return nil
}

// Tag-related core functions (create, list, delete).
func (e *Engine) TagAdd(name string, tag types.TagData) error {
	tagName := name

	// tag.ConfigFile が指定されている場合は、その設定ファイルを読み込み Program を補完
	if tag.ConfigFile != "" && tag.Program == "" {

		cfg, err := types.ReadConfig(e.Runtime, tag.ConfigFile)
		if err == nil {
			tag.Program = cfg.Program.Path
			tag.ProgramArgs = cfg.Program.Args
		} else {
			e.Runtime.Logger().Warn().Err(err).Str("configFile", tag.ConfigFile).Msg("failed to read config for program auto-fill")
		}
	}

	// make configFile absolute if present
	if tag.ConfigFile != "" {
		if e.FS().FileExists(tag.ConfigFile) {
			tag.ConfigFile = general.NormalizeSourcePath(tag.ConfigFile)
		}
	}
	// 2. ImportConfigFiles の各要素を正規化 (ここが足りなかった場所)
	for i, imp := range tag.ImportConfigFiles {
		// ファイルとして存在する場合のみ絶対パス化する
		if e.FS().FileExists(imp) {
			tag.ImportConfigFiles[i] = general.NormalizeSourcePath(imp)
		}
	}
	// 3. Phase 4: データディレクトリ内にあるファイルは、絶対パス
	// ではなくデータディレクトリ相対で保存する。ドライブレターが
	// 変わるなどデータディレクトリが移動してもタグが壊れない。
	// (TagData.Write 側でも同じ相対化を行うので、これは二重の安全策。)
	tag.ConfigFile = types.RelativizeDataDirPath(e.Runtime, tag.ConfigFile)
	for i, imp := range tag.ImportConfigFiles {
		tag.ImportConfigFiles[i] = types.RelativizeDataDirPath(e.Runtime, imp)
	}

	general.PrintBlock("Tag Add Debug", map[string]interface{}{
		"tagName":             tagName,
		"configFileFlag":      tag.ConfigFile,
		"programFlag":         tag.Program,
		"programArgsFlag":     strings.Join(tag.ProgramArgs, ", "),
		"Import config files": strings.Join(tag.ImportConfigFiles, ", "),
	})

	// タグファイル書き込み
	if err := tag.Write(e.Runtime, tagName); err != nil {
		e.Runtime.Logger().Error().Err(err).Msg("タグファイルの書き込みに失敗しました")
		return fmt.Errorf("Failed to tag file")
	}

	e.Runtime.Logger().Info().Str("Tag name", tagName).Msg("Tag added")
	return nil
}

func (e *Engine) TagRead(tagName string) error {
	data, err := types.ReadTagData(e.Runtime, tagName)
	if err != nil {
		e.Runtime.Logger().Error().Err(err).Msg("タグファイルの読み込みに失敗しました")
		return fmt.Errorf("Failed to tag read")
	}
	general.PrintBlock("Tag information", map[string]interface{}{
		"Tag":                 tagName,
		"Config":              data.ConfigFile,
		"Program":             data.Program,
		"Args":                strings.Join(data.ProgramArgs, ", "),
		"Import config files": strings.Join(data.ImportConfigFiles, ", "),
	})
	return nil
}
func (e *Engine) TagList() error {
	tagDir := types.ResolveDataDir(e.Runtime)
	if tagDir == "" {
		e.Runtime.Logger().Error().Msg("データディレクトリ (EEC_HOME / ~/.eec) が解決できません")
		return fmt.Errorf("Missing required homeDir")
	}
	fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")
	if err != nil {
		e.Runtime.Logger().Error().Err(err).Msg("タグファイルが見つかりませんでした")
		return fmt.Errorf("Failed to tag list")
	}
	general.PrintBlock("Current Tag Lists", map[string]interface{}{
		"Tags": strings.Join(fileLists, "\n"),
	})
	return nil
}

func (e *Engine) TagRemove(name string) error {
	tagName := name
	if err := types.ValidateTagName(tagName); err != nil {
		return err
	}
	tagDir := types.ResolveDataDir(e.Runtime)
	if tagDir == "" {
		e.Runtime.Logger().Error().Msg("データディレクトリ (EEC_HOME / ~/.eec) が解決できません")
		return fmt.Errorf("Missing required homeDir")
	}
	tagPath := filepath.Join(tagDir, fmt.Sprintf("%s.tag", tagName))
	err := e.FS().Remove(tagPath)
	if err != nil {
		e.Runtime.Logger().Error().
			Err(err).
			Str("tagName", tagName).
			Msg("Failed to remove tag file")
		return fmt.Errorf("failed to remove tag %s: %w", tagName, err)
	}
	e.TagList()
	return nil
}

func (e *Engine) Dump(opts types.RunOptions, shell string) error {
	var tagData types.TagData
	var err error

	if opts.Tag != "" {
		tagData, err = types.ReadTagData(e.Runtime, opts.Tag)
		if err != nil {
			e.Runtime.Logger().Error().Err(err).Str("tag", opts.Tag).Msg("failed to read tag")
			return fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
		}
	}

	_, _, _, finalEnv, _, err := domain.ResolveRunOptions(opts, tagData, e.Runtime)
	if err != nil {
		return err
	}

	for _, kv := range finalEnv {
		if line, ok := FormatDumpLine(kv, shell); ok {
			fmt.Println(line)
		}
	}
	return nil
}
