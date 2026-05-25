package checker_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fbsarracini/url-checker/internal/checker"
)

func TestCheck_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := checker.New(5*time.Second, "test-agent")
	res := c.Check(context.Background(), srv.URL)

	if res.Status != checker.StatusOK {
		t.Fatalf("expected ok, got %s: %s", res.Status, res.Error)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("expected 200, got %d", res.HTTPStatus)
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := checker.New(50*time.Millisecond, "test-agent")
	res := c.Check(context.Background(), srv.URL)

	if res.Status != checker.StatusTimeout {
		t.Fatalf("expected timeout, got %s", res.Status)
	}
}

func TestCheck_Unreachable(t *testing.T) {
	c := checker.New(2*time.Second, "test-agent")
	res := c.Check(context.Background(), "http://127.0.0.1:1")

	if res.Status != checker.StatusError {
		t.Fatalf("expected error, got %s", res.Status)
	}
}

func TestCheck_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	c := checker.New(5*time.Second, "test-agent")
	res := c.Check(ctx, srv.URL)

	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("expected fast cancellation, took %s", elapsed)
	}
	if res.Status == checker.StatusOK {
		t.Fatal("expected non-ok status after cancellation")
	}
}

func TestCheck_InvalidURL(t *testing.T) {
	c := checker.New(5*time.Second, "test-agent")
	res := c.Check(context.Background(), "://invalid")

	if res.Status != checker.StatusError {
		t.Fatalf("expected error, got %s", res.Status)
	}
}
