package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestRequestIDIsAdded(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(&buf, "info", "json", "service", "bank-api")
	if err != nil {
		t.Fatal(err)
	}

	ctx := WithRequestID(context.Background(), "abc-123")
	log.With("component", "test").InfoContext(ctx, "hello")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid json %q: %v", buf.String(), err)
	}
	if entry["request_id"] != "abc-123" || entry["service"] != "bank-api" || entry["component"] != "test" {
		t.Errorf("unexpected entry %v", entry)
	}
}

func TestLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	log, err := New(&buf, "warn", "text")
	if err != nil {
		t.Fatal(err)
	}

	log.Info("hidden")
	if buf.Len() != 0 {
		t.Errorf("info logged at warn level: %q", buf.String())
	}
	log.Warn("shown")
	if buf.Len() == 0 {
		t.Error("warn not logged")
	}
}

func TestInvalidConfig(t *testing.T) {
	if _, err := New(&bytes.Buffer{}, "loud", "json"); err == nil {
		t.Error("expected error for invalid level")
	}
	if _, err := New(&bytes.Buffer{}, "info", "xml"); err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestRequestIDMissing(t *testing.T) {
	if id := RequestID(context.Background()); id != "" {
		t.Errorf("RequestID = %q, want empty", id)
	}
}
