BINARY=bin/api
CMD=./cmd/api

.PHONY: dev run build test tidy docker-build docker-run

dev:
	go run $(CMD)

run:
	$(BINARY)

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./...

tidy:
	go mod tidy

docker-build:
	docker build -t flight-price-service .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env flight-price-service
