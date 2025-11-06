# Kafka Rack Awareness: Comprehensive Guide

Rack awareness in Kafka is a critical feature for building fault-tolerant, highly available systems. Let me walk you through everything from basics to edge cases.

## What is Rack Awareness?

Rack awareness enables Kafka to distribute partition replicas across different failure domains (racks, availability zones, data centers). This ensures that if one failure domain goes down, your data remains available.

**Key concept**: A "rack" in Kafka is a logical grouping - it could represent:
- Physical server racks
- AWS Availability Zones
- Data centers
- Any failure domain you define

## Core Configuration

### Broker Configuration
```properties
# On each broker
broker.rack=us-east-1a  # or rack1, dc1, etc.
```

### How It Works
When creating topic partitions, Kafka's replica assignment algorithm tries to:
1. Spread replicas across different racks
2. Maintain even distribution across brokers
3. Follow the rack-aware assignment strategy

## All Possible Cases

### Case 1: Ideal Scenario - More Racks Than Replication Factor

**Setup**: 3 racks, replication factor = 3
```
Rack A: broker-1, broker-2
Rack B: broker-3, broker-4
Rack C: broker-5, broker-6
```

**Result**: Each replica goes to a different rack
```
Partition 0: Leader on Rack A, Replica on Rack B, Replica on Rack C ✓
```

### Case 2: Equal Racks and Replication Factor

**Setup**: 3 racks, replication factor = 3
```
Rack A: broker-1
Rack B: broker-2
Rack C: broker-3
```

**Result**: Perfect distribution, one replica per rack

### Case 3: Fewer Racks Than Replication Factor

**Setup**: 2 racks, replication factor = 3

```
Rack A: broker-1, broker-2
Rack B: broker-3, broker-4
```

**Result**: Kafka must place multiple replicas in the same rack
```
Partition 0: 
  - Replica 1: Rack A, broker-1 (Leader)
  - Replica 2: Rack B, broker-3
  - Replica 3: Rack A, broker-2  # Same rack as leader!
```

**Implication**: If Rack A fails, you lose 2 out of 3 replicas

### Case 4: Single Rack (No Rack Awareness)

**Setup**: All brokers in one rack or rack not configured

**Result**: Behaves like rack-unaware Kafka - only broker-level distribution

### Case 5: Unbalanced Rack Distribution

**Setup**: 3 racks with unequal brokers
```
Rack A: broker-1, broker-2, broker-3, broker-4
Rack B: broker-5
Rack C: broker-6
```

**Challenge**: Rack A has more brokers, potentially leading to:
- Uneven load distribution
- More partitions concentrated in Rack A
- Rack A brokers may be underutilized per-broker

## Edge Cases and Gotchas

### Edge Case 1: Mixed Configuration (Some Brokers Without Rack)

**Scenario**: Some brokers have `broker.rack` set, others don't

```
broker-1: rack=A
broker-2: rack=B
broker-3: rack=null  # No rack configured
```

**Behavior**: 
- Brokers without rack are treated as being in their own "rack" (the broker ID itself)
- This breaks rack-aware guarantees
- **Solution**: Always configure ALL brokers with rack IDs

### Edge Case 2: Rack Reassignment After Topic Creation

**Scenario**: You change a broker's rack assignment

```
# Original
broker-1: rack=A

# Changed to
broker-1: rack=B
```

**Behavior**:
- Existing partition assignments DON'T automatically rebalance
- New topics use the new rack configuration
- **Solution**: Run partition reassignment tool manually

```bash
kafka-reassign-partitions.sh --bootstrap-server localhost:9092 \
  --topics-to-move-json-file topics.json \
  --broker-list "1,2,3" \
  --generate
```

### Edge Case 3: Broker Addition to Existing Rack

**Scenario**: Adding new brokers to an existing rack

```
# Before
Rack A: broker-1, broker-2
Rack B: broker-3, broker-4

# After
Rack A: broker-1, broker-2, broker-5  # New broker added
Rack B: broker-3, broker-4
```

**Behavior**:
- New broker-5 doesn't automatically get partitions
- Existing partitions remain on broker-1 and broker-2
- **Solution**: Manual rebalancing or wait for new topics

### Edge Case 4: Insufficient Brokers in Racks

**Scenario**: RF=3, but some racks have fewer brokers than needed

```
Rack A: broker-1
Rack B: broker-2
Rack C: broker-3
# Creating topic with min.insync.replicas=3
```

**Problem**: If any rack/broker fails, you can't meet ISR requirements

**Solution**: Ensure `min.insync.replicas < replication.factor` for safety margin

### Edge Case 5: Leader Election and Rack Awareness

**Scenario**: Leader fails, need to elect new leader

**Behavior**:
- Rack awareness does NOT affect leader election
- Leader is chosen from ISR based on replica order, not rack
- Could end up with leader in "wrong" rack from a latency perspective

**Solution**: Use `PreferredReplicaLeaderElectionCommand` to rebalance leaders

### Edge Case 6: Cross-Datacenter Rack Awareness

**Scenario**: Multi-datacenter setup

```
DC1 Rack A: broker-1, broker-2
DC1 Rack B: broker-3, broker-4
DC2 Rack C: broker-5, broker-6
```

