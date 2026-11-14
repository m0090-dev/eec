package types

import (
	"encoding/json"
	"fmt"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/rs/zerolog/log"
	"path/filepath"
	"strings"
)

type TagData struct {
	ConfigFile        string   `json:"config_file"`
	Program           string   `json:"program"`
	ProgramArgs       []string `json:"program_args"`
	ImportConfigFiles []string `json:"import_config_files"`
}

// ValidateTagName rejects tag names that could escape the tag
// directory once they are turned into a file path ("%s.tag").
// Empty names, path separators and ".." are not allowed.
func ValidateTagName(tagName string) error {
	if tagName == "" {
		return fmt.Errorf("tag name must not be empty")
	}
	if strings.Contains(tagName, "..") {
		return fmt.Errorf("invalid tag name %q: must not contain '..'", tagName)
	}
	if strings.ContainsAny(tagName, "/\\") {
		return fmt.Errorf("invalid tag name %q: must not contain path separators", tagName)
	}
	return nil
}

// ---------------------------
// TagData JSON 保存処理
// ---------------------------
func (t *TagData) Write(rt interfaces.Runtime, tagName string) error {
	if err := ValidateTagName(tagName); err != nil {
		return err
	}
	dir := ResolveDataDir(rt)
	if dir == "" {
		return fmt.Errorf("home dir not set")
	}
	if err := rt.FS().MkdirAll(dir, 0755); err != nil {
		return err
	}
	tagPath := filepath.Join(dir, fmt.Sprintf("%s.tag", tagName))

	// Store paths that live inside the data directory relative to it,
	// so the tag keeps working when the data directory moves (e.g. a
	// portable drive letter change). Work on a copy so the caller's
	// TagData keeps its original (absolute) paths.
	stored := *t
	stored.ConfigFile = RelativizeDataDirPath(rt, t.ConfigFile)
	if t.ImportConfigFiles != nil {
		stored.ImportConfigFiles = make([]string, len(t.ImportConfigFiles))
		for i, imp := range t.ImportConfigFiles {
			stored.ImportConfigFiles[i] = RelativizeDataDirPath(rt, imp)
		}
	}

	data, err := json.MarshalIndent(&stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tag data: %w", err)
	}

	if err := rt.FS().WriteFile(tagPath, data, 0644); err != nil {
		return err
	}

	log.Debug().
		Str("tagPath", tagPath).
		Str("ConfigFile", t.ConfigFile).
		Str("Program", t.Program).
		Str("Args", strings.Join(t.ProgramArgs, ", ")).
		Str("Import config files", strings.Join(t.ImportConfigFiles, ", ")).
		Msg("TagData written successfully")

	return nil
}

// --------------------------
// 読み取り処理
// --------------------------
func ReadTagData(rt interfaces.Runtime, tagName string) (TagData, error) {
	if err := ValidateTagName(tagName); err != nil {
		return TagData{}, err
	}
	dir := ResolveDataDir(rt)
	if dir == "" {
		return TagData{}, fmt.Errorf("home dir not set")
	}
	tagPath := filepath.Join(dir, fmt.Sprintf("%s.tag", tagName))

	content, err := rt.FS().ReadFile(tagPath)
	if err != nil {
		return TagData{}, err
	}

	// JSON を試みる（新形式）
	var data TagData
	if err := json.Unmarshal(content, &data); err != nil {
		// 旧バイナリ形式へのフォールバック（移行期間用）
		data, err = readTagDataLegacy(content)
		if err != nil {
			return TagData{}, fmt.Errorf("failed to parse tag file (tried JSON and legacy binary): %w", err)
		}
		// 旧形式だったので JSON に書き直して移行完了
		_ = data.Write(rt, tagName)
	}

	log.Debug().
		Str("ConfigFile", data.ConfigFile).
		Str("Program", data.Program).
		Str("Args", strings.Join(data.ProgramArgs, " ")).
		Str("Import config files", strings.Join(data.ImportConfigFiles, ", ")).
		Msg("TagData read successfully")

	return data, nil
}

// readTagDataLegacy は旧バイナリ形式（general.ReadString）で読み込む。
// 移行期間が終わったら削除してよい。
func readTagDataLegacy(content []byte) (TagData, error) {
	// import cycle を避けるため bytes.NewReader + general パッケージを直接使う。
	// ここでは bytes.Reader + binary.Read を使った簡易実装を用意する。
	// 実プロジェクトでは general.ReadString / ReadStringSlice をそのまま呼び出すこと。
	return TagData{}, fmt.Errorf("legacy binary format not supported; please re-register the tag")
}
