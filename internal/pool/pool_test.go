package pool_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fbsarracini/url-checker/internal/checker"
	"github.com/fbsarracini/url-checker/internal/pool"
)

type mockChecker struct {
	delay  time.Duration
	result checker.Result
}

func (m *mockChecker) Check(ctx context.Context, url string) checker.Result {
	select {
	case <-ctx.Done():
		return checker.Result{URL: url, Status: checker.StatusCancelled}
	case <-time.After(m.delay):
		r := m.result
		r.URL = url
		return r
	}
}

func TestRun_AllURLs(t *testing.T) {
	urls := make([]string, 100)
	for i := range urls {
		urls[i] = "http://example.com"
	}

	c := &mockChecker{delay: 1 * time.Millisecond, result: checker.Result{Status: checker.StatusOK}}
	results := pool.Run(context.Background(), urls, c, 10)

	if len(results) != 100 {
		t.Fatalf("expected 100 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != checker.StatusOK {
			t.Fatalf("expected ok, got %s", r.Status)
		}
	}
}

func TestRun_CancelMidway(t *testing.T) {
	urls := make([]string, 50)
	for i := range urls {
		urls[i] = "http://example.com"
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &mockChecker{delay: 100 * time.Millisecond, result: checker.Result{Status: checker.StatusOK}}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	pool.Run(ctx, urls, c, 5)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("pool did not cancel fast enough, took %s", elapsed)
	}
}

func TestRun_EmptyURLs(t *testing.T) {
	c := &mockChecker{result: checker.Result{Status: checker.StatusOK}}
	results := pool.Run(context.Background(), []string{}, c, 10)

	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestRun_NoDataRace(t *testing.T) {
	urls := make([]string, 100)
	for i := range urls {
		urls[i] = "http://example.com"
	}

	var calls atomic.Int64
	c := &counterChecker{calls: &calls}
	results := pool.Run(context.Background(), urls, c, 10)

	if int(calls.Load()) != 100 {
		t.Fatalf("expected 100 calls, got %d", calls.Load())
	}
	if len(results) != 100 {
		t.Fatalf("expected 100 results, got %d", len(results))
	}
}

type counterChecker struct {
	calls *atomic.Int64
}

func (c *counterChecker) Check(ctx context.Context, url string) checker.Result {
	c.calls.Add(1)
	return checker.Result{URL: url, Status: "ok"}
}