**Challenge**: 
- Network latency between DC1 and DC2
- Should DC be treated as "super-rack"?

**Best Practice**:
```properties
# Use hierarchical naming
broker.rack=dc1-rackA
broker.rack=dc1-rackB
broker.rack=dc2-rackC
```

### Edge Case 7: Rack Awareness with Constrained Resources

**Scenario**: Limited brokers, high replication factor

```
3 racks with 1 broker each
Replication Factor = 5
```

**Problem**: Impossible to satisfy both constraints
- Can't get 5 replicas with only 3 brokers

**Behavior**: Topic creation fails or violates rack-awareness

### Edge Case 8: Follower Fetching and Rack Awareness

**Scenario**: Consumer fetching from followers (KIP-392)

```properties
# Consumer configuration
replica.selector.class=org.apache.kafka.common.replica.RackAwareReplicaSelector
```

**Benefit**: Consumers can read from follower in the same rack, reducing cross-rack traffic

**Requirement**: Kafka 2.4+

### Edge Case 9: Partition Reassignment Tool Limitations

**Scenario**: Running reassignment without rack awareness

```bash
# Without --replica-alter-log-dirs option
kafka-reassign-partitions.sh --generate
```

**Problem**: Generated assignment may not respect rack awareness

**Solution**: Use `--replica-alter-log-dirs-throttle` and verify rack distribution manually

### Edge Case 10: Decommissioning a Rack

**Scenario**: Need to remove an entire rack

```
Removing Rack C with broker-5, broker-6
```

**Process**:
1. Reassign all partitions off those brokers
2. Wait for reassignment to complete
3. Verify no under-replicated partitions
4. Shut down brokers

**Gotcha**: If you shut down before reassignment completes, you lose data!

## Production Scenarios

### Scenario 1: AWS Multi-AZ Setup

```properties
# Broker configs across AZs
broker-1: broker.rack=us-east-1a
broker-2: broker.rack=us-east-1b
broker-3: broker.rack=us-east-1c
broker-4: broker.rack=us-east-1a
broker-5: broker.rack=us-east-1b
broker-6: broker.rack=us-east-1c
```

**Topic config**:
```properties
replication.factor=3
min.insync.replicas=2
```

**Fault tolerance**: Survives one AZ failure

### Scenario 2: Rack Failure During Peak Traffic

**Situation**: Rack B goes down during high load

**Impact**:
- Partitions with leader in Rack B become unavailable briefly
- Leader election triggers for those partitions
- Producers with `acks=all` may timeout
- Under-replicated partitions until Rack B returns

**Mitigation**:
```properties
# Producer configs
acks=all
retries=10
max.in.flight.requests.per.connection=1  # For ordering
```

### Scenario 3: Rolling Restart with Rack Awareness

**Best practice order**:
1. Restart one broker from each rack
2. Wait for ISR to stabilize
3. Move to next set

**Rationale**: Maintains rack diversity throughout restart

## Verification and Monitoring

### Check Rack Assignment
```bash
# View broker rack assignments
kafka-broker-api-versions.sh --bootstrap-server localhost:9092 | grep rack

# Check topic partition distribution
kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe --topic my-topic
```

### Monitor Rack-Related Metrics
- `UnderReplicatedPartitions`: Indicates rack/broker issues
- `OfflinePartitionsCount`: Complete rack failures
- `kafka.cluster:type=Partition,name=ReplicasCount`: Per-partition replica count

## Common Pitfalls

1. **Forgetting to configure ALL brokers**: Partial configuration breaks guarantees
2. **Not rebalancing after topology changes**: Stale assignments persist
3. **Ignoring network topology**: Treating all racks as equal when latency differs
4. **Over-replication in small clusters**: RF=5 with 3 racks wastes resources
5. **Not testing failure scenarios**: Assumptions about behavior during rack failures

## Best Practices Summary

1. **Always configure rack for ALL brokers** - no exceptions
2. **Use meaningful rack names** - reflect actual failure domains
3. **Replication factor ≤ number of racks** - for proper distribution
4. **min.insync.replicas < replication.factor** - safety margin
5. **Monitor under-replicated partitions** - early warning system
6. **Test rack failure scenarios** - chaos engineering
7. **Document your rack topology** - for operations team
8. **Plan capacity per rack** - not just per cluster
9. **Use rack-aware consumers** - optimize network usage (Kafka 2.4+)
10. **Automate reassignment** - don't rely on manual intervention

## Quick Decision Matrix

| Racks | Brokers/Rack | RF  | Recommendation              |
| ----- | ------------ | --- | --------------------------- |
| 3     | 2+           | 3   | ✓ Ideal                     |
| 2     | 2+           | 3   | ⚠️ Limited fault tolerance   |
| 3     | 1            | 5   | ✗ Impossible                |
| 1     | Many         | Any | ✗ No rack awareness benefit |
| 3     | Unequal      | 3   | ⚠️ May cause load imbalance  |

This covers the breadth of rack awareness scenarios you'll encounter. The key is understanding that rack awareness is about **failure domain isolation** - every decision should be evaluated through that lens.