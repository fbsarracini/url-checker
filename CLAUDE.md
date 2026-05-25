# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All common workflows go through the Makefile:

- `make run` — run the server (`go run ./cmd/checker`)
- `make build` — build binary to `bin/url-checker`
- `make test` — run all tests
- `make test-race` — run tests with the race detector (use this before declaring concurrency work done)
- `make bench` — benchmark the worker pool (`go test -bench=. -benchmem ./internal/pool/...`)
- `make lint` — `go vet ./...` plus `staticcheck ./...` if installed

Single-test invocation (no Make target):

```bash
go test -race ./internal/pool -run TestRun_CancelMidFlight -v
```

## Architecture

This is a Go 1.26 HTTP service that fans out URL health checks across a bounded worker pool. The design follows "packages by capability" (Bill Kennedy) — there is intentionally no `models/`, `utils/`, or `helpers/` package.

### Composition root

`cmd/checker/main.go` is the only place where dependencies are wired together. It:

1. Calls `config.Load()` once (no `os.Getenv` lives anywhere else).
2. Builds the `slog` logger and a private Prometheus registry.
3. Constructs `checker.HTTPChecker` with its own `*http.Client` (never `http.DefaultClient`).
4. Wires the `/check` handler behind a middleware chain: `RequestID → Logging → Recover → ConcurrencyLimit`.
5. Starts two HTTP servers: the public one on `PORT`, and Prometheus on a separate `METRICS_PORT`.
6. Uses `signal.NotifyContext(SIGINT, SIGTERM)` + `srv.Shutdown(ctx)` for graceful shutdown, flipping an `atomic.Bool` that `/readyz` reads to start returning 503 during drain.

### Three layers of bounded concurrency

This is the load-bearing invariant of the service. Touch any of these and re-read the project guide first.

| Layer | Where | Mechanism |
|-------|-------|-----------|
| Server-wide | `handler.ConcurrencyLimit` middleware | `chan struct{}` semaphore sized to `MAX_CONCURRENT_REQUESTS`. When full, responds **503 immediately** — never queues. |
| Per-request | `pool.Run` | `errgroup.WithContext` + `g.SetLimit(MAX_CHECKS_PER_REQUEST)`. |
| Per-check | `checker.HTTPChecker` | `http.Client.Timeout` plus the per-request context deadline (whichever fires first). |

`MAX_URLS_PER_REQUEST` is a validation cap on payload size, not a concurrency limit.

### Cascading context

The handler derives `ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)` and passes that into `pool.Run`, which passes it to `errgroup.WithContext`, which gives each worker a derived `gctx` that flows into `http.NewRequestWithContext`. If the client disconnects, cancellation propagates all the way down to the in-flight TCP read.

`checkTimeout` is `req.TimeoutMs` clamped by `CHECK_MAX_TIMEOUT`, defaulting to `CHECK_DEFAULT_TIMEOUT`.

### Worker pool: partial results, not fail-fast

`internal/pool/pool.go` is small but the semantics are deliberate:

- Each worker writes its result into `results[i]` at a pre-allocated index. **No mutex, no channel** — workers never overlap on the same slot, and `g.Wait()` provides the happens-before for the final read.
- Workers **never return an error from `g.Go`**. Even on failure, they build a `checker.Result` with `status: "error"` or `"timeout"`. `errgroup` is used purely for `SetLimit` and coordinated cancellation, not for fail-fast — one bad URL must not abort the others.
- If you ever change a worker to `append(results, ...)` or to return a non-nil error, you've broken both invariants.

### Interface placement: `Checker` lives in the consumer

The `Checker` interface is defined in `internal/handler/interface.go` (and re-declared structurally in `internal/pool/pool.go`), **not** in `internal/checker/`. The `checker` package returns a concrete `*HTTPChecker`. This follows "accept interfaces, return structs" and means tests in `handler/` and `pool/` can mock without the production package owning the abstraction.

If you find yourself adding a `Checker` interface inside `internal/checker/`, stop — it belongs at the call site.

### Config is immutable and validated at boot

`internal/config/config.go` reads all env vars once into a `Config` struct, validates relationships (e.g. `CheckDefaultTimeout <= CheckMaxTimeout`), and returns it by value. Failures abort the process. Nothing else in the codebase should call `os.Getenv` — pass the `Config` (or individual fields) as dependencies.

### Observability

- `internal/observability/logger.go` — `slog` JSON/text handler plus a typed context key for `request_id`. Use `LoggerWithRequestID(logger, ctx)` to get a child logger with the ID attached; never reach into the context manually outside this package.
- `internal/observability/metrics.go` — Prometheus registry and a **separate HTTP server** for `/metrics` on `METRICS_PORT`. This is intentional: metrics expose internals and must not sit behind the public load balancer. Keep them off the main mux.

### Endpoints

- `POST /check` — main work endpoint; body `{urls: [...], timeout_ms: N}`.
- `GET /healthz` — liveness (always 200 while process is alive).
- `GET /readyz` — readiness; flips to 503 once `isShuttingDown` is set.
- `GET :METRICS_PORT/metrics` — Prometheus scrape target.

## Conventions to preserve

- No `os.Getenv` outside `internal/config/`.
- No global mutable state.
- Every goroutine has a `ctx` or a `WaitGroup`/`errgroup` that owns its lifetime.
- Always test concurrency code with `-race` (`make test-race`) before considering it done.
- Prefer `http.NewRequestWithContext` and explicit `*http.Client` construction; never reach for `http.DefaultClient`.

## Further reading

`url-checker-project-guide.md` (Portuguese) is the design document and contains the full rationale for every decision above, plus interview-style defenses of each tradeoff. Treat it as the source of truth when guidance here is ambiguous.
