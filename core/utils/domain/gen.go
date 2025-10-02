package domain

import (
	"strings"
	//"github.com/rs/zerolog/log"
	"github.com/m0090-dev/eec-go/core/utils/general"
	"github.com/m0090-dev/eec-go/core/types"
	"github.com/m0090-dev/eec-go/core/interfaces"
	//"os"
	"path/filepath"
	"runtime"
)

const (
	windowsWrapEECScript = `@echo off
chcp 65001
REM set "eec_deleter=D:\win\program\go\main-project\eec\build\eec-deleter"
REM set "eec_exe=D:\win\program\go\main-project\eec\build\eec"
set "PATH=D:\win\program\go\main-project\eec\build\;%PATH%"
set "eec_deleter=eec-deleter"
set "eec_exe=eec.exe"

tasklist /FI "IMAGENAME eq %eec_deleter%" /NH | find /I "%eec_deleter%" >nul

if "%1"=="run" (
    rem エラーレベルが 0（すでに実行中）か確認
    if %ERRORLEVEL% equ 0 (
        echo [%eec_deleter%] は既に実行中です。
    ) else (
        echo [%eec_deleter%] を起動します…
        powershell -WindowStyle Normal -Command "Start-Process -FilePath '%eec_deleter%' -WindowStyle Hidden"
    )
)
%eec_exe% %*
`

	windowsWrapGEECScript = `@echo off
	chcp 65001
	set "PATH=D:\win\program\go\main-project\eec\build\;%PATH%"
	geec.exe
`

	
	// Windows batch script (for cmd.exe)
	windowsTagUtilsScript = `@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM Use the first argument as --program
set PROGRAM=%1
shift

REM Concatenate the remaining arguments with commas for --program-args
set ARGS=
:loop
if "%~1"=="" goto run
if defined ARGS (
  set ARGS=!ARGS!,%~1
) else (
  set ARGS=%~1
)
shift
goto loop

:run
eec run --deleter-hide-window --tag %TAGNAME% --program %PROGRAM% --program-args=!ARGS!
`

	windowsTagUtilsScriptProgram = `@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM Use the first argument as --program
set PROGRAM=%1
shift

REM Concatenate the remaining arguments with commas for --program-args
set ARGS=
:loop
if "%~1"=="" goto run
if defined ARGS (
  set ARGS=!ARGS!,%~1
) else (
  set ARGS=%~1
)
shift
goto loop

:run
eec run --deleter-hide-window --tag %TAGNAME% --program %PROGRAM% 
`


	windowsTagUtilsScriptProgramArgs = `@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM Use the first argument as --program
set PROGRAM=%1
shift

REM Concatenate the remaining arguments with commas for --program-args
set ARGS=
:loop
if "%~1"=="" goto run
if defined ARGS (
  set ARGS=!ARGS!,%~1
) else (
  set ARGS=%~1
)
shift
goto loop

:run
eec run --deleter-hide-window --tag %TAGNAME% --program-args=!ARGS!
`


	windowsTagSimpleUtilsScript = `@echo off
setlocal enabledelayedexpansion

REM Use the first argument as --program
set PROGRAM=%1
shift

REM Concatenate the remaining arguments with commas for --program-args
set ARGS=
:loop
if "%~1"=="" goto run
if defined ARGS (
  set ARGS=!ARGS!,%~1
) else (
  set ARGS=%~1
)
shift
goto loop

:run
eec run --tag %TAGNAME%
`
	unixTagUtilsScript = ``
	unixTagSimpleUtilsScript = ``
	unixWrapEECScript = `#!/bin/bash
# UTF-8 前提

# ビルドディレクトリを PATH に追加
BUILD_DIR="/mnt/d/win/program/go/main-project/eec/build"
export PATH="$BUILD_DIR:$PATH"

# 実行ファイル名
eec_deleter="eec-deleter"
eec_exe="eec"

# 第一引数が "run" の場合は deleter を起動
if [[ "$1" == "run" ]]; then
    # プロセスが既に動いているか確認
    if pgrep -x "$eec_deleter" > /dev/null; then
        echo "[$eec_deleter] は既に実行中です。"
    else
        echo "[$eec_deleter] を起動します…"
        # バックグラウンドで起動して出力を捨てる
        nohup "$eec_deleter" >/dev/null 2>&1 &
    fi
fi

# eec 実行、引数をそのまま渡す
"$eec_exe" "$@"
`


)
func toWindowsLineEndings(s string) string {
    s = strings.ReplaceAll(s, "\r\n", "\n")  // まずCRLFをLFに統一
    s = strings.ReplaceAll(s, "\r", "\n")    // 万一CRのみがあればLFに変換
    return strings.ReplaceAll(s, "\n", "\r\n")
}

