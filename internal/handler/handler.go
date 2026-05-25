package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/fbsarracini/url-checker/internal/checker"
	"github.com/fbsarracini/url-checker/internal/observability"
	"github.com/fbsarracini/url-checker/internal/pool"
)

type CheckRequest struct {
	URLs      []string `json:"urls"`
	TimeoutMs int      `json:"timeout_ms,omitempty"`
}

type CheckResponse struct {
	RequestID string           `json:"request_id"`
	Results   []checker.Result `json:"results"`
}

type CheckConfig struct {
	MaxURLs        int
	MaxChecks      int
	DefaultTimeout time.Duration
	MaxTimeout     time.Duration
}

type CheckHandler struct {
	checker             Checker
	logger              *slog.Logger
	metrics             CheckMetrics
	maxURLsPerRequest   int
	checkDefaultTimeout time.Duration
	checkMaxTimeout     time.Duration
	maxChecksPerRequest int
}

type CheckMetrics interface {
	ObserveCheck(result string)
}

func NewCheckHandler(c Checker, logger *slog.Logger, metrics CheckMetrics, cfg CheckConfig) *CheckHandler {
	return &CheckHandler{
		checker:             c,
		logger:              logger,
		metrics:             metrics,
		maxURLsPerRequest:   cfg.MaxURLs,
		checkDefaultTimeout: cfg.DefaultTimeout,
		checkMaxTimeout:     cfg.MaxTimeout,
		maxChecksPerRequest: cfg.MaxChecks,
	}
}

func (h *CheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.URLs) == 0 {
		http.Error(w, "urls must not be empty", http.StatusBadRequest)
		return
	}
	if len(req.URLs) > h.maxURLsPerRequest {
		http.Error(w, fmt.Sprintf("urls exceeds max of %d", h.maxURLsPerRequest), http.StatusBadRequest)
		return
	}

	for _, u := range req.URLs {
		if _, err := url.ParseRequestURI(u); err != nil {
			http.Error(w, fmt.Sprintf("invalid url %q: %v", u, err), http.StatusBadRequest)
			return
		}
	}

	checkTimeout := h.checkDefaultTimeout
	if req.TimeoutMs > 0 {
		requested := time.Duration(req.TimeoutMs) * time.Millisecond
		if requested > h.checkMaxTimeout {
			checkTimeout = h.checkMaxTimeout
		} else {
			checkTimeout = requested
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
	defer cancel()

	results := pool.Run(ctx, req.URLs, h.checker, h.maxChecksPerRequest)

	for _, res := range results {
		h.metrics.ObserveCheck(string(res.Status))
	}

	requestID := observability.RequestIDFromContext(r.Context())
	resp := CheckResponse{RequestID: requestID, Results: results}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}
