package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewWritesStructuredJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	logger := New(slog.LevelDebug, &buf)
	logger.Info("booted", "mode", "http")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if record["msg"] != "booted" {
		t.Fatalf("msg = %#v, want %q", record["msg"], "booted")
	}

	if record["level"] != "INFO" {
		t.Fatalf("level = %#v, want %q", record["level"], "INFO")
	}

	if record["mode"] != "http" {
		t.Fatalf("mode = %#v, want %q", record["mode"], "http")
	}
}
