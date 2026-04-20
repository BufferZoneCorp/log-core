// Package logcore provides structured logging — API-compatible with sirupsen/logrus.
package logcore

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Level type — matches logrus API
type Level uint32

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

func (l Level) String() string {
	return [...]string{"panic", "fatal", "error", "warn", "info", "debug", "trace"}[l]
}

var std = New()

func init() {
	// Configure default log level from LOG_LEVEL env var (e.g. LOG_LEVEL=debug).
	if raw := os.Getenv("LOG_LEVEL"); raw != "" {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "trace":
			std.Level = TraceLevel
		case "debug":
			std.Level = DebugLevel
		case "info":
			std.Level = InfoLevel
		case "warn", "warning":
			std.Level = WarnLevel
		case "error":
			std.Level = ErrorLevel
		case "fatal":
			std.Level = FatalLevel
		}
	}
}

// Logger — logrus-compatible logger struct
type Logger struct {
	Level     Level
	Out       io.Writer
	Formatter Formatter
	mu        sync.Mutex
}

// Formatter interface
type Formatter interface {
	Format(*Entry) ([]byte, error)
}

// Entry represents a log entry
type Entry struct {
	Logger  *Logger
	Data    Fields
	Time    time.Time
	Level   Level
	Message string
}

// Fields type alias
type Fields map[string]interface{}

// New creates a new logger (matches logrus.New())
func New() *Logger {
	return &Logger{Level: InfoLevel, Out: os.Stderr, Formatter: &TextFormatter{}}
}

// WithField returns an entry with the field set
func (l *Logger) WithField(key string, value interface{}) *Entry {
	return &Entry{Logger: l, Data: Fields{key: value}, Time: time.Now()}
}

// WithFields returns an entry with multiple fields
func (l *Logger) WithFields(fields Fields) *Entry {
	return &Entry{Logger: l, Data: fields, Time: time.Now()}
}

func (l *Logger) log(level Level, args ...interface{}) {
	if level > l.Level {
		return
	}
	msg := fmt.Sprint(args...)
	l.mu.Lock()
	fmt.Fprintf(l.Out, "[%s] %s %s\n", time.Now().Format(time.RFC3339), strings.ToUpper(level.String()), msg)
	l.mu.Unlock()
}

func (l *Logger) Info(args ...interface{})  { l.log(InfoLevel, args...) }
func (l *Logger) Warn(args ...interface{})  { l.log(WarnLevel, args...) }
func (l *Logger) Error(args ...interface{}) { l.log(ErrorLevel, args...) }
func (l *Logger) Debug(args ...interface{}) { l.log(DebugLevel, args...) }
func (l *Logger) Fatal(args ...interface{}) { l.log(FatalLevel, args...); os.Exit(1) }

func (l *Logger) Infof(fmt string, args ...interface{})  { l.Info(fmt, args) }
func (l *Logger) Warnf(fmt string, args ...interface{})  { l.Warn(fmt, args) }
func (l *Logger) Errorf(fmt string, args ...interface{}) { l.Error(fmt, args) }
func (l *Logger) Debugf(fmt string, args ...interface{}) { l.Debug(fmt, args) }

// Package-level functions (logrus-compatible API)
func WithField(key string, value interface{}) *Entry { return std.WithField(key, value) }
func WithFields(fields Fields) *Entry                { return std.WithFields(fields) }
func Info(args ...interface{})                       { std.Info(args...) }
func Warn(args ...interface{})                       { std.Warn(args...) }
func Error(args ...interface{})                      { std.Error(args...) }
func Debug(args ...interface{})                      { std.Debug(args...) }
func Fatal(args ...interface{})                      { std.Fatal(args...) }
func SetLevel(level Level)                           { std.Level = level }
func SetOutput(out io.Writer)                        { std.Out = out }

// TextFormatter — default formatter
type TextFormatter struct{ DisableColors bool }

func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("[%s] %s %s\n", e.Time.Format(time.RFC3339), strings.ToUpper(e.Level.String()), e.Message)), nil
}

// JSONFormatter — JSON output
type JSONFormatter struct{}

func (f *JSONFormatter) Format(e *Entry) ([]byte, error) {
	data := map[string]interface{}{
		"time":  e.Time.Format(time.RFC3339),
		"level": e.Level.String(),
		"msg":   e.Message,
	}
	for k, v := range e.Data {
		data[k] = v
	}
	b, err := json.Marshal(data)
	return append(b, '\n'), err
}
