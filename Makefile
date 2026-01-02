.PHONY: build clean run-bot run-backtest

# Build all binaries
build:
	go build -o bin/trade-bot ./cmd/trade-bot
	go build -o bin/backtest ./cmd/backtest

# Build only the trading bot
build-bot:
	go build -o bin/trade-bot ./cmd/trade-bot

# Build only the backtesting tool
build-backtest:
	go build -o bin/backtest ./cmd/backtest

# Clean build artifacts
clean:
	rm -rf bin/

# Run the trading bot
run-bot:
	go run ./cmd/trade-bot

# Run backtesting
run-backtest:
	go run ./cmd/backtest

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