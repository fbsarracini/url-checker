# url-checker

HTTP service that checks the health of a list of URLs using a bounded concurrent worker pool.

## Quick start

```bash
go run ./cmd/checker
```

## API

### POST /check

```json
{
  "urls": ["https://example.com", "https://google.com"],
  "timeout_ms": 2000
}
```

Response:

```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "results": [
    {"url": "https://example.com", "status": "ok", "http_status": 200, "latency_ms": 142},
    {"url": "https://google.com", "status": "timeout", "error": "checker: timeout"}
  ]
}
```

### GET /healthz — liveness
### GET /readyz — readiness (503 during shutdown)
### GET :9090/metrics — Prometheus metrics

## Make targets

| Target | Description |
|--------|-------------|
| `make run` | Run the server |
| `make test` | Run all tests |
| `make test-race` | Run tests with race detector |
| `make bench` | Benchmark the worker pool |
| `make lint` | Run go vet + staticcheck |
| `make build` | Build binary to bin/ |

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout |
| `READ_TIMEOUT` | `5s` | HTTP read timeout |
| `WRITE_TIMEOUT` | `30s` | HTTP write timeout |
| `IDLE_TIMEOUT` | `120s` | HTTP idle timeout |
| `MAX_CONCURRENT_REQUESTS` | `100` | Global concurrency semaphore |
| `MAX_CHECKS_PER_REQUEST` | `50` | Worker pool limit per request |
| `MAX_URLS_PER_REQUEST` | `200` | Max URLs per payload |
| `CHECK_DEFAULT_TIMEOUT` | `3s` | Default check timeout |
| `CHECK_MAX_TIMEOUT` | `10s` | Max check timeout |
| `LOG_LEVEL` | `info` | debug\|info\|warn\|error |
| `LOG_FORMAT` | `json` | json\|text |
