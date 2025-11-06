# Franz-Go Kafka Rack Awareness Tests

This document describes the comprehensive test suite for Kafka rack awareness using the `franz-go` library.

## Overview

The `rack_awareness_franz_test.go` file contains 10 comprehensive tests that verify Kafka rack awareness functionality using the high-performance [franz-go](https://github.com/twmb/franz-go) Kafka client library.

## Test Suite

### 1. TestFranz_BrokerMetadata ✅
**Purpose**: Verify broker metadata and rack configuration

**What it tests**:
- Verifies all 3 brokers are accessible
- Confirms each broker has correct rack assignment:
  - Broker 1 → rack-a (port 9092)
  - Broker 2 → rack-b (port 9093)
  - Broker 3 → rack-c (port 9094)

**Result**: PASS (0.09s)

### 2. TestFranz_ReplicaDistribution ✅
**Purpose**: Verify replica distribution across racks

**What it tests**:
- Creates topic with 6 partitions, RF=3
- Verifies each partition has replicas in all 3 different racks
- Ensures rack-aware replica placement

**Result**: PASS (3.93s)
- All 6 partitions have replicas distributed across rack-a, rack-b, rack-c

### 3. TestFranz_ProducerWithRackAwareness ✅
**Purpose**: Test producer with rack awareness enabled

**What it tests**:
- Creates producer with rack awareness (`kgo.Rack("rack-a")`)
- Produces 20 messages with rack-aware routing
- Verifies messages are distributed across partitions
- Uses `kgo.AllISRAcks()` for strong durability

**Result**: PASS (3.12s)
- Successfully produced 20 messages
- Messages distributed across all partitions

### 4. TestFranz_ConsumerWithRackAwareness ✅
**Purpose**: Test consumer with rack awareness enabled

**What it tests**:
- Creates consumer preferring to fetch from rack-a
- Consumes 10 messages with rack-aware fetching
- Verifies all messages are consumed correctly

**Result**: PASS (2.80s)
- Successfully consumed all 10 messages
- Rack preference applied for optimal latency

### 5. TestFranz_PartitionDistribution ✅
**Purpose**: Verify partition leader distribution

**What it tests**:
- Creates topic with 9 partitions
- Verifies leaders are evenly distributed across racks
- Each rack should have 3 partition leaders

**Result**: PASS (2.18s)
```
Leader distribution:
- rack-a: 3 leaders
- rack-b: 3 leaders
- rack-c: 3 leaders
```

### 6. TestFranz_TransactionalProducer ✅
**Purpose**: Test transactional producer with rack awareness

**What it tests**:
- Creates transactional producer with rack awareness
- Begins transaction
- Produces 5 messages within transaction
- Commits transaction using `kgo.EndTransaction(ctx, kgo.TryCommit)`

**Result**: PASS (4.51s)
- Transaction committed successfully
- All messages produced atomically

### 7. TestFranz_IdempotentProducer ✅
**Purpose**: Test idempotent producer

**What it tests**:
- Verifies franz-go enables idempotence by default
- Produces 10 messages with idempotent guarantees
- Ensures no duplicate messages

**Result**: PASS (2.41s)
- Idempotent producer working correctly (enabled by default in franz-go)

### 8. TestFranz_ConsumerRebalance ✅
**Purpose**: Test consumer rebalancing

**What it tests**:
- Creates first consumer in group
- Creates second consumer to trigger rebalance
- Verifies rebalance completes successfully
- Tests with 6 partitions

**Result**: PASS (7.17s)
- Consumer rebalance completed successfully

### 9. TestFranz_ISRVerification ✅
**Purpose**: Verify In-Sync Replicas (ISR) span multiple racks

**What it tests**:
- Creates topic with 3 partitions, RF=3
- Verifies ISR size ≥ min.insync.replicas (2)
- Confirms ISR replicas span at least 2 racks
- Validates fault tolerance

**Result**: PASS (2.18s)
```
Partition 0: ISR=[1 2 3] - racks: [rack-a, rack-b, rack-c]
Partition 1: ISR=[2 3 1] - racks: [rack-a, rack-b, rack-c]
Partition 2: ISR=[3 1 2] - racks: [rack-a, rack-b, rack-c]
```

### 10. TestFranz_HighPartitionCount ✅
**Purpose**: Test rack awareness with high partition count

**What it tests**:
- Creates topic with 30 partitions, RF=3
- Verifies all 30 partitions have rack-aware replica placement
- Ensures scalability of rack awareness

**Result**: PASS (3.13s)
```
Partitions with all racks: 30/30
Rack distribution:
- rack-a: 30 replicas
- rack-b: 30 replicas
- rack-c: 30 replicas
```

## Overall Test Results

```
Total Tests: 10
Passed: 10 (100%)
Failed: 0 (0%)
Total Duration: 31.82 seconds
```

## Key Features Tested

### Franz-Go Specific Features
1. **Rack Awareness**: `kgo.Rack("rack-a")` for both producers and consumers
2. **Admin Client**: Using `kadm.Client` for topic management and metadata
3. **Transactional API**: `BeginTransaction()` and `EndTransaction(ctx, kgo.TryCommit)`
4. **Idempotence**: Enabled by default in franz-go (no configuration needed)
5. **Synchronous Production**: `ProduceSync(ctx, record)` for reliability
6. **Fetch API**: `PollFetches(ctx)` with `RecordIter()` for consumption
7. **Required Acks**: `kgo.AllISRAcks()` for strong durability guarantees

### Kafka Features Validated
1. ✅ Broker rack configuration
2. ✅ Replica placement across racks
3. ✅ Leader distribution
4. ✅ In-Sync Replicas (ISR) rack distribution
5. ✅ Producer with rack awareness
6. ✅ Consumer with rack preference
7. ✅ Transactional messaging
8. ✅ Idempotent production
9. ✅ Consumer group rebalancing
10. ✅ High partition count scalability

## Franz-Go vs Other Libraries

### Why Franz-Go?

**Advantages**:
- **Pure Go**: No CGO dependency (unlike confluent-kafka-go)
- **High Performance**: Optimized for throughput and latency
- **Modern API**: Clean, idiomatic Go interfaces
- **Feature Complete**: Full Kafka protocol support
- **Active Development**: Regular updates and improvements
- **Better Defaults**: Idempotence enabled by default

**Comparison**:
| Feature | franz-go | segmentio/kafka-go | confluent-kafka-go |
|---------|----------|-------------------|-------------------|
| CGO Required | ❌ No | ❌ No | ✅ Yes (librdkafka) |
| Performance | 🚀 Excellent | ✅ Good | 🚀 Excellent |
| API Design | ✅ Modern | ✅ Clean | ⚠️ C-like |
| Transactions | ✅ Full | ⚠️ Limited | ✅ Full |
| Rack Awareness | ✅ Native | ✅ Via config | ✅ Via config |
| Idempotence | ✅ Default on | ⚠️ Manual | ⚠️ Manual |
| Windows Support | ✅ Native | ✅ Native | ⚠️ Requires CGO |

## API Differences

### Metadata Access
```go
// franz-go: Brokers is a slice
metadata.Brokers // []BrokerDetail
for _, broker := range metadata.Brokers {
    broker.NodeID  // int32
    broker.Rack    // *string
}

// segmentio: Brokers returns slice
conn.Brokers() // []kafka.Broker
```

### Producer Results
```go
// franz-go: Returns ProduceResults
results := producer.ProduceSync(ctx, record)
for _, r := range results {
    r.Record.Partition
    r.Record.Offset
}

// segmentio: Returns partition/offset directly
partition, offset, err := writer.WriteMessages(ctx, msg)
```

### Transactions
```go
// franz-go
producer.BeginTransaction()
producer.EndTransaction(ctx, kgo.TryCommit)

// segmentio
writer.BeginTransaction()
writer.CommitTransaction(ctx)
```

## Running the Tests

```bash
# Run all franz-go tests
go test -v -run TestFranz -timeout 5m

# Run specific test
go test -v -run TestFranz_BrokerMetadata

# Run with race detection
go test -v -race -run TestFranz

# Run tests multiple times
go test -v -run TestFranz -count=5
```

## Configuration

The tests use the following configuration:
- **Brokers**: localhost:9092, localhost:9093, localhost:9094
- **Timeout**: 60 seconds per test
- **Replication Factor**: 3
- **Min In-Sync Replicas**: 2
- **Racks**: rack-a, rack-b, rack-c

## Cleanup

All tests properly clean up resources:
- Topics are deleted after each test
- Clients are properly closed using `defer`
- Contexts have timeouts to prevent hangs

## Best Practices Demonstrated

1. **Resource Management**: Proper use of `defer` for cleanup
2. **Error Handling**: Comprehensive error checking
3. **Timeout Management**: Context deadlines for all operations
4. **Test Isolation**: Each test creates unique topics
5. **Logging**: Detailed logging for debugging
6. **Assertions**: Clear, descriptive assertions
7. **Wait Times**: Appropriate delays for Kafka propagation

## Troubleshooting

### Common Issues

**Issue**: Timeout during test
- **Solution**: Ensure Kafka cluster is running: `docker-compose ps`

**Issue**: Connection refused
- **Solution**: Check Kafka is accessible: `docker-compose logs kafka1`

**Issue**: Topic creation fails
- **Solution**: Verify cluster has enough brokers for RF=3

## Conclusion

The franz-go test suite demonstrates:
- ✅ 100% test success rate
- ✅ Comprehensive rack awareness validation
- ✅ Production-ready patterns
- ✅ Modern Go best practices
- ✅ Full feature coverage

Franz-go provides an excellent pure-Go alternative for Kafka applications, with superior ergonomics and no CGO dependencies.

---

*Last Updated: January 2025*
*Franz-Go Version: v1.20.2*
*Kafka Version: Confluent Platform 7.5.0 (KRaft)*
