package domain

import (
	"syscall"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"path/filepath"
	"github.com/m0090-dev/eec-go/core/interfaces"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/m0090-dev/eec-go/core/utils/general"
	gos "os"
)

// ReadOrFallback is the same helper behavior as original core.
func ReadOrFallback(opts types.RunOptions,os types.OS,logger interfaces.Logger,name string) (types.Config, error) {
	var cfg types.Config
	if os.FS.FileExists(name) {
		return types.ReadConfig(os,logger,name)
	}
	tagData, err := types.ReadTagData(os,logger,name)
	if err != nil {
		return cfg, err
	}
	for _, f := range tagData.ImportConfigFiles {
		var fcfg types.Config
		if general.FileExists(f) {
			fcfg, _ = types.ReadConfig(os,logger,f)
		} else {
			fcfg, _ = types.ReadInlineConfig(os,logger,f)
		}
		fcfg.ApplyEnvs(os,logger,opts.Separator)
		cfg = fcfg
	}
	return cfg, nil
}

func ReadOrFallbackRecursive(opts types.RunOptions, os types.OS, logger interfaces.Logger, name string) (types.Config, error) {
    var cfg types.Config

    // 1. ファイルとして存在する場合はそのまま読み込む
    if os.FS.FileExists(name) {
        return types.ReadConfig(os, logger, name)
    }

    // 2. タグデータとして読み込む
    tagData, err := types.ReadTagData(os, logger, name)
    if err != nil {
        return cfg, err
    }

    // 3. import ファイルを再帰的に読み込む
    for _, f := range tagData.ImportConfigFiles {
        fcfg, err := ReadOrFallbackRecursive(opts, os, logger, f)
        if err != nil {
            logger.Warn().Str("import", f).Err(err).Msg("failed to read import config")
            continue
        }

        // 4. 読み込んだ import の env を適用
        fcfg.ApplyEnvs(os, logger, opts.Separator)

        // 5. cfg に集約
        cfg.Envs = append(cfg.Envs, fcfg.Envs...)

        // Program.Path は未設定なら設定
        if cfg.Program.Path == "" {
            cfg.Program.Path = fcfg.Program.Path
        }
        // Program.Args は append
        cfg.Program.Args = append(cfg.Program.Args, fcfg.Program.Args...)

        // Configs も append
        cfg.Configs = append(cfg.Configs, fcfg.Configs...)
    }

    return cfg, nil
}



func IsProcessRunning(os types.OS,logger interfaces.Logger,name string) (bool, error) {
	switch runtime.GOOS {
	case "windows":
		if !strings.HasSuffix(name,".exe"){
			name += ".exe"
		}
		// Windows: tasklist
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s", name))
		output, err := cmd.Output()
		if err != nil {
			return false, err
		}
		return strings.Contains(string(output), name), nil

	case "linux", "darwin":
		// Linux / macOS: pgrep -x
		cmd := exec.Command("pgrep", "-x", name)
		err := cmd.Run()
		if err == nil {
			return true, nil // exit code 0 → 実行中
		}
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return false, nil // exit code 1 → 見つからない
		}
		return false, err

	default:
		return false, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

func IsPIDRunning(os types.OS, logger interfaces.Logger, pid int) (bool, error) {
    if pid <= 0 {
        return false, fmt.Errorf("invalid PID: %d", pid)
    }

    proc, err := os.Executor.FindProcess(pid)
    if err != nil {
        // プロセスが見つからない場合は false,errorを返す
        return false, err
    }

    // プロセスに kill 0 シグナルを送る（Linux/macOS）など、存在確認
    if runtime.GOOS != "windows" {
        err = proc.Signal(syscall.Signal(0))
        if err == nil {
            return true, nil
        }
        if err == syscall.ESRCH {
            return false, nil
        }
        return false, err
    }

    // Windows は FindProcess が返れば存在とみなす
    return true, nil
}

func LaunchDeleter(os types.OS,logger interfaces.Logger,opts types.RunOptions) error {
	// ----------------------*/
	// deleter起動
	// ----------------------*/
	deleterPath := opts.DeleterPath
	deleterHideWindow := opts.DeleterHideWindow
	if deleterPath == "" || !os.FS.FileExists(deleterPath) {
		deleterPath = filepath.Join(types.DEFAULT_DELETER_EXECUTE_NAME)
	}

	running, err := IsProcessRunning(os,logger,types.DEFAULT_DELETER_EXECUTE_NAME)
	if err != nil {
		logger.Error().Err(err).Msg("failed to check process")
		return fmt.Errorf("failed to check process: %w", err)
	}

	if running {
		logger.Debug().Msgf("[%s] は既に実行中です", deleterPath)
	} else {
		logger.Debug().Msgf("[%s] を起動します...", deleterPath)
		var pid int
		var execCmd *exec.Cmd
		if runtime.GOOS == "windows" {
			var out, errOut *gos.File
			if !deleterHideWindow {
				out, errOut = os.Console.Stdout(), os.Console.Stderr()
			}
			execCmd, err = os.Executor.StartProcess(
				deleterPath, []string{}, os.Env.Environ(), nil, out, errOut, deleterHideWindow,
			)
			pid = execCmd.Process.Pid
		} else {
			var out, errOut *gos.File
			if !deleterHideWindow {
				out, errOut = os.Console.Stdout(), os.Console.Stderr()
			}
			execCmd, err = os.Executor.StartProcess(
				deleterPath, []string{}, os.Env.Environ(), nil, out, errOut, deleterHideWindow,
			)
			pid = execCmd.Process.Pid
		}
		if err != nil {
			logger.Error().Err(err).Msg("failed to start process")
			return fmt.Errorf("failed to start process: %w", err)
		}
		logger.Info().Msgf("deleter started (pid=%d)", pid)
	}
	return nil
}
