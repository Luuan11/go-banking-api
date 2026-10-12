package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/luuan11/bank-system/pkg/logger"
)

// captureLogs swaps the default logger for one writing JSON lines to a buffer.
func captureLogs(t *testing.T, level string) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	log, err := logger.New(&buf, level, "json")
	if err != nil {
		t.Fatal(err)
	}
	previous := slog.Default()
	slog.SetDefault(log)
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, raw := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(raw) == 0 {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal(raw, &entry); err != nil {
			t.Fatalf("invalid log line %q: %v", raw, err)
		}
		lines = append(lines, entry)
	}
	return lines
}

// chain mirrors the order used in main.
func chain(mux *http.ServeMux) http.Handler {
	return RequestID(Logger(Recovery(CORS(mux))))
}

func TestRequestLogLine(t *testing.T) {
	buf := captureLogs(t, "info")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/accounts/{id}/deposits", func(w http.ResponseWriter, r *http.Request) {
		SetErrorCode(r.Context(), "INSUFFICIENT_BALANCE")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("12345"))
	})

	req := httptest.NewRequest(http.MethodPost, "/api/accounts/42/deposits", nil)
	req.Header.Set(RequestIDHeader, "client-id-1")
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	chain(mux).ServeHTTP(rec, req)

	if got := rec.Header().Get(RequestIDHeader); got != "client-id-1" {
		t.Errorf("response %s = %q", RequestIDHeader, got)
	}

	lines := logLines(t, buf)
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1", len(lines))
	}
	want := map[string]any{
		"level":      "WARN",
		"msg":        "request",
		"request_id": "client-id-1",
		"method":     "POST",
		"route":      "/api/accounts/{id}/deposits",
		"path":       "/api/accounts/42/deposits",
		"status":     float64(422),
		"bytes":      float64(5),
		"error_code": "INSUFFICIENT_BALANCE",
		"user_agent": "test-agent",
	}
	for k, v := range want {
		if lines[0][k] != v {
			t.Errorf("%s = %v, want %v", k, lines[0][k], v)
		}
	}
	if _, ok := lines[0]["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms = %v", lines[0]["duration_ms"])
	}
}

func TestRequestIDGeneratedWhenMissingOrInvalid(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger.RequestID(r.Context()) != w.Header().Get(RequestIDHeader) {
			t.Error("context and header request ids differ")
		}
	}))

	for _, incoming := range []string{"", "has spaces", strings.Repeat("a", 129), "<script>"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(RequestIDHeader, incoming)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		got := rec.Header().Get(RequestIDHeader)
		if got == "" || got == incoming {
			t.Errorf("incoming %q: got %q, want a generated id", incoming, got)
		}
	}
}

func TestLogLevels(t *testing.T) {
	buf := captureLogs(t, "info")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /ok", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("GET /boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	for _, path := range []string{"/health/live", "/ok", "/boom", "/missing"} {
		chain(mux).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
	}

	lines := logLines(t, buf)
	got := map[string]string{}
	for _, l := range lines {
		got[l["path"].(string)] = l["level"].(string) + " " + l["route"].(string)
	}
	want := map[string]string{
		"/ok":      "INFO /ok",
		"/boom":    "ERROR /boom",
		"/missing": "WARN unmatched",
	}
	if len(got) != len(want) {
		t.Errorf("logged %v, want %v (health probes must be debug)", got, want)
	}
	for path, w := range want {
		if got[path] != w {
			t.Errorf("%s: got %q, want %q", path, got[path], w)
		}
	}
}

func TestRecoveryLogsWithRequestID(t *testing.T) {
	buf := captureLogs(t, "info")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /panic", func(http.ResponseWriter, *http.Request) { panic("boom") })

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(RequestIDHeader, "trace-me")
	rec := httptest.NewRecorder()
	chain(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INTERNAL_ERROR") || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body = %s", rec.Body.String())
	}

	lines := logLines(t, buf)
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want panic + request", len(lines))
	}
	for _, l := range lines {
		if l["request_id"] != "trace-me" {
			t.Errorf("line without request id: %v", l)
		}
	}
	if lines[0]["msg"] != "panic recovered" || lines[1]["status"] != float64(500) {
		t.Errorf("unexpected lines %v", lines)
	}
}

func TestCORS(t *testing.T) {
	called := false
	handler := CORS(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/accounts", nil))
	if rec.Code != http.StatusNoContent || called {
		t.Errorf("preflight: status = %d, next called = %v", rec.Code, called)
	}
	if rec.Header().Get("Access-Control-Expose-Headers") != RequestIDHeader {
		t.Error("request id header not exposed")
	}

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if !called {
		t.Error("next handler not called for non-preflight request")
	}
}

func TestDurationMs(t *testing.T) {
	if got := durationMs(1234567 * time.Nanosecond); got != 1.234 {
		t.Errorf("durationMs = %v, want 1.234", got)
	}
}

func TestSetErrorCodeWithoutLogger(t *testing.T) {
	SetErrorCode(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "X")
}
