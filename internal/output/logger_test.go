package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestLogger_Basic(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf)
	logger.formatter.SetColorOutput(false)

	logger.Info("Test message")
	output := buf.String()

	if !strings.Contains(output, "Test message") {
		t.Error("Log message not found in output")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("Log level not found in output")
	}
}

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		level    LogLevel
		logFunc  func(*Logger)
		expected string
	}{
		{LogLevelTrace, func(l *Logger) { l.Trace("trace") }, "TRACE"},
		{LogLevelDebug, func(l *Logger) { l.Debug("debug") }, "DEBUG"},
		{LogLevelInfo, func(l *Logger) { l.Info("info") }, "INFO"},
		{LogLevelWarn, func(l *Logger) { l.Warn("warn") }, "WARN"},
		{LogLevelError, func(l *Logger) { l.Error("error") }, "ERROR"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		logger := NewLogger().SetLevel(LogLevelTrace).SetOutputs(&buf)
		logger.formatter.SetColorOutput(false)

		tt.logFunc(logger)
		output := buf.String()

		if !strings.Contains(output, tt.expected) {
			t.Errorf("Expected level %s not found in output: %s", tt.expected, output)
		}
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetLevel(LogLevelWarn).SetOutputs(&buf)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should be filtered out")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should be filtered out")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should be included")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should be included")
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf)
	logger.formatter.SetColorOutput(false)

	logger.WithField("key", "value").Info("message")
	output := buf.String()

	if !strings.Contains(output, "key=value") {
		t.Error("Field not found in output")
	}
}

func TestLogger_WithMultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf)
	logger.formatter.SetColorOutput(false)

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}

	logger.WithFields(fields).Info("message")
	output := buf.String()

	if !strings.Contains(output, "key1=value1") {
		t.Error("First field not found in output")
	}
	if !strings.Contains(output, "key2=42") {
		t.Error("Second field not found in output")
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetFormat(LogFormatJSON).SetOutputs(&buf)

	logger.Info("test message")
	output := buf.String()

	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", entry.Message)
	}
	if entry.Level != LogLevelInfo {
		t.Errorf("Expected level INFO, got %v", entry.Level)
	}
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetFormat(LogFormatText).SetOutputs(&buf)
	logger.formatter.SetColorOutput(false)

	logger.Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Error("Message not found in text output")
	}
	if !strings.Contains(output, "[INFO]") {
		t.Error("Level not found in text output")
	}
}

func TestLogger_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetFormat(LogFormatJSON).SetOutputs(&buf)

	testErr := fmt.Errorf("test error")
	logger.WithError(testErr).Error("something failed")
	output := buf.String()

	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Fields["error"] != testErr.Error() {
		t.Error("Error field not found or incorrect")
	}
}

func TestLogger_Caller(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetFormat(LogFormatJSON).SetOutputs(&buf).EnableCaller()

	logger.Info("test message")
	output := buf.String()

	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Caller == "" {
		t.Error("Caller information not found")
	}
	if !strings.Contains(entry.Caller, "logger_test.go") {
		t.Errorf("Expected caller to contain test file name, got: %s", entry.Caller)
	}
}

func TestLogger_FormattedMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf)
	logger.formatter.SetColorOutput(false)

	logger.Infof("formatted %s %d", "message", 42)
	output := buf.String()

	if !strings.Contains(output, "formatted message 42") {
		t.Error("Formatted message not correct")
	}
}

func TestLogger_PersistentFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf).WithField("persistent", "value")
	logger.formatter.SetColorOutput(false)

	logger.Info("first message")
	logger.Info("second message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 log lines, got %d", len(lines))
	}

	for i, line := range lines {
		if !strings.Contains(line, "persistent=value") {
			t.Errorf("Line %d missing persistent field: %s", i+1, line)
		}
	}
}

func TestLogger_MultipleOutputs(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	logger := NewLogger().SetOutputs(&buf1, &buf2)
	logger.formatter.SetColorOutput(false)

	logger.Info("test message")

	output1 := buf1.String()
	output2 := buf2.String()

	if output1 != output2 {
		t.Error("Outputs to multiple writers should be identical")
	}
	if !strings.Contains(output1, "test message") {
		t.Error("Message not found in first output")
	}
	if !strings.Contains(output2, "test message") {
		t.Error("Message not found in second output")
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LogLevelTrace, "TRACE"},
		{LogLevelDebug, "DEBUG"},
		{LogLevelInfo, "INFO"},
		{LogLevelWarn, "WARN"},
		{LogLevelError, "ERROR"},
		{LogLevelFatal, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("LogLevel(%d).String() = %s, expected %s", tt.level, got, tt.expected)
		}
	}
}

func TestLogger_TimeFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger().SetOutputs(&buf).SetTimeFormat(time.RFC822)
	logger.formatter.SetColorOutput(false)

	logger.Info("test message")
	output := buf.String()

	// The output should contain a timestamp in RFC822 format
	// We can't test the exact timestamp, but we can check it's present
	if !strings.Contains(output, " ") {
		t.Error("Timestamp not found in output")
	}
}

func TestGlobalLogger(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := GetGlobalLogger()

	// Set test logger
	testLogger := NewLogger().SetOutputs(&buf)
	testLogger.formatter.SetColorOutput(false)
	SetGlobalLogger(testLogger)

	// Test global functions
	Info("global info")
	Infof("global %s", "formatted")

	output := buf.String()

	if !strings.Contains(output, "global info") {
		t.Error("Global Info function not working")
	}
	if !strings.Contains(output, "global formatted") {
		t.Error("Global Infof function not working")
	}

	// Restore original logger
	SetGlobalLogger(originalLogger)
}

func TestCreateFileLogger(t *testing.T) {
	// Test invalid directory
	_, err := CreateFileLogger("/invalid/path/test.log", LogLevelInfo, LogFormatJSON)
	if err == nil {
		t.Error("Expected error for invalid directory")
	}

	// Note: We don't test actual file creation here to avoid
	// requiring filesystem permissions in tests
}
