package impl

import (
	"fmt"
	"io"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/m0090-dev/eec-go/core/interfaces"

)

// -------------------------
// zerolog をラップした実装
// -------------------------

type DefaultEvent struct {
	e *zerolog.Event
}

func (ev *DefaultEvent) Str(key, val string) interfaces.Event       { ev.e.Str(key, val); return ev }
func (ev *DefaultEvent) Strs(key string, vals []string) interfaces.Event { ev.e.Strs(key, vals); return ev }
func (ev *DefaultEvent) Int(key string, val int) interfaces.Event   { ev.e.Int(key, val); return ev }
func (ev *DefaultEvent) Bool(key string, val bool) interfaces.Event { ev.e.Bool(key, val); return ev }
func (ev *DefaultEvent) Err(err error) interfaces.Event             { ev.e.Err(err); return ev }
func (ev *DefaultEvent) Interface(key string, v interface{}) interfaces.Event { ev.e.Interface(key, v); return ev }
func (ev *DefaultEvent) Msg(msg string)                   { ev.e.Msg(msg) }
func (ev *DefaultEvent) Msgf(format string, args ...interface{}) { ev.e.Msg(fmt.Sprintf(format, args...)) }

// DefaultLogger は zerolog.Logger を内部に持つ実装
type DefaultLogger struct {
	l zerolog.Logger
}

// コンストラクタ
func NewDefaultLogger() interfaces.Logger {
	return &DefaultLogger{l: log.Logger}
}

// ---- ポインタレシーバで実装 ----
func (d *DefaultLogger) Debug() interfaces.Event { return &DefaultEvent{e: d.l.Debug()} }
func (d *DefaultLogger) Info() interfaces.Event  { return &DefaultEvent{e: d.l.Info()} }
func (d *DefaultLogger) Warn() interfaces.Event  { return &DefaultEvent{e: d.l.Warn()} }
func (d *DefaultLogger) Error() interfaces.Event { return &DefaultEvent{e: d.l.Error()} }
func (d *DefaultLogger) Fatal() interfaces.Event { return &DefaultEvent{e: d.l.Fatal()} }
func (d *DefaultLogger) Panic() interfaces.Event { return &DefaultEvent{e: d.l.Panic()} }

func (d *DefaultLogger) WithField(key string, value interface{}) interfaces.Logger {
	ctx := d.l.With().Interface(key, value)
	return &DefaultLogger{l: ctx.Logger()}
}

func (d *DefaultLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	ctx := d.l.With()
	for k, v := range fields {
		ctx = ctx.Interface(k, v)
	}
	return &DefaultLogger{l: ctx.Logger()}
}

func (d *DefaultLogger) Level(l interfaces.Level) interfaces.Logger {
	var zl zerolog.Level
	switch l {
	case interfaces.LevelDebug:
		zl = zerolog.DebugLevel
	case interfaces.LevelInfo:
		zl = zerolog.InfoLevel
	case interfaces.LevelWarn:
		zl = zerolog.WarnLevel
	case interfaces.LevelError:
		zl = zerolog.ErrorLevel
	case interfaces.LevelFatal:
		zl = zerolog.FatalLevel
	case interfaces.LevelPanic:
		zl = zerolog.PanicLevel
	default:
		zl = zerolog.InfoLevel
	}
	return &DefaultLogger{l: d.l.Level(zl)}
}

func (d *DefaultLogger) Output(w io.Writer) interfaces.Logger {
	return &DefaultLogger{l: d.l.Output(w)}
}

