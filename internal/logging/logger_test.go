package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitDefaults(t *testing.T) {
	// Reset global state
	Shutdown()

	dir := t.TempDir()
	Init(Config{
		Debug:  true,
		LogDir: dir,
	})
	defer Shutdown()

	// Logger should not be nil
	l := Logger()
	if l == nil {
		t.Fatal("expected non-nil logger after Init")
	}

	// Log something and check the file exists with JSONL content
	l.Info("test_message", "key", "value")

	logPath := filepath.Join(dir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("log file is empty")
	}

	// Parse as JSON
	var record map[string]any
	if err := json.Unmarshal(data[:len(data)-1], &record); err != nil { // trim trailing newline
		// Try to find the first complete line
		for i, b := range data {
			if b == '\n' {
				if err := json.Unmarshal(data[:i], &record); err != nil {
					t.Fatalf("failed to parse JSONL: %v (data: %s)", err, string(data[:i]))
				}
				break
			}
		}
	}

	if record["msg"] != "test_message" {
		t.Errorf("expected msg=test_message, got %v", record["msg"])
	}
	if record["key"] != "value" {
		t.Errorf("expected key=value, got %v", record["key"])
	}
}

func TestInitNonDebug(t *testing.T) {
	// When debug is false and LogDir is empty, logs should be discarded
	Shutdown()

	Init(Config{
		Debug: false,
	})
	defer Shutdown()

	l := Logger()
	if l == nil {
		t.Fatal("expected non-nil logger even in non-debug mode")
	}

	// Should not panic
	l.Info("this goes nowhere")
}

func TestForComponent(t *testing.T) {
	Shutdown()

	dir := t.TempDir()
	Init(Config{
		Debug:  true,
		LogDir: dir,
	})
	defer Shutdown()

	cl := ForComponent(CompStatus)
	cl.Info("state_change", "from", "idle", "to", "running")

	logPath := filepath.Join(dir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	var record map[string]any
	for i, b := range data {
		if b == '\n' {
			if err := json.Unmarshal(data[:i], &record); err == nil {
				break
			}
		}
	}

	if record["component"] != CompStatus {
		t.Errorf("expected component=%s, got %v", CompStatus, record["component"])
	}
}

func TestLevelFiltering(t *testing.T) {
	Shutdown()

	dir := t.TempDir()
	Init(Config{
		Debug:  true,
		LogDir: dir,
		Level:  "warn",
	})
	defer Shutdown()

	l := Logger()
	l.Info("should_be_filtered")
	l.Warn("should_appear")

	logPath := filepath.Join(dir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	if len(content) == 0 {
		t.Fatal("log file is empty, expected at least the warn message")
	}

	// Check that the info message was filtered
	if containsMsg(data, "should_be_filtered") {
		t.Error("info message should have been filtered at warn level")
	}
	if !containsMsg(data, "should_appear") {
		t.Error("warn message should have appeared")
	}
}

func TestTextFormat(t *testing.T) {
	Shutdown()

	dir := t.TempDir()
	Init(Config{
		Debug:  true,
		LogDir: dir,
		Format: "text",
	})
	defer Shutdown()

	Logger().Info("text_format_test")

	logPath := filepath.Join(dir, "debug.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	// Text format should NOT be valid JSON
	var record map[string]any
	if err := json.Unmarshal(data, &record); err == nil {
		t.Error("expected text format, but got valid JSON")
	}
}

func TestDumpRingBuffer(t *testing.T) {
	Shutdown()

	dir := t.TempDir()
	Init(Config{
		Debug:          true,
		LogDir:         dir,
		RingBufferSize: 1024,
	})
	defer Shutdown()

	Logger().Info("ring_test_message")

	dumpPath := filepath.Join(dir, "crash-dump.jsonl")
	if err := DumpRingBuffer(dumpPath); err != nil {
		t.Fatalf("DumpRingBuffer failed: %v", err)
	}

	data, err := os.ReadFile(dumpPath)
	if err != nil {
		t.Fatalf("failed to read dump file: %v", err)
	}

	if len(data) == 0 {
		t.Error("crash dump file is empty")
	}
}

// containsMsg checks if JSONL data contains a record with the given msg field.
func containsMsg(data []byte, msg string) bool {
	start := 0
	for i, b := range data {
		if b == '\n' {
			var record map[string]any
			if err := json.Unmarshal(data[start:i], &record); err == nil {
				if record["msg"] == msg {
					return true
				}
			}
			start = i + 1
		}
	}
	return false
}
