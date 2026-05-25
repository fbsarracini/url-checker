FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /url-checker ./cmd/checker

FROM scratch
COPY --from=builder /url-checker /url-checker
EXPOSE 8080 9090
ENTRYPOINT ["/url-checker"]
