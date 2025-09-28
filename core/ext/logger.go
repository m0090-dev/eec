package ext

import (
	"fmt"
	"io"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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

// -------------------------
// zerolog をラップした実装
// -------------------------

type defaultEvent struct {
	e *zerolog.Event
}

func (ev *defaultEvent) Str(key, val string) Event       { ev.e.Str(key, val); return ev }
func (ev *defaultEvent) Strs(key string, vals []string) Event { ev.e.Strs(key, vals); return ev }
func (ev *defaultEvent) Int(key string, val int) Event   { ev.e.Int(key, val); return ev }
func (ev *defaultEvent) Bool(key string, val bool) Event { ev.e.Bool(key, val); return ev }
func (ev *defaultEvent) Err(err error) Event             { ev.e.Err(err); return ev }
func (ev *defaultEvent) Interface(key string, v interface{}) Event { ev.e.Interface(key, v); return ev }
func (ev *defaultEvent) Msg(msg string)                   { ev.e.Msg(msg) }
func (ev *defaultEvent) Msgf(format string, args ...interface{}) { ev.e.Msg(fmt.Sprintf(format, args...)) }

// DefaultLogger は zerolog.Logger を内部に持つ実装
type DefaultLogger struct {
	l zerolog.Logger
}

// コンストラクタ
func NewDefaultLogger() Logger {
	return &DefaultLogger{l: log.Logger}
}

// ---- ポインタレシーバで実装 ----
func (d *DefaultLogger) Debug() Event { return &defaultEvent{e: d.l.Debug()} }
func (d *DefaultLogger) Info() Event  { return &defaultEvent{e: d.l.Info()} }
func (d *DefaultLogger) Warn() Event  { return &defaultEvent{e: d.l.Warn()} }
func (d *DefaultLogger) Error() Event { return &defaultEvent{e: d.l.Error()} }
func (d *DefaultLogger) Fatal() Event { return &defaultEvent{e: d.l.Fatal()} }
func (d *DefaultLogger) Panic() Event { return &defaultEvent{e: d.l.Panic()} }

func (d *DefaultLogger) WithField(key string, value interface{}) Logger {
	ctx := d.l.With().Interface(key, value)
	return &DefaultLogger{l: ctx.Logger()}
}

func (d *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	ctx := d.l.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &DefaultLogger{l: ctx.Logger()}
}

func (d *DefaultLogger) Level(l Level) Logger {
	var zl zerolog.Level
	switch l {
	case LevelDebug:
		zl = zerolog.DebugLevel
	case LevelInfo:
		zl = zerolog.InfoLevel
	case LevelWarn:
		zl = zerolog.WarnLevel
	case LevelError:
		zl = zerolog.ErrorLevel
	case LevelFatal:
		zl = zerolog.FatalLevel
	case LevelPanic:
		zl = zerolog.PanicLevel
	default:
		zl = zerolog.InfoLevel
	}
	return &DefaultLogger{l: d.l.Level(zl)}
}

func (d *DefaultLogger) Output(w io.Writer) Logger {
	return &DefaultLogger{l: d.l.Output(w)}
}
