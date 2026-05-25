package checker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Status string

const (
	StatusOK        Status = "ok"
	StatusError     Status = "error"
	StatusTimeout   Status = "timeout"
	StatusCancelled Status = "cancelled"
)

type Result struct {
	URL        string `json:"url"`
	Status     Status `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	LatencyMs  int64  `json:"latency_ms,omitempty"`
	Error      string `json:"error,omitempty"`
}

type HTTPChecker struct {
	client    *http.Client
	userAgent string
}

func New(timeout time.Duration, userAgent string) *HTTPChecker {
	transport := &http.Transport{
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}
	return &HTTPChecker{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		userAgent: userAgent,
	}
}

func (h *HTTPChecker) Check(ctx context.Context, url string) Result {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{URL: url, Status: StatusError, Error: fmt.Sprintf("invalid url: %v", err)}
	}
	req.Header.Set("User-Agent", h.userAgent)

	resp, err := h.client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) {
			return Result{URL: url, Status: StatusTimeout, Error: ErrTimeout.Error()}
		}
		return Result{URL: url, Status: StatusError, Error: fmt.Sprintf("%s: %v", ErrUnreachable.Error(), err)}
	}
	defer resp.Body.Close()

	return Result{
		URL:        url,
		Status:     StatusOK,
		HTTPStatus: resp.StatusCode,
		LatencyMs:  time.Since(start).Milliseconds(),
	}
}

func isTimeoutError(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
