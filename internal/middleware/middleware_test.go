package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecovery(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "INTERNAL_ERROR") || strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestLoggerKeepsResponse(t *testing.T) {
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("short and stout"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot || rec.Body.String() != "short and stout" {
		t.Errorf("status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestStatusRecorder(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	rec.WriteHeader(http.StatusNotFound)
	if rec.status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.status)
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
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing Access-Control-Allow-Origin")
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	if !called {
		t.Error("next handler not called for non-preflight request")
	}
}
