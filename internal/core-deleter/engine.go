package core_deleter
import (
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"bufio"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

)
// ====================
// プロセスが終了するのを待機する関数
// ====================
func waitForProcessTermination(pid int) error {
	for {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
		default:
			cmd = exec.Command("ps", "-p", strconv.Itoa(pid))
		}

		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to check process: %w", err)
		}

		if strings.Contains(string(output), strconv.Itoa(pid)) {
			time.Sleep(3 * time.Second)
		} else {
			break
		}
	}
	return nil
}

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


func (e *Engine) Run() error{
    tempDir := os.TempDir()
    manifestPath := filepath.Join(tempDir, "eec_manifest.txt")

    for {
        if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
            log.Info().Msg("Manifest does not exist. Nothing to clean.")
            time.Sleep(3 * time.Second)
            continue
        }

        file, err := os.Open(manifestPath)
        if err != nil {
            log.Error().Err(err).Msg("Failed to open manifest")
            time.Sleep(3 * time.Second)
            continue
        }

        scanner := bufio.NewScanner(file)
        var newLines []string

        for scanner.Scan() {
            line := scanner.Text()
            parts := strings.Fields(line)
            if len(parts) != 2 {
                log.Error().Str("line", line).Msg("Invalid line in manifest")
                continue
            }

            tempFilePath := parts[0]
            pid, _ := strconv.Atoi(parts[1])

            // PID が存在すれば待機
            if pid > 0 {
                if err := waitForProcessTermination(pid); err != nil {
                    log.Error().Err(err).Int("pid", pid).Msg("Failed waiting for process")
                }
            }

            // 一時ファイルが残っていれば削除
            if _, err := os.Stat(tempFilePath); err == nil {
                if err := os.Remove(tempFilePath); err != nil {
                    log.Error().Err(err).Str("tempFilePath", tempFilePath).Msg("Failed to delete temp file")
                    // 削除失敗した行は manifest に残す
                    newLines = append(newLines, line)
                } else {
                    log.Info().Str("tempFilePath", tempFilePath).Msg("Deleted temp file")
                }
            } else {
                // ファイルが無い場合は行を manifest から削除（もう不要）
                log.Info().Str("tempFilePath", tempFilePath).Msg("Temp file already removed")
            }
        }

        file.Close()

        // manifest の更新
        if len(newLines) == 0 {
            if err := os.Remove(manifestPath); err != nil {
                log.Error().Err(err).Msg("Failed to delete manifest file")
            } else {
                log.Info().Msg("Deleted manifest file")
            }
            break // 全て処理済みならループ終了
        } else {
            // 新しい内容で manifest を上書き
            os.WriteFile(manifestPath, []byte(strings.Join(newLines, "\n")), 0644)
            log.Info().Msg("Updated manifest with remaining entries")
        }

        time.Sleep(5 * time.Second)
    }
 appID := "eec-deleter"
    title := "完了メッセージ"
    message := "一時ファイル等の削除が完了しました"

if err := SendNotification(appID, title, message); err != nil {
    return err
}
	return nil
}