func GenWindowsTagUtilsScript(tagName string) string {
    script := strings.ReplaceAll(windowsTagUtilsScript, "%TAGNAME%", tagName)
    return toWindowsLineEndings(script)
}

func GenWindowsSimpleTagUtilsScript(tagName string) string {
    script := strings.ReplaceAll(windowsTagSimpleUtilsScript, "%TAGNAME%", tagName)
    return toWindowsLineEndings(script)
}

func GenWindowsTagUtilsScriptProgram(tagName string) string {
    script := strings.ReplaceAll(windowsTagUtilsScriptProgram, "%TAGNAME%", tagName)
    return toWindowsLineEndings(script)
}
func GenWindowsTagUtilsScriptProgramArgs(tagName string) string {
    script := strings.ReplaceAll(windowsTagUtilsScriptProgramArgs, "%TAGNAME%", tagName)
    return toWindowsLineEndings(script)
}




func GenWindowsWrapScript() string {
    return toWindowsLineEndings(windowsWrapEECScript)
}
func GenWindowsGUIWrapScript() string {
    return toWindowsLineEndings(windowsWrapGEECScript)
}

func GenUnixTagUtilsScript(tagName string) string {
	return ""
}

func GenUnixSimpleTagUtilsScript(tagName string) string {
	return ""
}

func GenUnixWrapScript() string {
	return unixWrapEECScript
}


/*func GenWrapScript(os ext.OS,logger ext.Logger) {*/
	/*scriptDir := ext.DEFAULT_SCRIPT_DIR*/
	/*var wrapScriptContent string*/
	/*var wrapScriptFileName string*/
	/*var wrapScriptFile string*/
	/*baseName := "eec"*/
	/*if runtime.GOOS == "windows" {*/
		/*wrapScriptContent = GenWindowsWrapScript()*/
		/*wrapScriptFileName = general.AddExtension(baseName, ".bat")*/
	/*} else {*/
		/*wrapScriptContent = GenUnixWrapScript()*/
		/*wrapScriptFileName  = baseName*/
	/*}*/
	/*wrapScriptFile = filepath.Join(scriptDir, wrapScriptFileName)*/
	/*// ディレクトリがなければ作成*/
	/*if err := os.FS.MkdirAll(scriptDir, 0755); err != nil {*/
		/*logger.Error().Err(err).Msg("Failed to create utils script directory")*/
		/*return*/
	/*}*/
	/*file, err := os.FS.Create(wrapScriptFile)*/
	/*if err != nil {*/
		/*logger.Error().Err(err).Str("file", wrapScriptFile).Msg("Failed to create file")*/
		/*return*/
	/*}*/

	/*func() {*/
		/*defer file.Close()*/
		/*_, err := file.WriteString(wrapScriptContent)*/
		/*if err != nil {*/
			/*return*/
		/*}*/
	/*}()*/

/*}*/

func GenWrapScript(os types.OS, logger interfaces.Logger) {
	scriptDir := types.DEFAULT_SCRIPT_DIR
	baseName := "eec"
	guiBaseName := "geec"
	// ディレクトリがなければ作成
	if err := os.FS.MkdirAll(scriptDir, 0755); err != nil {
		logger.Error().Err(err).Msg("Failed to create utils script directory")
		return
	}

	// --- Windows用スクリプト ---
	windowsScriptContent := GenWindowsWrapScript()
	windowsScriptFileName := general.AddExtension(baseName, ".bat")
	windowsScriptFile := filepath.Join(scriptDir, windowsScriptFileName)

	func() {
		file, err := os.FS.Create(windowsScriptFile)
		if err != nil {
			logger.Error().Err(err).Str("file", windowsScriptFile).Msg("Failed to create Windows wrap script")
			return
		}
		defer file.Close()
		if _, err := file.WriteString(windowsScriptContent); err != nil {
			logger.Error().Err(err).Str("file", windowsScriptFile).Msg("Failed to write Windows wrap script")
		}
	}()

	// --- Windows用スクリプト ---
	windowsGUIScriptContent := GenWindowsGUIWrapScript()
	windowsGUIScriptFileName := general.AddExtension(guiBaseName, ".bat")
	windowsGUIScriptFile := filepath.Join(scriptDir, windowsGUIScriptFileName)

	func() {
		file, err := os.FS.Create(windowsGUIScriptFile)
		if err != nil {
			logger.Error().Err(err).Str("file", windowsGUIScriptFile).Msg("Failed to create Windows wrap script")
			return
		}
		defer file.Close()
		if _, err := file.WriteString(windowsGUIScriptContent); err != nil {
			logger.Error().Err(err).Str("file", windowsGUIScriptFile).Msg("Failed to write Windows wrap script")
		}
	}()







	// --- Unix用スクリプト ---
	unixScriptContent := GenUnixWrapScript()
	unixScriptFileName := baseName
	unixScriptFile := filepath.Join(scriptDir, unixScriptFileName)

	func() {
		file, err := os.FS.Create(unixScriptFile)
		if err != nil {
			logger.Error().Err(err).Str("file", unixScriptFile).Msg("Failed to create Unix wrap script")
			return
		}
		defer file.Close()
		if _, err := file.WriteString(unixScriptContent); err != nil {
			logger.Error().Err(err).Str("file", unixScriptFile).Msg("Failed to write Unix wrap script")
		}
	}()
}

