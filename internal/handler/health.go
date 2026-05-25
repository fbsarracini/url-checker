package handler

import (
	"net/http"
	"sync/atomic"
)

type HealthHandler struct {
	isShuttingDown *atomic.Bool
}

func NewHealthHandler(isShuttingDown *atomic.Bool) *HealthHandler {
	return &HealthHandler{isShuttingDown: isShuttingDown}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	if h.isShuttingDown.Load() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}
