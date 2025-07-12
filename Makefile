.PHONY: build run test clean dev deps

# Binary name
BINARY_NAME=agent-server

# Build the application
build:
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run the application
run: build
	./bin/$(BINARY_NAME)

# Run with config file
run-config: build
	./bin/$(BINARY_NAME) -config configs/config.yaml

# Run in development mode
dev:
	go run cmd/server/main.go -config configs/config.yaml

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./internal/...

# Run unit tests only
test-unit:
	go test -v ./internal/models/... ./internal/context/... ./internal/llm/... ./internal/api/handlers/...

# Run integration tests
test-integration:
	go test -v ./test/...

# Run all tests
test-all:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./internal/...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with coverage (all)
test-coverage-all:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	go clean
	rm -f bin/$(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -rf data/

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Create data directory
init-dirs:
	mkdir -p data bin

# Build for multiple platforms
build-all: init-dirs
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 cmd/server/main.go

# Docker build
docker-build:
	docker build -t agent-server .

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/app/data agent-server

# Development setup
setup: deps init-dirs
	@echo "Setup complete. Run 'make dev' to start the server."