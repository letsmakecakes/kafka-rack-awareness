# Kafka Rack Awareness Testing Suite

This project provides comprehensive testing for Kafka rack awareness using the confluent-kafka-go library with Docker.

## Overview

Rack awareness in Kafka helps distribute replicas across different racks (availability zones) to improve fault tolerance. This test suite validates that rack awareness is properly configured and functioning across various scenarios.

## Prerequisites

- **Docker** (with Docker Compose)
- **Go** 1.21 or higher
- **Git Bash** or WSL2 (for Windows users)
- At least 4GB of available RAM for Docker containers

## Project Structure

```
kafka-rack-awareness/
├── docker-compose.yml          # Kafka cluster with 3 brokers in different racks
├── go.mod                      # Go module dependencies
├── rack_awareness_test.go      # Comprehensive test suite (20 test cases)
├── run_tests.sh               # Automated test runner script (Linux/Mac)
├── run_tests.bat              # Automated test runner script (Windows)
├── Makefile                   # Make commands for easy testing
└── README.md                  # This file
```

## Kafka Cluster Configuration

The Docker Compose setup creates:
- **3 Kafka brokers** (one per rack):
  - Broker 1: rack-a (port 9092)
  - Broker 2: rack-b (port 9093)
  - Broker 3: rack-c (port 9094)
- **1 Zookeeper instance**
- **Replication Factor**: 3
- **Min In-Sync Replicas**: 2

## Test Cases Covered (20 Tests)

### 1. **Broker Configuration Tests**
- ✅ `TestBrokerRackConfiguration` - Verify broker rack configuration
- ✅ Confirm each broker is assigned to the correct rack

### 2. **Replica Distribution Tests**
- ✅ `TestReplicaDistributionAcrossRacks` - Verify replicas are distributed across different racks
- ✅ `TestReplicationFactorLessThanRacks` - Test replication factor less than number of racks
- ✅ `TestSinglePartitionTopicRackAwareness` - Single partition topic rack awareness
- ✅ `TestHighPartitionCountTopic` - High partition count topics (30 partitions)

### 3. **In-Sync Replicas (ISR) Tests**
- ✅ `TestISRAcrossRacks` - Verify ISR spans multiple racks
- ✅ `TestMinISRWithRackAwareness` - Test min.insync.replicas with rack awareness

### 4. **Producer Tests**
- ✅ `TestProducerRackAwareness` - Producer with rack awareness (client.rack)
- ✅ `TestProducerIdempotenceWithRackAwareness` - Idempotent producer with rack awareness
- ✅ `TestTransactionalProducerWithRackAwareness` - Transactional producer with rack awareness
- ✅ `TestConcurrentProducersWithDifferentRacks` - Concurrent producers with different rack configurations
- ✅ `TestRackAwarenessWithCustomPartitioning` - Custom partitioner with rack awareness
- ✅ `TestLargeMessagesWithRackAwareness` - Large message handling with rack awareness (1MB messages)
- ✅ `TestRackAwarenessWithCompression` - Compression with rack awareness (gzip, snappy, lz4, zstd)

### 5. **Consumer Tests**
- ✅ `TestConsumerRackAwareness` - Consumer with rack awareness (fetch from follower)
- ✅ `TestPreferredReadReplica` - Preferred read replica functionality
- ✅ `TestConsumerGroupRebalancingWithRackAwareness` - Consumer group rebalancing with rack awareness

### 6. **Leader Distribution Tests**
- ✅ `TestLeaderDistributionAcrossRacks` - Verify leaders are distributed across racks

### 7. **Metadata Consistency Tests**
- ✅ `TestMetadataConsistencyAcrossRacks` - Verify metadata consistency across rack-aware clients

### 8. **Edge Cases**
- ✅ `TestEmptyRackConfiguration` - Empty rack configuration (client without client.rack)

## Quick Start

### Option 1: Using Windows Batch Script

```cmd
run_tests.bat
```

