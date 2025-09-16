package output

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
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
	Timestamp time.Time      `json:"timestamp"`
	Level     LogLevel       `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	Caller    string         `json:"caller,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// Logger handles structured logging with multiple outputs and formats
type Logger struct {
	level         LogLevel
	format        LogFormat
	outputs       []io.Writer
	fields        map[string]any
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
		fields:     make(map[string]any),
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
func (l *Logger) WithField(key string, value any) *Logger {
	newLogger := &Logger{
		level:         l.level,
		format:        l.format,
		outputs:       l.outputs,
		fields:        make(map[string]any),
		formatter:     l.formatter,
		includeCaller: l.includeCaller,
		timeFormat:    l.timeFormat,
	}

	// Copy existing fields
	maps.Copy(newLogger.fields, l.fields)

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields
func (l *Logger) WithFields(fields map[string]any) *Logger {
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
func (l *Logger) log(level LogLevel, message string, fields ...map[string]any) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]any),
	}

	// Add persistent fields
	maps.Copy(entry.Fields, l.fields)

	// Add call-specific fields
	for _, fieldMap := range fields {
		maps.Copy(entry.Fields, fieldMap)
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

	// Timestamp - shorter format for better readability
	timeFormat := "15:04:05"
	if l.timeFormat == time.RFC3339 {
		parts = append(parts, entry.Timestamp.Format(timeFormat))
	} else {
		parts = append(parts, entry.Timestamp.Format(l.timeFormat))
	}

	// Level with color and better formatting
	levelStr := l.formatLogLevel(entry.Level)
	parts = append(parts, levelStr)

	// Caller information
	if entry.Caller != "" {
		caller := l.formatter.colorize(fmt.Sprintf("(%s)", entry.Caller), l.formatter.theme.Muted, StyleDim)
		parts = append(parts, caller)
	}

	// Message
	parts = append(parts, entry.Message)

	// Fields with better formatting
	if entry.Fields != nil && len(entry.Fields) > 0 {
		fieldsStr := l.formatFields(entry.Fields)
		parts = append(parts, fieldsStr)
	}

	return strings.Join(parts, " ") + "\n"
}

// formatLogLevel formats the log level with appropriate colors and icons
func (l *Logger) formatLogLevel(level LogLevel) string {
	var icon, text string
	var color Color
	var style Style

	switch level {
	case LogLevelTrace:
		icon, text, color, style = "🔍", "TRACE", l.formatter.theme.Muted, StyleDim
	case LogLevelDebug:
		icon, text, color, style = "🐛", "DEBUG", l.formatter.theme.Muted, StyleDim
	case LogLevelInfo:
		icon, text, color, style = "ℹ️", "INFO", l.formatter.theme.Info, StyleNormal
	case LogLevelWarn:
		icon, text, color, style = "⚠️", "WARN", l.formatter.theme.Warning, StyleBold
	case LogLevelError:
		icon, text, color, style = "❌", "ERROR", l.formatter.theme.Error, StyleBold
	case LogLevelFatal:
		icon, text, color, style = "💀", "FATAL", l.formatter.theme.Error, StyleBold
	default:
		icon, text, color, style = "❓", "UNKNOWN", l.formatter.theme.Muted, StyleNormal
	}

	if l.formatter.colorOutput {
		return l.formatter.colorize(fmt.Sprintf("%s [%s]", icon, text), color, style)
	}
	return fmt.Sprintf("[%s]", text) // No icons in non-color mode for better screen reader compatibility
}

// formatFields formats the log fields for better readability
func (l *Logger) formatFields(fields map[string]any) string {
	if len(fields) == 0 {
		return ""
	}

	var fieldPairs []string

	// Sort by key for consistent output
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}

	for _, k := range keys {
		v := fields[k]
		// Format special field types
		switch k {
		case "error":
			fieldPairs = append(fieldPairs, l.formatter.colorize(fmt.Sprintf("error=%v", v), l.formatter.theme.Error, StyleNormal))
		case "duration", "elapsed":
			fieldPairs = append(fieldPairs, l.formatter.colorize(fmt.Sprintf("%s=%v", k, v), l.formatter.theme.Success, StyleNormal))
		case "component", "module":
			fieldPairs = append(fieldPairs, l.formatter.colorize(fmt.Sprintf("%s=%v", k, v), l.formatter.theme.Primary, StyleNormal))
		default:
			fieldPairs = append(fieldPairs, fmt.Sprintf("%s=%v", k, v))
		}
	}

	fieldsText := strings.Join(fieldPairs, " ")
	return l.formatter.colorize(fmt.Sprintf("[%s]", fieldsText), l.formatter.theme.Secondary, StyleDim)
}

// SetFormatter allows changing the formatter used by the logger
func (l *Logger) SetFormatter(formatter *Formatter) *Logger {
	l.formatter = formatter
	return l
}

// Performance logging helpers
func (l *Logger) LogDuration(operation string, duration time.Duration, fields ...map[string]any) {
	durationFields := map[string]any{
		"operation": operation,
		"duration":  duration.String(),
	}
	// Merge additional fields
	for _, fieldMap := range fields {
		maps.Copy(durationFields, fieldMap)
	}
	// Choose appropriate log level based on duration
	if duration > 5*time.Second {
		l.Warn("Slow operation detected", durationFields)
	} else if duration > 1*time.Second {
		l.Info("Operation completed", durationFields)
	} else {
		l.Debug("Operation completed", durationFields)
	}
}

// LogMemoryUsage logs memory usage statistics
func (l *Logger) LogMemoryUsage(component string) {
	// This would integrate with runtime.MemStats in a real implementation
	fields := map[string]interface{}{
		"component": component,
		"memory":    "tracking_placeholder",
	}
	l.Debug("Memory usage", fields)
}

// Trace logs a trace message
func (l *Logger) Trace(message string, fields ...map[string]any) {
	l.log(LogLevelTrace, message, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]any) {
	l.log(LogLevelDebug, message, fields...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]any) {
	l.log(LogLevelInfo, message, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]any) {
	l.log(LogLevelWarn, message, fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, fields ...map[string]any) {
	l.log(LogLevelError, message, fields...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(message string, fields ...map[string]any) {
	l.log(LogLevelFatal, message, fields...)
	os.Exit(1)
}

// Tracef logs a formatted trace message
func (l *Logger) Tracef(format string, args ...any) {
	l.Trace(fmt.Sprintf(format, args...))
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

// Fatalf logs a formatted fatal message and exits
func (l *Logger) Fatalf(format string, args ...any) {
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

// Trace - Global logging functions that use the global logger
func Trace(message string, fields ...map[string]any) {
	globalLogger.Trace(message, fields...)
}

func Debug(message string, fields ...map[string]any) {
	globalLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]any) {
	globalLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]any) {
	globalLogger.Warn(message, fields...)
}

func Error(message string, fields ...map[string]any) {
	globalLogger.Error(message, fields...)
}

func Fatal(message string, fields ...map[string]any) {
	globalLogger.Fatal(message, fields...)
}

func Tracef(format string, args ...any) {
	globalLogger.Tracef(format, args...)
}

func Debugf(format string, args ...any) {
	globalLogger.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	globalLogger.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	globalLogger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	globalLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	globalLogger.Fatalf(format, args...)
}
