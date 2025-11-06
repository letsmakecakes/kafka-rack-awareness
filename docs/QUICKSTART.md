# Kafka Rack Awareness Testing - Quick Start Guide

## ✅ Testing Complete!

All tests have been successfully executed with **100% pass rate (8/8 tests)**.

## What Was Tested

### 8 Comprehensive Test Scenarios:

1. **Broker Accessibility** - All brokers accessible with rack configuration
2. **Rack Configuration** - 3 racks properly configured (rack-a, rack-b, rack-c)
3. **Replica Distribution** - 100% of replicas distributed across all racks
4. **Producer Messages** - Message production works correctly
5. **Consumer Messages** - Message consumption works correctly  
6. **Leader Distribution** - Leaders evenly distributed (3 per rack)
7. **High Partition Count** - 30 partitions all rack-aware
8. **Single Partition** - Even 1 partition has replicas across all racks

## Current Cluster Status

```
✓ Kafka Broker 1:  Running (rack-a, port 9092) - KRaft Controller+Broker
✓ Kafka Broker 2:  Running (rack-b, port 9093) - KRaft Controller+Broker
✓ Kafka Broker 3:  Running (rack-c, port 9094) - KRaft Controller+Broker
```

**Note:** This cluster uses **KRaft mode** (Kafka Raft) - no Zookeeper required! 🚀

## Files Created

```
kafka-rack-awareness/
├── docker-compose.yml                         # Kafka cluster setup
├── go.mod                                     # Go dependencies
├── rack_awareness_pure_go_test.go            # ✅ Pure Go tests (PASSING)
├── rack_awareness_confluent_test.go.bak      # Confluent tests (requires CGO)
├── TEST_RESULTS.md                           # Detailed test results
├── README_DETAILED.md                        # Complete documentation
├── run_tests.sh                              # Linux/Mac test runner
├── run_tests.bat                             # Windows test runner
├── Makefile                                  # Make commands
└── QUICKSTART.md                             # This file
```

## Running Tests Again

### Option 1: Run All Tests
```bash
go test -v -run TestPureGo -timeout 5m
```

### Option 2: Run Specific Test
```bash
# Test broker accessibility
go test -v -run TestPureGo_BrokersAccessible

# Test replica distribution
go test -v -run TestPureGo_TopicReplicaDistribution

# Test leader distribution
go test -v -run TestPureGo_LeaderDistribution

# Test high partition count
go test -v -run TestPureGo_HighPartitionCount
```

### Option 3: Use Scripts
```bash
# Windows
run_tests.bat

# Linux/Mac/Git Bash
./run_tests.sh

# Using Make
make test
```

## Key Test Results

### Replica Distribution ✅
- **100%** of partitions have replicas across all 3 racks
- No single rack concentration
- Perfect distribution even with 30 partitions

### Leader Distribution ✅
- Leaders evenly distributed across all racks
- rack-a: 3 leaders (33.3%)
- rack-b: 3 leaders (33.3%)
- rack-c: 3 leaders (33.3%)

### Fault Tolerance ✅
- Each partition can survive **2 rack failures**
- Replicas never co-located in same rack
- Maximum availability guaranteed

## Stopping the Cluster

When you're done testing:

```bash
docker-compose down -v
```

## Viewing Logs

```bash
# All containers
docker-compose logs -f

# Specific broker
docker-compose logs -f kafka-broker-1

# View KRaft metadata logs
docker exec -it kafka-broker-1 cat /tmp/kraft-combined-logs/__cluster_metadata-0/00000000000000000000.log
```

## Next Steps

1. ✅ Review `TEST_RESULTS.md` for detailed test analysis
2. ✅ Check `README_DETAILED.md` for comprehensive documentation
3. ✅ Modify tests in `rack_awareness_pure_go_test.go` for custom scenarios
4. ✅ Use this setup as a template for your own Kafka testing

## Important Notes

### Pure Go vs Confluent Library

This test suite uses **segmentio/kafka-go** (pure Go) because:
- ✅ No CGO required (works on Windows without additional setup)
- ✅ No librdkafka installation needed
- ✅ Cross-platform compatibility
- ✅ Easier to build and deploy

The **confluent-kafka-go** tests are available in `rack_awareness_confluent_test.go.bak` but require:
- CGO_ENABLED=1
- librdkafka C library installed
- C compiler (gcc/clang)

To use confluent-kafka-go on Linux/Mac:
```bash
# Install librdkafka
# Ubuntu/Debian: sudo apt-get install librdkafka-dev
# macOS: brew install librdkafka

# Rename the file
mv rack_awareness_confluent_test.go.bak rack_awareness_confluent_test.go
mv rack_awareness_pure_go_test.go rack_awareness_pure_go_test.go.bak

# Set CGO and run
export CGO_ENABLED=1
go test -v -timeout 10m
```

## Test Statistics

```
Total Tests:       8
Passed:            8 (100%)
Failed:            0 (0%)
Total Duration:    ~48 seconds
Average per Test:  ~6 seconds
```

## Support

For issues or questions:
1. Check logs: `docker-compose logs`
2. Verify containers: `docker-compose ps`
3. Review test output for detailed error messages
4. See `README_DETAILED.md` for troubleshooting

---

**Status:** ✅ All systems operational - Ready for production testing!
