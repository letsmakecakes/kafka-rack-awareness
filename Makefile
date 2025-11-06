# Makefile for Kafka Rack Awareness Testing

.PHONY: help start stop test test-verbose test-race clean logs status

# Default target
help:
	@echo "Kafka Rack Awareness Test Suite"
	@echo ""
	@echo "Available targets:"
	@echo "  make start        - Start Kafka cluster"
	@echo "  make stop         - Stop Kafka cluster"
	@echo "  make test         - Run all tests"
	@echo "  make test-verbose - Run tests with verbose output"
	@echo "  make test-race    - Run tests with race detection"
	@echo "  make clean        - Clean up and remove all containers"
	@echo "  make logs         - View Kafka cluster logs"
	@echo "  make status       - Check cluster status"
	@echo "  make init         - Initialize Go modules"

# Start Kafka cluster
start:
	@echo "Starting Kafka cluster..."
	docker-compose up -d
	@echo "Waiting for Kafka to be ready..."
	@sleep 30
	@echo "Kafka cluster is ready!"

# Stop Kafka cluster
stop:
	@echo "Stopping Kafka cluster..."
	docker-compose down

# Clean up everything
clean:
	@echo "Cleaning up Kafka cluster and volumes..."
	docker-compose down -v
	@echo "Cleanup complete!"

# Initialize Go modules
init:
	@echo "Initializing Go modules..."
	go mod tidy
	@echo "Go modules initialized!"

# Run all tests
test: start
	@echo "Running tests..."
	go test -timeout 10m
	@$(MAKE) stop

# Run tests with verbose output
test-verbose: start
	@echo "Running tests with verbose output..."
	go test -v -timeout 10m
	@$(MAKE) stop

# Run tests with race detection
test-race: start
	@echo "Running tests with race detection..."
	go test -v -race -timeout 10m
	@$(MAKE) stop

# Run specific test
test-one: start
	@echo "Running test: $(TEST)"
	go test -v -run $(TEST) -timeout 5m
	@$(MAKE) stop

# View logs
logs:
	docker-compose logs -f

# Check cluster status
status:
	@echo "Checking cluster status..."
	docker-compose ps
	@echo ""
	@echo "Broker connectivity:"
	@for port in 9092 9093 9094; do \
		echo -n "Port $$port: "; \
		nc -z localhost $$port && echo "OK" || echo "FAILED"; \
	done

# Quick test without cleanup (cluster stays running)
quick-test:
	@echo "Running quick test (cluster remains running)..."
	go test -v -timeout 10m

# Run coverage
coverage: start
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out -timeout 10m
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@$(MAKE) stop

# Benchmark tests
bench: start
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem -timeout 15m
	@$(MAKE) stop

# Lint code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# All - initialize, start, test, and clean
all: init start test clean
	@echo "Complete test cycle finished!"
