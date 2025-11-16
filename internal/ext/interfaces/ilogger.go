package interfaces
import "io"

// Event はログビルダー（チェーンでフィールド追加して最後に Msg() を呼ぶ）を表す。
type Event interface {
	Str(key, val string) Event
	Strs(key string, vals []string) Event
	Int(key string, val int) Event
	Bool(key string, val bool) Event
	Err(err error) Event
	Interface(key string, v interface{}) Event

	Msg(msg string)
	Msgf(format string, args ...interface{})
}

// Level は抽象化したログレベル
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

// Logger インターフェース（zerolog の型を一切含まない）
type Logger interface {
	Debug() Event
	Info() Event
	Warn() Event
	Error() Event
	Fatal() Event
	Panic() Event

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	Level(l Level) Logger
	Output(w io.Writer) Logger
}


