package ext

import (
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"

	//"os"
	"path/filepath"
	"strings"

	"github.com/m0090-dev/eec-go/core/utils/general"
)

type TagData struct {
	ConfigFile  string
	Program     string
	ProgramArgs []string
	ImportConfigFiles []string
	//Description string TODO: 要追加検討
}

// ---------------------------
// TagData バイナリ保存処理
// ---------------------------
func (t *TagData) Write(os OS,logger Logger,tagName string) error {
	homeDir, err := os.Env.UserHomeDir()
	if homeDir == "" {
		return err
	}
	dir := filepath.Join(homeDir, DEFAULT_TAG_DIR)
	if err := os.FS.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tagPath := filepath.Join(dir, fmt.Sprintf("%s.tag", tagName))

	var buf bytes.Buffer

	// 各フィールドを手動で書き込む
	if err := general.WriteString(&buf, t.ConfigFile); err != nil {
		return err
	}
	if err := general.WriteString(&buf, t.Program); err != nil {
		return err
	}
	if err := general.WriteStringSlice(&buf, t.ProgramArgs); err != nil {
		return err
	}
	if err := general.WriteStringSlice(&buf, t.ImportConfigFiles); err != nil {
		return err
	}

	if err := os.FS.WriteFile(tagPath, buf.Bytes(), 0644); err != nil {
		return err
	}

	log.Info().
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
func ReadTagData(os OS,logger Logger,tagName string) (TagData, error) {
	homeDir, _ := os.Env.UserHomeDir()
	if homeDir == "" {
		return TagData{}, fmt.Errorf(fmt.Sprintf("%s not set", homeDir))
	}
	tagPath := filepath.Join(homeDir, ".eec", fmt.Sprintf("%s.tag", tagName))

	content, err := os.FS.ReadFile(tagPath)
	if err != nil {
		return TagData{}, err
	}

	buf := bytes.NewReader(content)

	var data TagData
	if data.ConfigFile, err = general.ReadString(buf); err != nil {
		return TagData{}, err
	}
	if data.Program, err = general.ReadString(buf); err != nil {
		return TagData{}, err
	}
	if data.ProgramArgs, err = general.ReadStringSlice(buf); err != nil {
		return TagData{}, err
	}
	
	if data.ImportConfigFiles, err = general.ReadStringSlice(buf); err != nil {
		return TagData{}, err
	}

	log.Info().
		Str("ConfigFile", data.ConfigFile).
		Str("Program", data.Program).
		Str("Args", strings.Join(data.ProgramArgs, " ")).
		Str("Import config files", strings.Join(data.ImportConfigFiles, ", ")).
		Msg("TagData read successfully")

	return data, nil
}

