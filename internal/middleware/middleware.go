package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/pkg/logger"
)

const RequestIDHeader = "X-Request-ID"

type annotationsKey struct{}

type annotations struct {
	errorCode string
}

// SetErrorCode attaches the API error code to the request log line.
func SetErrorCode(ctx context.Context, code string) {
	if a, ok := ctx.Value(annotationsKey{}).(*annotations); ok {
		a.errorCode = code
	}
}

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// RequestID reuses a well-formed incoming X-Request-ID or generates one.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !validRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(logger.WithRequestID(r.Context(), id)))
	})
}

// Logger must wrap the router so r.Pattern is set when it logs.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		ann := &annotations{}
		r = r.WithContext(context.WithValue(r.Context(), annotationsKey{}, ann))

		next.ServeHTTP(rec, r)

		attrs := []any{
			"method", r.Method,
			"route", Route(r),
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", durationMs(time.Since(start)),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		}
		if ann.errorCode != "" {
			attrs = append(attrs, "error_code", ann.errorCode)
		}
		slog.Log(r.Context(), level(r, rec.status), "request", attrs...)
	})
}

// Route is the matched pattern without the method, or "unmatched".
func Route(r *http.Request) string {
	if r.Pattern == "" {
		return "unmatched"
	}
	if _, path, ok := strings.Cut(r.Pattern, " "); ok {
		return path
	}
	return r.Pattern
}

// level keeps probes and scrapes out of info logs.
func level(r *http.Request, status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	case strings.HasPrefix(r.URL.Path, "/health") || r.URL.Path == "/metrics":
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func durationMs(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.ErrorContext(r.Context(), "panic recovered", "error", err, "stack", string(debug.Stack()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"success":false,"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)
		w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
