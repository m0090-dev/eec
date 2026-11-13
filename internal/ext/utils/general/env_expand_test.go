package general_test

import (
	"io"
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/m0090-dev/eec/internal/ext/interfaces/impl"
	"github.com/m0090-dev/eec/internal/ext/utils/general"
)

// --- capturing logger: records Warn messages and their "command" field ---

type captureEvent struct {
	owner  *captureLogger
	fields map[string]string
	err    error
}

func (e *captureEvent) Str(k, v string) interfaces.Event {
	e.fields[k] = v
	return e
}
func (e *captureEvent) Strs(k string, v []string) interfaces.Event         { return e }
func (e *captureEvent) Int(k string, v int) interfaces.Event               { return e }
func (e *captureEvent) Bool(k string, v bool) interfaces.Event             { return e }
func (e *captureEvent) Err(err error) interfaces.Event                     { e.err = err; return e }
func (e *captureEvent) Interface(k string, v interface{}) interfaces.Event { return e }
func (e *captureEvent) Msg(msg string) {
	e.owner.warns = append(e.owner.warns, msg+" command="+e.fields["command"])
}
func (e *captureEvent) Msgf(format string, args ...interface{}) {}

type noopEvent struct{}

func (e *noopEvent) Str(k, v string) interfaces.Event                   { return e }
func (e *noopEvent) Strs(k string, v []string) interfaces.Event         { return e }
func (e *noopEvent) Int(k string, v int) interfaces.Event               { return e }
func (e *noopEvent) Bool(k string, v bool) interfaces.Event             { return e }
func (e *noopEvent) Err(err error) interfaces.Event                     { return e }
func (e *noopEvent) Interface(k string, v interface{}) interfaces.Event { return e }
func (e *noopEvent) Msg(msg string)                                     {}
func (e *noopEvent) Msgf(format string, args ...interface{})            {}

type captureLogger struct{ warns []string }

func (l *captureLogger) Warn() interfaces.Event {
	return &captureEvent{owner: l, fields: map[string]string{}}
}
func (l *captureLogger) Debug() interfaces.Event { return &noopEvent{} }
func (l *captureLogger) Info() interfaces.Event  { return &noopEvent{} }
func (l *captureLogger) Error() interfaces.Event { return &noopEvent{} }
func (l *captureLogger) Fatal() interfaces.Event { return &noopEvent{} }
func (l *captureLogger) Panic() interfaces.Event { return &noopEvent{} }
func (l *captureLogger) WithField(key string, value interface{}) interfaces.Logger {
	return l
}
func (l *captureLogger) WithFields(fields map[string]interface{}) interfaces.Logger {
	return l
}
func (l *captureLogger) EnableDebug() interfaces.Logger              { return l }
func (l *captureLogger) Level(lv interfaces.Level) interfaces.Logger { return l }
func (l *captureLogger) Output(w io.Writer) interfaces.Logger        { return l }

func newCaptureRuntime() (*impl.DefaultRuntime, *captureLogger) {
	logger := &captureLogger{}
	rt := &impl.DefaultRuntime{}
	rt.SetLogger(logger)
	return rt, logger
}

func TestExpandEnvAndCommands_FailingCommand_LogsWarning(t *testing.T) {
	rt, logger := newCaptureRuntime()
	got := general.ExpandEnvAndCommands(rt, "x=$(exit 3)", map[string][]string{})
	if got != "x=" {
		t.Errorf("failing substitution should expand to empty, got %q", got)
	}
	if len(logger.warns) == 0 {
		t.Fatal("expected a warning for the failed command substitution, got none")
	}
	if !strings.Contains(logger.warns[0], "exit 3") {
		t.Errorf("warning should include the failed command line, got: %q", logger.warns[0])
	}
}

func TestExpandEnvAndCommands_SuccessfulCommand_NoWarning(t *testing.T) {
	rt, logger := newCaptureRuntime()
	got := general.ExpandEnvAndCommands(rt, "x=$(echo hi)", map[string][]string{})
	if got != "x=hi" {
		t.Errorf("expected %q, got %q", "x=hi", got)
	}
	if len(logger.warns) != 0 {
		t.Errorf("expected no warnings, got %v", logger.warns)
	}
}
