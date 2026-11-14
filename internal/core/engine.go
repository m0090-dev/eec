// engine.go
package core

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
	"path/filepath"
	"strings"
)

// Engine is the core library entrypoint. It contains pluggable implementations
// for executing commands and file operations so CLI can inject mocks for tests.
type Engine struct {
	Runtime interfaces.Runtime
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
		Str("Deleter path", opts.DeleterPath).
		Bool("Deleter hide window", opts.DeleterHideWindow).
		Msg("Run called")

	if opts.Verbose {
		e.Runtime.SetLogger(
			e.Runtime.Logger().EnableDebug(),
		)
	}
	// -----------------------*/
	// deleter起動
	// -----------------------*/
	if err := domain.LaunchDeleter(e.Runtime, opts); err != nil {
		return err
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
	configFile, program, pArgs, finalEnv, allConfigs, err := domain.ResolveRunOptions(opts, tagData, e.Runtime)
	if err != nil {
		return err // 循環参照などのエラーがあれば、ここで即座に終了
	}
	if program == "" {
		return errors.New("no program specified")
	}

	// #20: 環境変数の上書き追跡・表示
	tracker := types.TrackEnvOverrides(allConfigs)
	if err := tracker.PrintOverrides(e.Runtime.Logger(), opts.DuplicateStrict, opts.DuplicateNoWarn); err != nil {
		return err
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
		e.Runtime.Logger().Error().Err(err).Str("prefix", tmpPrefix).Msg("failed to create temp file")
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	e.Runtime.Logger().Info().Str("tempFile", tmpPath).Msg("created temp file")

	// manifest 書き込み
	manifest := types.Manifest{
		TempFilePath: tmpPath,
		EECPID:       e.Executor().Getpid(),
	}
	if _, err := manifest.WriteToManifest(e.Runtime); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// ----------------------*/
	// Start process
	// -----------------------*/
	var childPid int
	cmd, err := e.Executor().StartProcess(program, pArgs, finalEnv,
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
		return fmt.Errorf("failed to encode temp data: %w", err)
	}
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to flush temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	e.Runtime.Logger().Info().
		Int("ParentPID", tempData.ParentPID).
		Int("ChildPID", tempData.ChildPID).
		Str("ConfigFile", tempData.ConfigFile).
		Str("Program", tempData.Program).
		Strs("ProgramArgs", tempData.ProgramArgs).
		Msg("temp file written")

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
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Runtime.Logger().Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		return fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, types.DEFAULT_TAG_DIR)
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
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Runtime.Logger().Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		return fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, types.DEFAULT_TAG_DIR)
	tagPath := filepath.Join(tagDir, fmt.Sprintf("%s.tag", tagName))
	err = e.FS().Remove(tagPath)
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

// Tree は指定されたタグ名に基づき、関連する設定ファイルや依存構造をツリー表示する。
func (e *Engine) Tree(tagName string) error {
	data, err := types.ReadTagData(e.Runtime, tagName)
	if err != nil {
		return fmt.Errorf("failed to read tag data for %s: %w", tagName, err)
	}

	fmt.Printf("Dependency tree for tag: %s\n", tagName)
	visited := make(map[string]bool)

	// タグ自身のConfigFileを起点に展開
	if data.ConfigFile != "" {
		if err := e.printConfigTree(data.ConfigFile, "", visited); err != nil {
			return err
		}
	}

	// ImportConfigFiles を展開
	for _, imp := range data.ImportConfigFiles {
		// タグかファイルか判定
		if isConfigFile(imp) {
			fmt.Printf("└── Imported file: %s\n", filepath.Base(imp))
			if err := e.printConfigTree(imp, "    ", visited); err != nil {
				return err
			}
		} else {
			// タグ名として存在するか確認
			_, err := types.ReadTagData(e.Runtime, imp)
			if err == nil {
				fmt.Printf("└── Imported tag: %s\n", imp)
				// 再帰的にツリーを出す
				if err := e.Tree(imp); err != nil {
					return err
				}
			} else {
				// タグでもファイルでもない場合
				fmt.Printf("└── Unknown import: %s\n", imp)
			}
		}
	}

	data2, _ := types.ReadTagData(e.Runtime, tagName)
	_, _, _, _, allConfigs, err2 := domain.ResolveRunOptions(types.RunOptions{Tag: tagName}, data2, e.Runtime)
	if err2 == nil {
		tracker := types.TrackEnvOverrides(allConfigs)
		tracker.PrintOverrides(e.Runtime.Logger(), false, false)
	}
	return nil
}

// ファイル拡張子で設定ファイルか判定
func isConfigFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".toml", ".yaml", ".yml", ".json", ".env":
		return true
	default:
		return false
	}
}

func (e *Engine) printConfigTree(filePath string, prefix string, visited map[string]bool) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to resolve path %s: %w", filePath, err)
	}

	if visited[absPath] {
		fmt.Println(prefix + filepath.Base(filePath) + " (already visited)")
		return nil
	}
	visited[absPath] = true

	fmt.Println(prefix + filepath.Base(filePath))

	config, err := types.ReadConfig(e.Runtime, absPath)
	if err != nil {
		e.Runtime.Logger().Warn().Err(err).Str("config", absPath).Msg("config読み込み失敗（スキップ）")
		return nil
	}

	// configs[] を出力
	for i, meta := range config.Configs {
		last := (i == len(config.Configs)-1) && len(config.Envs) == 0 && config.Program.Path == ""
		var branch string
		if last {
			branch = "└── "
		} else {
			branch = "├── "
		}

		desc := meta.Description
		if desc == "" {
			desc = "(no description)"
		}
		fmt.Println(prefix + branch + fmt.Sprintf("Config: %s  [sep='%s']", desc, meta.Separator))
	}

	// envs[] を出力
	for i, env := range config.Envs {
		last := (i == len(config.Envs)-1) && config.Program.Path == ""
		var branch, nextPrefix string
		if last {
			branch = "└── "
			nextPrefix = prefix + "    "
		} else {
			branch = "├── "
			nextPrefix = prefix + "│   "
		}
		fmt.Println(prefix + branch + fmt.Sprintf("Env: %s", env.Key))

		// .envファイル参照ならさらにツリー展開
		if val, ok := env.Value.(string); ok && strings.HasSuffix(val, ".env") {
			if err := e.printConfigTree(val, nextPrefix, visited); err != nil {
				return err
			}
		}
	}

	// program.path が存在すれば、それもノードとして表示
	if config.Program.Path != "" {
		fmt.Println(prefix + "└── " + fmt.Sprintf("Program: %s", config.Program.Path))
	}

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