func GenUtilsScript(os types.OS,logger interfaces.Logger) {
	homeDir, _ := os.Env.UserHomeDir()
	if homeDir == "" {
		return
	}
	tagDir := filepath.Join(homeDir, types.DEFAULT_TAG_DIR)
	tagFileLists, _ := general.GetFilesWithExtension(tagDir, ".tag")
	tagNameLists := general.RemoveExtensions(general.BaseSlice(tagFileLists))
	logger.Debug().
		Str("tagNameLists", strings.Join(tagNameLists, ",")).Msg("")

	for _, name := range tagNameLists {
		baseName := "t" + name
		logger.Debug().
			Str("tagName", name).Msg("")

		tagData, err := types.ReadTagData(os,logger,name)
		if err != nil {
			logger.Error().Err(err).Str("tag", name).Msg("Failed to read tag data")
			continue
		}

		
		configFile := tagData.ConfigFile
		var config types.Config
		if configFile != "" && general.FileExists(configFile) {
			config, err = types.ReadConfig(os,logger,configFile)
			if err != nil {
				logger.Error().Err(err).Str("configFile", configFile).Msg("Failed to read config file")
			}
		}

		var tagUtilsScriptContent string
		var tagUtilsScriptFileName string

		if runtime.GOOS == "windows" {
			hasProgram := tagData.Program != "" || config.Program.Path != ""
			hasProgramArgs := len(tagData.ProgramArgs) != 0 || len(config.Program.Args) != 0

			switch {
			case hasProgram:
				tagUtilsScriptContent = GenWindowsSimpleTagUtilsScript(name)

			case !hasProgram && hasProgramArgs:
				tagUtilsScriptContent = GenWindowsTagUtilsScriptProgramArgs(name)

			case !hasProgram && !hasProgramArgs:
				if len(config.Program.Args) != 0 {
					tagUtilsScriptContent = GenWindowsSimpleTagUtilsScript(name)
				} else {
					tagUtilsScriptContent = GenWindowsTagUtilsScriptProgram(name)
				}

			default:
				tagUtilsScriptContent = GenWindowsSimpleTagUtilsScript(name)
			}

			tagUtilsScriptFileName = general.AddExtension(baseName, ".bat")

		} else {
			tagUtilsScriptContent = GenUnixTagUtilsScript(name)
			tagUtilsScriptFileName = general.AddExtension(baseName, ".sh")
		}

		logger.Debug().
			Str("tagUtilsScriptContent", tagUtilsScriptContent).
			Str("tagUtilsScriptFileName", tagUtilsScriptFileName).
			Msg("")

		scriptDir := filepath.Join(types.DEFAULT_SCRIPT_DIR, types.DEFAULT_UTILS_SCRIPT_DIR)
		tagUtilsScriptFile := filepath.Join(scriptDir, tagUtilsScriptFileName)
		logger.Debug().
			Str("tagUtilsScriptFile", tagUtilsScriptFile).
			Msg("")

		if err := os.FS.MkdirAll(scriptDir, 0755); err != nil {
			logger.Error().Err(err).Msg("Failed to create utils script directory")
			return
		}

		file, err := os.FS.Create(tagUtilsScriptFile)
		if err != nil {
			logger.Error().Err(err).Str("file", tagUtilsScriptFile).Msg("Failed to create file")
			return
		}

		func() {
			defer file.Close()
			_, err := file.WriteString(tagUtilsScriptContent)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to write to file")
			}
		}()
	}
}


