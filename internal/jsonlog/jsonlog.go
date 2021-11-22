package jsonlog

import (
	"encoding/json"
	"io"
	"runtime/debug"
	"sync"
	"time"
)

// Level type for severity level of log entries
type Level int8

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

func (l Level) String() string {
	switch l {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

// Logger holds output destination that log entries will be written to, minimum severity level & mutex for coordinating the writes
type Logger struct {
	out      io.Writer
	minLevel Level
	mu       sync.Mutex
}

// New logger instance
func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

// PrintInfo helper method
func (l *Logger) PrintInfo(message string, properties map[string]string) {
	l.print(LevelInfo, message, properties)
}

// PrintError helper method
func (l *Logger) PrintError(err error, properties map[string]string) {
	l.print(LevelError, err.Error(), properties)
}

// PrintFatal helper method
func (l *Logger) PrintFatal(err error, properties map[string]string) {
	l.print(LevelFatal, err.Error(), properties)
}

func (l *Logger) print(level Level, message string, properties map[string]string) (int, error) {
	// No further action iof severity level below minimum
	if level < l.minLevel {
		return 0, nil
	}

	// Anonymous struct holding log entry data
	aux := struct {
		Level      string            `json:"level"`
		Time       string            `json:"time"`
		Message    string            `json:"message"`
		Properties map[string]string `json:"properties,omitempty"`
		Trace      string            `json:"trace,omitempty"`
	}{
		Level:      level.String(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	// Include stack trace for entries at ERROR/FATAL levels
	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	// Variable for holding log entry text
	var line []byte

	// Marshal anonymous struct to JSON & store in line variable
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.String() + ": unable to marshal log message:" + err.Error())
	}

	// Lock mutex to prevent multiple log entries
	l.mu.Lock()
	defer l.mu.Unlock()

	// Write log entry + new line
	return l.out.Write(append(line, '\n'))
}

// Write loge entry at ERROR level w/o additional properties
func (l *Logger) Write(message []byte) (n int, err error) {
	return l.print(LevelError, string(message), nil)
}
