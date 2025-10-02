package types
import (
    "fmt"
    "os"
    "path/filepath"
)
type Manifest struct {
	TempFilePath string
	EECPID	int
}

func (m *Manifest) WriteToManifest() (string, error) {
    manifestDir := filepath.Dir(m.TempFilePath)
    manifestPath := filepath.Join(manifestDir, DEFAULT_MANIFEST_FILE_NAME + ".txt")
	
    // 追記モードで開く（存在しなければ作成）
    file, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return "", err
    }
    defer file.Close()

    // 一時ファイルのパスと eecPID を追記
    if _, err = fmt.Fprintf(file, "%s %d\n", m.TempFilePath, m.EECPID); err != nil {
        return "", err
    }

    return manifestPath, nil
}
