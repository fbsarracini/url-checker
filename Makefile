BINARY  := url-checker
CMD     := ./cmd/checker

.PHONY: run build test test-race bench lint clean

run:
	go run $(CMD)

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

test-race:
	go test -race ./...

bench:
	go test -bench=. -benchmem ./internal/pool/...

lint:
	go vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

clean:
	rm -rf bin/
