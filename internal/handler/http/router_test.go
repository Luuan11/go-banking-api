package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/luuan11/bank-system/internal/middleware"
)

func TestRouterUnmatchedRoutesReturnJSON(t *testing.T) {
	router := NewRouter(NewOperation(&stubMover{}, &stubMover{}, stubTransferer{}, &stubRecorder{}))

	t.Run("unknown route", func(t *testing.T) {
		rec, resp := do(t, router, http.MethodGet, "/api/nope", "")
		if rec.Code != http.StatusNotFound || resp.Error == nil || resp.Error.Code != "ROUTE_NOT_FOUND" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		rec, resp := do(t, router, http.MethodGet, "/api/transfers", "")
		if rec.Code != http.StatusMethodNotAllowed || resp.Error == nil || resp.Error.Code != "METHOD_NOT_ALLOWED" {
			t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
		}
		if allow := rec.Header().Get("Allow"); allow != "POST" {
			t.Errorf("Allow = %q, want POST", allow)
		}
	})

	t.Run("error code reaches the request log", func(t *testing.T) {
		var buf bytes.Buffer
		previous := slog.Default()
		slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
		t.Cleanup(func() { slog.SetDefault(previous) })

		middleware.Logger(router).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/nope", nil))
		if !strings.Contains(buf.String(), `"error_code":"ROUTE_NOT_FOUND"`) {
			t.Errorf("log = %s", buf.String())
		}
	})

	t.Run("matched route is untouched", func(t *testing.T) {
		rec, resp := do(t, router, http.MethodPost, "/api/transfers", `{"amount":1}`)
		if rec.Code != http.StatusCreated || !resp.Success {
			t.Errorf("status = %d, resp = %+v", rec.Code, resp)
		}
	})
}
