.PHONY: build clean run-bot

# Build all binaries
build:
	go build -o bin/trade-bot ./cmd/trade-bot

# Clean build artifacts
clean:
	rm -rf bin/

# Run the trading bot
run-bot:
	go run ./cmd/trade-bot

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Download dependencies
vendor:
	go mod vendor