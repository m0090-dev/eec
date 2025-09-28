package core

import (
	"runtime"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/m0090-dev/eec-go/core/ext"
	"github.com/m0090-dev/eec-go/core/utils/domain"
	"github.com/m0090-dev/eec-go/core/utils/general"
	//"github.com/rs/zerolog/log"
	//"os"
	"path/filepath"
	"strings"
	"time"
)



// Engine is the core library entrypoint. It contains pluggable implementations
// for executing commands and file operations so CLI can inject mocks for tests.
type Engine struct {
	OS       ext.OS
	Logger   ext.Logger
}


func (e *Engine) FS() ext.FS { return e.OS.FS }
func (e *Engine) Env() ext.Env { return e.OS.Env }
func (e *Engine) Executor() ext.Executor{return e.OS.Executor}
func (e *Engine) CommandLine() ext.CommandLine{return e.OS.CommandLine}
func (e *Engine) Console() ext.Console{return e.OS.Console}


// NewEngine returns an Engine with sensible defaults (os-backed).

func NewEngine(os *ext.OS, logger ext.Logger) *Engine {
    if os == nil {
        temp := ext.NewOS()
        os = &temp
    }
    if logger == nil {
	logger = ext.NewDefaultLogger() 
    }
    return &Engine{
        OS:     *os,
        Logger: logger,
    }
}

// RunOptions contains all inputs that were previously taken from flags / tag file.
type RunOptions struct {
	ConfigFile  string
	Program     string
	ProgramArgs []string
	Tag         string
	Imports     []string
	// Timeout for waiting program; zero means wait indefinitely
	WaitTimeout time.Duration
	HideWindow  bool
}


