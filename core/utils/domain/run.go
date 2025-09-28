package domain

import (
	"os/exec"
	"runtime"
	"strings"
	"fmt"
	"github.com/m0090-dev/eec-go/core/ext"
	"github.com/m0090-dev/eec-go/core/utils/general"
)

// ReadOrFallback is the same helper behavior as original core.
func ReadOrFallback(os ext.OS,logger ext.Logger,name string) (ext.Config, error) {
	var cfg ext.Config
	if os.FS.FileExists(name) {
		return ext.ReadConfig(os,logger,name)
	}
	tagData, err := ext.ReadTagData(os,logger,name)
	if err != nil {
		return cfg, err
	}
	for _, f := range tagData.ImportConfigFiles {
		var fcfg ext.Config
		if general.FileExists(f) {
			fcfg, _ = ext.ReadConfig(os,logger,f)
		} else {
			fcfg, _ = ext.ReadInlineConfig(os,logger,f)
		}
		fcfg.ApplyEnvs(os,logger)
		cfg = fcfg
	}
	return cfg, nil
}

func IsProcessRunning(os ext.OS,logger ext.Logger,name string) (bool, error) {
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
