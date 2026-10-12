package http

import (
	"context"
	"net/http"

	"github.com/luuan11/bank-system/internal/middleware"
)

type registrar interface {
	Register(mux *http.ServeMux)
}

type RegisterFunc func(mux *http.ServeMux)

func (f RegisterFunc) Register(mux *http.ServeMux) { f(mux) }

// Router answers unmatched routes with JSON instead of the mux plain text.
type Router struct {
	mux *http.ServeMux
}

func NewRouter(handlers ...registrar) *Router {
	mux := http.NewServeMux()
	for _, h := range handlers {
		h.Register(mux)
	}
	return &Router{mux: mux}
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, pattern := rt.mux.Handler(r); pattern == "" {
		w = &unmatchedRouteWriter{ResponseWriter: w, ctx: r.Context()}
	}
	rt.mux.ServeHTTP(w, r)
}

type unmatchedRouteWriter struct {
	http.ResponseWriter
	ctx      context.Context
	replaced bool
}

func (w *unmatchedRouteWriter) WriteHeader(status int) {
	switch status {
	case http.StatusNotFound:
		w.replace(status, "ROUTE_NOT_FOUND", "route not found")
	case http.StatusMethodNotAllowed:
		w.replace(status, "METHOD_NOT_ALLOWED", "method not allowed")
	default:
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *unmatchedRouteWriter) replace(status int, code, message string) {
	w.replaced = true
	middleware.SetErrorCode(w.ctx, code)
	Error(w.ResponseWriter, status, code, message)
}

func (w *unmatchedRouteWriter) Write(b []byte) (int, error) {
	if w.replaced {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