// Run executes the previous run() logic but returns errors instead of os.Exit.
// It is CLI-agnostic: the CLI just constructs RunOptions and provides Engine.
func (e *Engine) Run(ctx context.Context, opts RunOptions) error {
	// -----------------------*/
	// 開始時環境変数表示*/
	// -----------------------*/
	{
	  envs := e.Env().Environ()
	  envStr := strings.Join(envs, ", ")
	  e.Logger.Debug().Str("Started envs", envStr).Msg("")
        }

	e.Logger.Debug().Str("configFile", opts.ConfigFile).Str("program", opts.Program).
		Strs("programArgs", opts.ProgramArgs).Str("tag", opts.Tag).Strs("imports", opts.Imports).Msg("Run called")

	// Read tag data if provided
	var tagData ext.TagData
	var err error
	if opts.Tag != "" {
		tagData, err = ext.ReadTagData(e.OS,e.Logger,opts.Tag)
		if err != nil {
			e.Logger.Error().Err(err).Str("tag", opts.Tag).Msg("failed to read tag")
			return fmt.Errorf("failed to read tag %s: %w", opts.Tag, err)
		}
	}

	// Resolve configFile / program / args with same precedence as original:
	configFile := opts.ConfigFile
	program := opts.Program
	pArgs := opts.ProgramArgs

	if opts.Tag != "" {
		if tagData.ConfigFile != "" {
			configFile = tagData.ConfigFile
		}
		if tagData.Program != "" {
			program = tagData.Program
		}
		if len(tagData.ProgramArgs) != 0 {
			pArgs = tagData.ProgramArgs
		}
	}

	// Flag override (already reflected by opts values); keep the same logic as original:
	// If opts.ConfigFile / Program / ProgramArgs provided they override previous values.
	if opts.ConfigFile != "" {
		configFile = opts.ConfigFile
	}
	if opts.Program != "" {
		program = opts.Program
	}
	if len(opts.ProgramArgs) != 0 {
		pArgs = opts.ProgramArgs
	}

	// Load main config if exists
	var config ext.Config
	if configFile != "" && e.FS().FileExists(configFile) {
		if config, err = ext.ReadConfig(e.OS,e.Logger,configFile); err != nil {
			e.Logger.Error().Err(err).Str("configFile", configFile).Msg("failed to read config")
			return fmt.Errorf("failed to read config %s: %w", configFile, err)
		}
	}

	// Fill from config if program missing
	if (opts.Tag == "" || tagData.Program == "") && config.Program.Path != "" && program == "" {
		program = config.Program.Path
	}
	if (opts.Tag == "" || len(tagData.ProgramArgs) == 0) && len(config.Program.Args) != 0 && len(pArgs) == 0 {
		pArgs = config.Program.Args
	}

	// build manifest/temp prefix
	selfProgram := e.CommandLine().Args()[0]
	tmpDir := e.FS().TempDir()
	if tmpDir == "" {
		tmpDir = e.FS().TempDir()
	}
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
	defer tmpFile.Close()
	e.Logger.Info().Str("tempFile", tmpPath).Msg("created temp file")

	manifest := ext.Manifest{
		TempFilePath: tmpFile.Name(),
		EECPID:       e.Executor().Getpid(),
	}
	manifestPath, err := manifest.WriteToManifest()
	if err != nil {
		e.Logger.Error().Err(err).Str("manifestPath", manifestPath).Msg("failed to write manifest")
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	e.Logger.Info().Str("manifest", manifestPath).Msg("created manifest")

	// Merge envs (imports -> tag imports -> main config)
	allConfigs := []ext.Config{}

	// imports from opts
	for _, imp := range opts.Imports {
		cfg, rerr := domain.ReadOrFallback(e.OS,e.Logger,imp)
		if rerr == nil {
			allConfigs = append(allConfigs, cfg)
		} else {
			// ignore missing import: keep behavior consistent with original
			e.Logger.Debug().Err(rerr).Str("import", imp).Msg("import skipped")
		}
	}

	// tag imports
	for _, imp := range tagData.ImportConfigFiles {
		cfg, rerr := domain.ReadOrFallback(e.OS,e.Logger,imp)
		if rerr == nil {
			allConfigs = append(allConfigs, cfg)
		} else {
			e.Logger.Debug().Err(rerr).Str("import", imp).Msg("tag import skipped")
		}
	}

	// main config last
	allConfigs = append(allConfigs, config)

	// start with current environ
	finalEnv := e.Env().Environ()
	for _, cfg := range allConfigs {
		finalEnv = cfg.BuildEnvs(e.OS,e.Logger,finalEnv)
	}

	// Debug prints similar to original
	for _, eStr := range finalEnv {
		if strings.HasPrefix(strings.ToUpper(eStr), "PATH=") {
			e.Logger.Debug().Str("Final PATH", eStr).Msg("path check")
		}
	}
	//e.Logger.Debug().Str("LookPath PATH", os.Getenv("PATH")).Msg("")

	// resolve executable
	if program == "" {
		return errors.New("no program specified")
	}
	/*
		progPath, err := e.Executor.LookPath(program)
		if err != nil {
			e.Logger.Error().Err(err).Str("program", program).Msg("executable not found")
			return fmt.Errorf("executable not found: %w", err)
		}
	*/

	// Start process
	childPid, proc, err := e.Executor().StartProcess(program, pArgs, finalEnv, e.Console().Stdin(), e.Console().Stdout(), e.Console().Stderr(),opts.HideWindow)
	if err != nil {
		e.Logger.Error().Err(err).Msg("failed to start process")
		return fmt.Errorf("failed to start process: %w", err)
	}
	e.Logger.Info().Int("PID", childPid).Msg("sub process started")

	// write tempData
	tempData := ext.TempData{
		ParentPID:   e.Executor().Getpid(),
		ChildPID:    childPid,
		ConfigFile:  configFile,
		Program:     program,
		ProgramArgs: pArgs,
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(tempData); err != nil {
		e.Logger.Error().Err(err).Msg("failed to encode temp data")
		// attempt to stop child process if possible
		_ = proc.Kill()
		return fmt.Errorf("failed to encode temp data: %w", err)
	}
	if _, err := tmpFile.Write(buf.Bytes()); err != nil {
		e.Logger.Error().Err(err).Msg("failed to write temp file")
		_ = proc.Kill()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	e.Logger.Info().
		Int("ParentPID", tempData.ParentPID).
		Int("ChildPID", tempData.ChildPID).
		Str("ConfigFile", tempData.ConfigFile).
		Str("Program", tempData.Program).
		Strs("ProgramArgs", tempData.ProgramArgs).
		Msg("temp file written")

	// Wait for process (with optional timeout)
	if err := e.Executor().WaitProcess(proc, opts.WaitTimeout); err != nil {
		e.Logger.Error().Err(err).Msg("process finished with error or wait failed")
		return fmt.Errorf("process wait error: %w", err)
	}
	// -----------------------*/
	// 終了時環境変数表示*/
	// -----------------------*/
	{
	  envs := e.Env().Environ()
	  envStr := strings.Join(envs, ", ")
	  e.Logger.Debug().Str("Finished envs", envStr).Msg("")
        }

	e.Logger.Info().Msg("process finished normally")
	return nil
}

// ----------------- Stubs for other command core behaviors ------------------

// Gen performs generator-related core work (placeholder).
func (e *Engine) GenScript() error {
	domain.GenUtilsScript(e.OS,e.Logger)
	domain.GenWrapScript(e.OS,e.Logger)
	return nil
}

// Info returns structured information about the environment or config.
func (e *Engine) Info() error {
	infos := []string{}
	infos = append(infos, fmt.Sprintf("version=%s",ext.VERSION))
	infos = append(infos, fmt.Sprintf("pid=%d", e.Executor().Getpid()))
	infos = append(infos, fmt.Sprintf("goOS=%s", runtime.GOOS))

	e.Logger.Info().Strs("infos", infos).Msg("eec Info messages")
	return nil
}
// Restart attempts to restart a child process based on manifest/temp file.
// This is a suggestion-level implementation: locate manifest, read temp data, kill old, start new.
func (e *Engine) Restart(manifestPath string) {
	/* // Basic stub: caller should implement robust restart flow.*/
	/*if manifestPath == "" {*/
	/*return errors.New("manifestPath required")*/
	/*}*/
	/*m, err := meta.ReadManifest(manifestPath)*/
	/*if err != nil {*/
	/*return err*/
	/*}*/
	/*// read temp file*/
	/*f, err := os.Open(m.TempFilePath)*/
	/*if err != nil {*/
	/*return err*/
	/*}*/
	/*defer f.Close()*/
	/*var td meta.TempData*/
	/*if err := gob.NewDecoder(f).Decode(&td); err != nil {*/
	/*return err*/
	/*}*/
	/*// try to kill old child if exists*/
	/*if td.ChildPID != 0 {*/
	/*if proc, err := os.FindProcess(td.ChildPID); err == nil {*/
	/*_ = proc.Kill() // ignore error: best-effort*/
	/*}*/
	/*}*/
	/*// start new process using previous config*/
	/*opts := RunOptions{*/
	/*ConfigFile:  td.ConfigFile,*/
	/*Program:     td.Program,*/
	/*ProgramArgs: td.ProgramArgs,*/
	/*}*/
	/*// no timeout*/
	/*return e.Run(context.Background(), opts)*/
}

// Tag-related core functions (create, list, delete).
func (e *Engine) TagAdd(name string, tag ext.TagData) error {
	tagName := name
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

	if err := tag.Write(e.OS,e.Logger,tagName); err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルの書き込みに失敗しました")
		fmt.Errorf("Failed to tag file")
	}
	fmt.Println("Tag added:", tagName)
	return nil
}
func (e *Engine) TagRead(name string) error {
	tagName := name
	data, err := ext.ReadTagData(e.OS,e.Logger,tagName)
	if err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルの読み込みに失敗しました")
		fmt.Errorf("Failed to tag read")
	}
	fmt.Printf("Tag: %s\n  Config: %s\n  Program: %s\n  Args: %v\n  Import config files: %v\n",
		tagName, data.ConfigFile, data.Program, data.ProgramArgs, data.ImportConfigFiles)
	return nil
}
func (e *Engine) TagList() error {
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Logger.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, ext.DEFAULT_TAG_DIR)
	fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")
	if err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルが見つかりませんでした")
		fmt.Errorf("Failed to tag list")
	}
	fmt.Printf("-- current tag lists  --\n")
	fmt.Printf("%s\n", strings.Join(fileLists, "\n"))
	return nil

}
func (e *Engine) TagRemove(name string) error {
	tagName := name
	homeDir, err := e.Env().UserHomeDir()
	if homeDir == "" {
		e.Logger.Error().Err(err).Msg(fmt.Sprintf("homeDir(%s)が設定されていません", homeDir))
		fmt.Errorf("Missing required homeDir")
	}
	tagDir := filepath.Join(homeDir, ext.DEFAULT_TAG_DIR)
	tagPath := filepath.Join(tagDir, fmt.Sprintf("%s.tag", tagName))
	err = e.FS().Remove(tagPath)
	if err != nil {
		e.Logger.Error().Err(err).Msg(fmt.Sprintf("タグ名: %sの削除に失敗しました", tagName))
		fmt.Errorf("Failed to tag remove")
	}
	fmt.Printf("タグ名: %sを削除しました\n", tagName)

	fileLists, err := general.GetFilesWithExtension(tagDir, ".tag")
	if err != nil {
		e.Logger.Error().Err(err).Msg("タグファイルが見つかりませんでした")
		fmt.Errorf("Failed to tag list")
	}
	fmt.Printf("-- current tag lists  --\n")
	fmt.Printf("%s\n", strings.Join(fileLists, "\n"))
	return nil
}

