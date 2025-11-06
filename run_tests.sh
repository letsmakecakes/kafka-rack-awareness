#!/bin/bash

# Kafka Rack Awareness Test Runner Script

set -e

echo "=== Kafka Rack Awareness Test Suite ==="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_status "$RED" "Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

print_status "$GREEN" "✓ Docker is running"

# Step 1: Clean up any existing containers
print_status "$YELLOW" "Cleaning up existing containers..."
docker-compose down -v 2>/dev/null || true
print_status "$GREEN" "✓ Cleanup complete"

# Step 2: Start Kafka cluster
print_status "$YELLOW" "Starting Kafka cluster with rack awareness..."
docker-compose up -d

# Step 3: Wait for Kafka to be ready
print_status "$YELLOW" "Waiting for Kafka cluster to be ready..."
sleep 20

# Check if all containers are running
if [ $(docker-compose ps -q | wc -l) -eq 4 ]; then
    print_status "$GREEN" "✓ All Kafka brokers are running"
else
    print_status "$RED" "Error: Not all containers are running"
    docker-compose ps
    exit 1
fi

# Step 4: Verify broker rack configuration
print_status "$YELLOW" "Verifying broker rack configuration..."
for port in 9092 9093 9094; do
    if nc -z localhost $port 2>/dev/null; then
        print_status "$GREEN" "✓ Broker on port $port is accessible"
    else
        print_status "$YELLOW" "  Waiting for broker on port $port..."
        sleep 5
    fi
done

# Step 5: Initialize Go modules
print_status "$YELLOW" "Initializing Go modules..."
go mod tidy
print_status "$GREEN" "✓ Go modules initialized"

# Step 6: Run tests
print_status "$YELLOW" "Running Kafka Rack Awareness Tests..."
echo ""

# Run all tests with verbose output
if go test -v -timeout 10m ./... ; then
    print_status "$GREEN" "========================================="
    print_status "$GREEN" "✓ All tests passed successfully!"
    print_status "$GREEN" "========================================="
else
    print_status "$RED" "========================================="
    print_status "$RED" "✗ Some tests failed"
    print_status "$RED" "========================================="
    exit 1
fi

# Step 7: Cleanup option
echo ""
read -p "Do you want to stop the Kafka cluster? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_status "$YELLOW" "Stopping Kafka cluster..."
    docker-compose down -v
    print_status "$GREEN" "✓ Kafka cluster stopped"
else
    print_status "$YELLOW" "Kafka cluster is still running. Use 'docker-compose down -v' to stop it."
fi

echo ""
print_status "$GREEN" "Test run complete!"
