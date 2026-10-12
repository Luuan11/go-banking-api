package http

import (
	"context"
	"net/http"
	"time"
)

type pinger interface {
	PingContext(ctx context.Context) error
}

type Health struct {
	db pinger
}

func NewHealth(db pinger) *Health {
	return &Health{db: db}
}

func (h *Health) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Check)
}

func (h *Health) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		Error(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "database is unreachable")
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
