package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the importance level of a log message
type LogLevel int

const (
	LogLevelTrace LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogFormat represents the output format for logs
type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Caller    string                 `json:"caller,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger handles structured logging with multiple outputs and formats
type Logger struct {
	level         LogLevel
	format        LogFormat
	outputs       []io.Writer
	fields        map[string]interface{}
	formatter     *Formatter
	includeCaller bool
	timeFormat    string
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	return &Logger{
		level:      LogLevelInfo,
		format:     LogFormatText,
		outputs:    []io.Writer{os.Stderr},
		fields:     make(map[string]interface{}),
		formatter:  NewFormatter(os.Stderr),
		timeFormat: time.RFC3339,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) *Logger {
	l.level = level
	return l
}

// SetFormat sets the output format (text or JSON)
func (l *Logger) SetFormat(format LogFormat) *Logger {
	l.format = format
	return l
}

// AddOutput adds an output writer for logs
func (l *Logger) AddOutput(w io.Writer) *Logger {
	l.outputs = append(l.outputs, w)
	return l
}

// SetOutputs replaces all output writers
func (l *Logger) SetOutputs(outputs ...io.Writer) *Logger {
	l.outputs = outputs
	return l
}

// WithField adds a field that will be included in all subsequent log entries
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:         l.level,
		format:        l.format,
		outputs:       l.outputs,
		fields:        make(map[string]interface{}),
		formatter:     l.formatter,
		includeCaller: l.includeCaller,
		timeFormat:    l.timeFormat,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := l
	for k, v := range fields {
		newLogger = newLogger.WithField(k, v)
	}
	return newLogger
}

// WithError adds an error field
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// EnableCaller includes caller information in log entries
func (l *Logger) EnableCaller() *Logger {
	l.includeCaller = true
	return l
}

// DisableCaller removes caller information from log entries
func (l *Logger) DisableCaller() *Logger {
	l.includeCaller = false
	return l
}

// SetTimeFormat sets the timestamp format
func (l *Logger) SetTimeFormat(format string) *Logger {
	l.timeFormat = format
	return l
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, fields ...map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}

	// Add persistent fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add call-specific fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			entry.Fields[k] = v
		}
	}

	// Add caller information if enabled
	if l.includeCaller {
		if pc, file, line, ok := runtime.Caller(2); ok {
			funcName := runtime.FuncForPC(pc).Name()
			entry.Caller = fmt.Sprintf("%s:%d:%s", filepath.Base(file), line, filepath.Base(funcName))
		}
	}

	// Remove empty fields map
	if len(entry.Fields) == 0 {
		entry.Fields = nil
	}

	l.writeEntry(entry)
}

// writeEntry writes a log entry to all configured outputs
func (l *Logger) writeEntry(entry LogEntry) {
	var output string

	switch l.format {
	case LogFormatJSON:
		if data, err := json.Marshal(entry); err == nil {
			output = string(data) + "\n"
		} else {
			output = fmt.Sprintf(`{"level":"ERROR","message":"Failed to marshal log entry: %v"}%s`, err, "\n")
		}
	case LogFormatText:
		output = l.formatTextEntry(entry)
	}

	for _, w := range l.outputs {
		fmt.Fprint(w, output)
	}
}

// formatTextEntry formats a log entry as human-readable text
func (l *Logger) formatTextEntry(entry LogEntry) string {
	var parts []string

	// Timestamp
	parts = append(parts, entry.Timestamp.Format(l.timeFormat))

	// Level with color
	levelStr := fmt.Sprintf("[%s]", entry.Level.String())
	switch entry.Level {
	case LogLevelTrace, LogLevelDebug:
		levelStr = l.formatter.colorize(levelStr, l.formatter.theme.Muted, StyleDim)
	case LogLevelInfo:
		levelStr = l.formatter.colorize(levelStr, l.formatter.theme.Info, StyleNormal)
	case LogLevelWarn:
		levelStr = l.formatter.colorize(levelStr, l.formatter.theme.Warning, StyleBold)
	case LogLevelError, LogLevelFatal:
		levelStr = l.formatter.colorize(levelStr, l.formatter.theme.Error, StyleBold)
	}
	parts = append(parts, levelStr)

	// Caller information
	if entry.Caller != "" {
		caller := l.formatter.colorize(fmt.Sprintf("(%s)", entry.Caller), l.formatter.theme.Muted, StyleDim)
		parts = append(parts, caller)
	}

	// Message
	parts = append(parts, entry.Message)

	// Fields
	if entry.Fields != nil && len(entry.Fields) > 0 {
		var fieldPairs []string
		for k, v := range entry.Fields {
			fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", k, v))
		}
		fields := l.formatter.colorize(fmt.Sprintf("[%s]", strings.Join(fieldPairs, " ")), l.formatter.theme.Secondary, StyleDim)
		parts = append(parts, fields)
	}

	return strings.Join(parts, " ") + "\n"
}

// Trace logs a trace message
func (l *Logger) Trace(message string, fields ...map[string]interface{}) {
	l.log(LogLevelTrace, message, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	l.log(LogLevelDebug, message, fields...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(LogLevelInfo, message, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(LogLevelWarn, message, fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	l.log(LogLevelError, message, fields...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(message string, fields ...map[string]interface{}) {
	l.log(LogLevelFatal, message, fields...)
	os.Exit(1)
}

// Tracef logs a formatted trace message
func (l *Logger) Tracef(format string, args ...interface{}) {
	l.Trace(fmt.Sprintf(format, args...))
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal(fmt.Sprintf(format, args...))
}

// CreateFileLogger creates a logger that writes to a file
func CreateFileLogger(filename string, level LogLevel, format LogFormat) (*Logger, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := NewLogger().
		SetLevel(level).
		SetFormat(format).
		SetOutputs(file)

	return logger, nil
}

// Global logger instance
var globalLogger = NewLogger()

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// Global logging functions that use the global logger
func Trace(message string, fields ...map[string]interface{}) {
	globalLogger.Trace(message, fields...)
}

func Debug(message string, fields ...map[string]interface{}) {
	globalLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	globalLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	globalLogger.Warn(message, fields...)
}

func Error(message string, fields ...map[string]interface{}) {
	globalLogger.Error(message, fields...)
}

func Fatal(message string, fields ...map[string]interface{}) {
	globalLogger.Fatal(message, fields...)
}

func Tracef(format string, args ...interface{}) {
	globalLogger.Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}
