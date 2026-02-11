.PHONY: build test clean run dev docker-up docker-down migrate

# Build all CLIs
build:
	go build -o bin/alpha ./cmd/alpha
	go build -o bin/beta ./cmd/beta
	go build -o bin/foundry ./cmd/foundry
	go build -o bin/omega ./cmd/omega

# Run all tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run database migrations
migrate:
	go run ./cmd/migrate up

# Start docker-compose stack
docker-up:
	docker-compose up -d

# Stop docker-compose stack
docker-down:
	docker-compose down

# Development: start stack and watch for changes
dev: docker-up
	@echo "Factory starter kit running:"
	@echo "  Postgres:   localhost:5432"
	@echo "  MinIO:      localhost:9000 (console: 9001)"
	@echo "  Prometheus: localhost:9090"
	@echo "  Grafana:    localhost:3000"
	@echo "  Loki:       localhost:3100"

# Tidy dependencies
tidy:
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:v1.63.4 golangci-lint run -v

# Install binaries to /usr/local/bin (requires sudo)
install:
	go build -ldflags="-w -s" -o /usr/local/bin/alpha ./cmd/alpha
	go build -ldflags="-w -s" -o /usr/local/bin/beta ./cmd/beta
	go build -ldflags="-w -s" -o /usr/local/bin/foundry ./cmd/foundry
	go build -ldflags="-w -s" -o /usr/local/bin/omega ./cmd/omega
	go build -ldflags="-w -s" -o /usr/local/bin/partir ./cmd/partir
	go build -ldflags="-w -s" -o /usr/local/bin/guard ./cmd/guard
	go build -ldflags="-w -s" -o /usr/local/bin/migrate ./cmd/migrate
	@echo "All binaries installed to /usr/local/bin"
