BINARY     = bin/api
CMD        = ./cmd/api
IMAGE_NAME = flight-price-service
IMAGE_TAG  = latest

.PHONY: dev run build test tidy \
        docker-build docker-run docker-stop docker-logs \
        env-setup help

# ── Local development ────────────────────────────────────────────────────────────

## dev: Copy .env.example → .env if missing, then run the app with hot reload
dev: env-setup tidy
	go run $(CMD)

## run: Build binary then execute it (requires .env)
run: env-setup build
	$(BINARY)

## build: Compile the binary into bin/api
build:
	@mkdir -p bin
	go build -o $(BINARY) $(CMD)

## test: Run all unit and integration tests
test:
	go test ./... -count=1

## test-v: Run tests with verbose output
test-v:
	go test ./... -count=1 -v

## test-cover: Run tests and show coverage summary
test-cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

## tidy: Tidy and verify the module graph
tidy:
	go mod tidy
	go mod verify

# ── Docker ───────────────────────────────────────────────────────────────────────

## docker-build: Build the Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

## docker-run: Build image (if needed) and run container on port 8080
docker-run: env-setup docker-build
	docker run --rm -d \
		--name $(IMAGE_NAME) \
		-p 8080:8080 \
		--env-file .env \
		$(IMAGE_NAME):$(IMAGE_TAG)
	@echo "Container started. Logs: make docker-logs  |  Stop: make docker-stop"

## docker-stop: Stop the running container
docker-stop:
	docker stop $(IMAGE_NAME) 2>/dev/null || true

## docker-logs: Tail container logs
docker-logs:
	docker logs -f $(IMAGE_NAME)

# ── Helpers ──────────────────────────────────────────────────────────────────────

## env-setup: Create .env from .env.example if it does not exist yet
env-setup:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env from .env.example — edit it before running in production."; \
	fi

## help: List all available targets with descriptions
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'