### Option 2: Using Bash Script (Git Bash/WSL)

```bash
chmod +x run_tests.sh
./run_tests.sh
```

### Option 3: Using Make (Linux/Mac/WSL)

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run specific test
make test-one TEST=TestBrokerRackConfiguration

# See all available commands
make help
```

### Option 4: Manual Steps

1. **Start the Kafka cluster:**
```bash
docker-compose up -d
```

2. **Wait for Kafka to be ready (about 30 seconds):**
```bash
# Windows
timeout /t 30

# Linux/Mac
sleep 30
```

3. **Initialize Go modules:**
```bash
go mod tidy
```

4. **Run the tests:**
```bash
# Run all tests
go test -v -timeout 10m

# Run specific test
go test -v -run TestBrokerRackConfiguration

# Run tests with race detection
go test -v -race -timeout 10m
```

5. **Stop the cluster:**
```bash
docker-compose down -v
```

## Running Individual Tests

```bash
# Test broker rack configuration
go test -v -run TestBrokerRackConfiguration

# Test replica distribution
go test -v -run TestReplicaDistributionAcrossRacks

# Test producer rack awareness
go test -v -run TestProducerRackAwareness

# Test consumer rack awareness
go test -v -run TestConsumerRackAwareness

# Test ISR across racks
go test -v -run TestISRAcrossRacks

# Test transactional producer
go test -v -run TestTransactionalProducerWithRackAwareness

# Test large messages
go test -v -run TestLargeMessagesWithRackAwareness

# Test compression
go test -v -run TestRackAwarenessWithCompression
```

## Test Output Example

```
=== RUN   TestBrokerRackConfiguration
    rack_awareness_test.go:92: Broker 1 is in rack: rack-a
    rack_awareness_test.go:92: Broker 2 is in rack: rack-b
    rack_awareness_test.go:92: Broker 3 is in rack: rack-c
--- PASS: TestBrokerRackConfiguration (0.15s)

=== RUN   TestReplicaDistributionAcrossRacks
    rack_awareness_test.go:125: Partition 0 replicas: [1 2 3], racks: [rack-a rack-b rack-c]
    rack_awareness_test.go:125: Partition 1 replicas: [2 3 1], racks: [rack-a rack-b rack-c]
    rack_awareness_test.go:125: Partition 2 replicas: [3 1 2], racks: [rack-a rack-b rack-c]
--- PASS: TestReplicaDistributionAcrossRacks (2.34s)
```

## Monitoring the Kafka Cluster

### View Container Logs

```bash
# All containers
docker-compose logs -f

# Specific broker
docker-compose logs -f kafka-broker-1

# Zookeeper
docker-compose logs -f zookeeper
```

### Check Broker Status

```bash
# List running containers
docker-compose ps

# Check cluster status (using Make)
make status
```

## Configuration Details

### Producer Configuration Options

```go
producer := kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092,localhost:9093,localhost:9094",
    "client.rack":       "rack-a",  // Enable rack awareness
    "acks":              "all",     // Wait for all replicas
    "enable.idempotence": true,     // Idempotent producer
    "compression.type":  "gzip",    // Compression
    "transactional.id":  "my-txn",  // Transactional producer
})
```

### Consumer Configuration Options

```go
consumer := kafka.NewConsumer(&kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092,localhost:9093,localhost:9094",
    "group.id":          "my-group",
    "client.rack":       "rack-a",  // Enable fetch from follower
    "auto.offset.reset": "earliest",
})
```

## Troubleshooting

### Issue: Containers fail to start

**Solution:**
```bash
# Check Docker resources
docker system df

# Clean up unused resources
docker system prune -a

# Increase Docker memory limit to at least 4GB
```

### Issue: Tests timeout

**Solution:**
```bash
# Increase test timeout
go test -v -timeout 15m

# Check if Kafka is ready
docker-compose logs kafka-broker-1 | grep "started"

