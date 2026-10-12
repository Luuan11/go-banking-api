package http

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"
)

type pinger interface {
	PingContext(ctx context.Context) error
}

type Health struct {
	db       pinger
	draining atomic.Bool
}

func NewHealth(db pinger) *Health {
	return &Health{db: db}
}

func (h *Health) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health/live", h.Live)
	mux.HandleFunc("GET /health/ready", h.Ready)
}

// Drain fails readiness so load balancers stop routing before shutdown.
func (h *Health) Drain() {
	h.draining.Store(true)
}

// Live only says the process is serving; it never checks dependencies.
func (h *Health) Live(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Health) Ready(w http.ResponseWriter, r *http.Request) {
	if h.draining.Load() {
		Error(w, http.StatusServiceUnavailable, "SHUTTING_DOWN", "server is shutting down")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		Error(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unreachable")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
