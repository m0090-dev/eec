package general

import (
	"os"
	"path/filepath"
	"strings"
	"regexp"
)

/*
func FileExists(path string) bool {
    _, err := os.Stat(path)
    return err == nil || !os.IsNotExist(err)
}
*/

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
func FileExt(path string) string {
	return filepath.Ext(path)
}
func FileBase(path string) string{
	return filepath.Base(path)
}

// AddExtension はファイル名に拡張子を付けます。
// すでに拡張子が付いている場合は何もしません。
// extは「.txt」のようにドットから始めてください。
func AddExtension(filename, ext string) string {
	if ext == "" {
		return filename
	}
	if strings.EqualFold(filepath.Ext(filename), ext) {
		return filename
	}
	return filename + ext
}

// extractFileNameSafe は
// 入力文字列からstartで始まり、extで終わる部分を抽出し、
// 拡張子の検証もfilepath.Extで行う安全版です。
// 見つからなければ空文字を返します。
func ExtractFileNameSafe(input, start, ext string) string {
	// 拡張子は小文字化して統一しておく
	ext = strings.ToLower(ext)

	// startからextまでの部分を抽出する正規表現（非貪欲）
	pattern := `(` + regexp.QuoteMeta(start) + `.*?` + regexp.QuoteMeta(ext) + `)`
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(input, -1)
	for _, m := range matches {
		// filepath.Extで実際の拡張子を検証（大文字小文字区別なし）
		if strings.ToLower(filepath.Ext(m)) == ext {
			return m
		}
	}
	return ""
}

// RemoveExtension は、ファイル名などの文字列から拡張子（.xxx）を取り除いて返します。
// 拡張子がなければ元の文字列をそのまま返します。
func RemoveExtension(name string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name
	}
	return strings.TrimSuffix(name, ext)
}
// RemoveExtensions は文字列スライスに対し、各要素から拡張子を取り除いたスライスを返します。
func RemoveExtensions(names []string) []string {
	result := make([]string, len(names))
	for i, name := range names {
		result[i] = RemoveExtension(name)
	}
	return result
}

// BaseSlice は文字列スライスの各要素からファイル名部分だけを抽出します。
func BaseSlice(paths []string) []string {
    res := make([]string, len(paths))
    for i, p := range paths {
        res[i] = filepath.Base(p)
    }
    return res
}


// ---------------------------
// 指定ディレクトリ内の特定拡張子ファイルを取得する関数
// ---------------------------
func GetFilesWithExtension(dir string, ext string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.EqualFold(filepath.Ext(entry.Name()), ext) {
			fullPath := filepath.Join(dir, entry.Name())
			files = append(files, fullPath)
		}
	}

	return files, nil
}

