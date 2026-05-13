# ── Build stage ─────────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# Install git (needed by `go mod download` for VCS-stamped modules)
RUN apk add --no-cache git

WORKDIR /app

# Cache dependency downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a statically linked binary with no debug info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/api \
    ./cmd/api

# ── Runtime stage ────────────────────────────────────────────────────────────────
FROM scratch

# Copy CA certificates so HTTPS calls to real provider APIs work
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/bin/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
