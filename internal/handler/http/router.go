package http

import "net/http"

type registrar interface {
	Register(mux *http.ServeMux)
}

// Router wraps http.ServeMux so that requests matching no route get the same
// JSON error envelope as the rest of the API instead of the mux's plain text.
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
		w = &unmatchedRouteWriter{ResponseWriter: w}
	}
	rt.mux.ServeHTTP(w, r)
}

// unmatchedRouteWriter replaces the mux's plain-text 404 and 405 responses
// with JSON ones. Headers set by the mux, like Allow on a 405, are kept.
type unmatchedRouteWriter struct {
	http.ResponseWriter
	replaced bool
}

func (w *unmatchedRouteWriter) WriteHeader(status int) {
	switch status {
	case http.StatusNotFound:
		w.replaced = true
		Error(w.ResponseWriter, status, "ROUTE_NOT_FOUND", "route not found")
	case http.StatusMethodNotAllowed:
		w.replaced = true
		Error(w.ResponseWriter, status, "METHOD_NOT_ALLOWED", "method not allowed")
	default:
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *unmatchedRouteWriter) Write(b []byte) (int, error) {
	if w.replaced {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