# Wait longer before running tests
sleep 45
```

### Issue: Connection refused errors

**Solution:**
```bash
# Ensure all brokers are running
docker-compose ps

# Check port availability (Windows)
netstat -an | findstr "9092 9093 9094"

# Check port availability (Linux/Mac)
netstat -an | grep -E "9092|9093|9094"
```

### Issue: Tests fail with "Leader not available"

**Solution:**
```bash
# Wait for leader election to complete
sleep 10

# Check broker logs
docker-compose logs kafka-broker-1 | grep -i leader

# Restart the cluster
docker-compose down -v
docker-compose up -d
sleep 30
```

## Performance Considerations

### Test Execution Time
- Full test suite: ~5-8 minutes
- Individual tests: 10-60 seconds each
- Cluster startup: ~20-30 seconds

### Resource Usage
- Memory: ~3-4 GB
- CPU: Moderate during tests
- Disk: ~1 GB for Docker images

## Best Practices

1. **Always wait for cluster readiness** before running tests (30+ seconds)
2. **Clean up between test runs** to avoid conflicts (`docker-compose down -v`)
3. **Monitor Docker resources** to prevent OOM errors
4. **Use verbose output** (`-v`) for debugging
5. **Run tests sequentially** to avoid resource contention

## Make Commands Reference

```bash
make help          # Show all available commands
make start         # Start Kafka cluster
make stop          # Stop Kafka cluster
make test          # Run all tests (with auto cleanup)
make test-verbose  # Run tests with verbose output
make test-race     # Run tests with race detection
make test-one TEST=<TestName>  # Run specific test
make quick-test    # Run tests without cleanup
make clean         # Clean up and remove all containers
make logs          # View Kafka cluster logs
make status        # Check cluster status
make init          # Initialize Go modules
make coverage      # Run tests with coverage report
```

## Advanced Usage

### Custom Test Scenarios

Create your own tests by following the pattern:

```go
func TestMyCustomRackScenario(t *testing.T) {
    adminClient := createAdminClient(t)
    defer adminClient.Close()
    
    topicName := fmt.Sprintf("test-custom-%d", time.Now().Unix())
    defer deleteTopic(t, adminClient, topicName)
    
    // Your test logic here
}
```

### Coverage Report

```bash
# Generate coverage report
make coverage

# View coverage in browser
# Opens coverage.html automatically
```

### Integration with CI/CD

```yaml
# Example GitHub Actions workflow
name: Kafka Rack Awareness Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Start Kafka
      run: docker-compose up -d
      
    - name: Wait for Kafka
      run: sleep 30
      
    - name: Run Tests
      run: go test -v -timeout 10m
      
    - name: Cleanup
      run: docker-compose down -v
      if: always()
```

## Key Features of the Test Suite

1. **Comprehensive Coverage**: 20 different test scenarios covering all aspects of rack awareness
2. **Docker-based**: Easy setup with Docker Compose
3. **Cross-platform**: Works on Windows, Linux, and macOS
4. **Well-documented**: Detailed comments and logging in tests
5. **Helper Functions**: Reusable functions for common operations
6. **Automated Scripts**: Bash and batch scripts for easy execution
7. **Makefile Support**: Quick commands for common tasks
8. **Production-ready**: Tests real-world scenarios including transactions, compression, and large messages

## References

- [Confluent Kafka Go Documentation](https://docs.confluent.io/kafka-clients/go/current/overview.html)
- [Apache Kafka Rack Awareness](https://kafka.apache.org/documentation/#basic_ops_racks)
- [Kafka Replication](https://kafka.apache.org/documentation/#replication)
- [Kafka Consumer Fetch from Follower](https://kafka.apache.org/documentation/#design_followerreplication)

## License

MIT License - Feel free to use and modify for your needs.

## Support

For issues or questions:
1. Check the troubleshooting section
2. Review Docker and Kafka logs (`docker-compose logs`)
3. Verify your environment meets prerequisites
4. Ensure adequate Docker resources (4GB+ RAM)
