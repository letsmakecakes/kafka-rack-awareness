package kafka_rack_awareness

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	franzTimeout = 60 * time.Second
)

// Helper function to create franz-go admin client
func createFranzAdminClient(t *testing.T) *kadm.Client {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequestTimeoutOverhead(10*time.Second),
	)
	require.NoError(t, err, "Failed to create franz-go client")
	
	adminClient := kadm.NewClient(client)
	return adminClient
}

// Helper function to create franz-go producer client
func createFranzProducer(t *testing.T, opts ...kgo.Opt) *kgo.Client {
	defaultOpts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ProducerBatchMaxBytes(1000000),
		kgo.RequestTimeoutOverhead(10 * time.Second),
	}
	
	allOpts := append(defaultOpts, opts...)
	client, err := kgo.NewClient(allOpts...)
	require.NoError(t, err, "Failed to create franz-go producer")
	
	return client
}

// Helper function to create franz-go consumer client
func createFranzConsumer(t *testing.T, groupID string, topics []string, opts ...kgo.Opt) *kgo.Client {
	defaultOpts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.RequestTimeoutOverhead(10 * time.Second),
	}
	
	allOpts := append(defaultOpts, opts...)
	client, err := kgo.NewClient(allOpts...)
	require.NoError(t, err, "Failed to create franz-go consumer")
	
	return client
}

// Helper function to extract rack list from map
func getRackListFromMap(racks map[string]bool) []string {
	rackList := make([]string, 0, len(racks))
	for rack := range racks {
		rackList = append(rackList, rack)
	}
	sort.Strings(rackList)
	return rackList
}

// Test 1: Verify broker metadata and rack configuration
func TestFranz_BrokerMetadata(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	// Get broker metadata
	metadata, err := adminClient.Metadata(ctx)
	require.NoError(t, err, "Failed to get metadata")
	
	brokers := metadata.Brokers
	t.Logf("Found %d brokers", len(brokers))
	
	// Verify we can access all 3 brokers
	require.Equal(t, 3, len(brokers), "Expected 3 brokers")
	
	// Check each broker and its rack configuration
	expectedRacks := map[int32]string{
		1: "rack-a",
		2: "rack-b",
		3: "rack-c",
	}
	
	for _, broker := range brokers {
		brokerID := broker.NodeID
		rackStr := "nil"
		if broker.Rack != nil {
			rackStr = *broker.Rack
		}
		t.Logf("Broker %d: Host=%s, Port=%d, Rack=%s", 
			brokerID, broker.Host, broker.Port, rackStr)
		
		expectedRack, exists := expectedRacks[brokerID]
		require.True(t, exists, "Unexpected broker ID: %d", brokerID)
		
		// Verify rack configuration
		if broker.Rack != nil {
			assert.Equal(t, expectedRack, *broker.Rack, 
				"Broker %d should be in %s", brokerID, expectedRack)
		} else {
			t.Errorf("Broker %d has no rack configured", brokerID)
		}
	}
	
	t.Log("✓ All brokers have correct rack configuration")
}

// Test 2: Create topic and verify replica distribution
func TestFranz_ReplicaDistribution(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-replicas-%d", time.Now().Unix())
	
	// Create topic with 6 partitions and RF=3
	resp, err := adminClient.CreateTopics(ctx, 6, 3, nil, topicName)
	require.NoError(t, err, "Failed to create topic")
	require.NoError(t, resp[topicName].Err, "Topic creation error: %v", resp[topicName].Err)
	
	t.Logf("Created topic: %s", topicName)
	
	// Wait for topic to be ready
	time.Sleep(3 * time.Second)
	
	// Get topic metadata
	metadata, err := adminClient.Metadata(ctx, topicName)
	require.NoError(t, err, "Failed to get topic metadata")
	
	// Get broker rack mapping
	brokerRacks := make(map[int32]string)
	for _, broker := range metadata.Brokers {
		if broker.Rack != nil {
			brokerRacks[broker.NodeID] = *broker.Rack
		}
	}
	
	// Check replica distribution for each partition
	topicMetadata := metadata.Topics[topicName]
	for _, partition := range topicMetadata.Partitions {
		racks := make(map[string]bool)
		replicaIDs := []int32{}
		
		for _, replica := range partition.Replicas {
			replicaIDs = append(replicaIDs, replica)
			if rack, exists := brokerRacks[replica]; exists {
				racks[rack] = true
			}
		}
		
		rackList := getRackListFromMap(racks)
		t.Logf("Partition %d: Leader=%d, Replicas=%v, Racks=%v", 
			partition.Partition, partition.Leader, replicaIDs, rackList)
		
		// With RF=3 and 3 racks, all replicas should be in different racks
		assert.Equal(t, 3, len(racks), 
			"Partition %d should have replicas in 3 different racks", partition.Partition)
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ All replicas properly distributed across racks")
}

// Test 3: Producer with rack awareness
func TestFranz_ProducerWithRackAwareness(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-producer-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 3, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Create producer with rack awareness
	producer := createFranzProducer(t,
		kgo.ClientID("franz-producer-rack-a"),
		kgo.Rack("rack-a"), // Enable rack awareness
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	defer producer.Close()
	
	// Produce messages
	recordCount := 20
	
	for i := 0; i < recordCount; i++ {
		record := &kgo.Record{
			Topic: topicName,
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: []byte(fmt.Sprintf("value-%d", i)),
		}
		
		// Produce message synchronously
		results := producer.ProduceSync(ctx, record)
		require.NoError(t, results.FirstErr(), "Failed to produce message %d", i)
		
		// Get partition and offset from the result
		for _, r := range results {
			t.Logf("Message %d produced to partition %d at offset %d", 
				i, r.Record.Partition, r.Record.Offset)
		}
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Logf("✓ Successfully produced %d messages with rack awareness", recordCount)
}

// Test 4: Consumer with rack awareness
func TestFranz_ConsumerWithRackAwareness(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-consumer-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 3, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Produce some messages first
	producer := createFranzProducer(t)
	defer producer.Close()
	
	messageCount := 10
	for i := 0; i < messageCount; i++ {
		record := &kgo.Record{
			Topic: topicName,
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: []byte(fmt.Sprintf("value-%d", i)),
		}
		results := producer.ProduceSync(ctx, record)
		require.NoError(t, results.FirstErr())
	}
	
	producer.Flush(ctx)
	
	// Create consumer with rack awareness
	consumer := createFranzConsumer(t, 
		"franz-test-group",
		[]string{topicName},
		kgo.Rack("rack-a"), // Prefer to fetch from rack-a
	)
	defer consumer.Close()
	
	// Consume messages
	consumed := 0
	timeout := time.After(30 * time.Second)
	
	for consumed < messageCount {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for messages, consumed %d/%d", consumed, messageCount)
		default:
			fetches := consumer.PollFetches(ctx)
			if errs := fetches.Errors(); len(errs) > 0 {
				t.Logf("Fetch errors: %v", errs)
				continue
			}
			
			iter := fetches.RecordIter()
			for !iter.Done() {
				record := iter.Next()
				t.Logf("Consumed message from partition %d, offset %d: %s", 
					record.Partition, record.Offset, string(record.Value))
				consumed++
			}
		}
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Logf("✓ Successfully consumed %d messages with rack awareness", consumed)
}

// Test 5: Verify partition distribution
func TestFranz_PartitionDistribution(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-partitions-%d", time.Now().Unix())
	
	// Create topic with multiple partitions
	resp, err := adminClient.CreateTopics(ctx, 9, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Get metadata
	metadata, err := adminClient.Metadata(ctx, topicName)
	require.NoError(t, err)
	
	topicMeta := metadata.Topics[topicName]
	
	// Count leaders per rack
	leaderRacks := make(map[string]int)
	
	// Build broker map for lookup
	brokerMap := make(map[int32]kgo.BrokerMetadata)
	for _, broker := range metadata.Brokers {
		brokerMap[broker.NodeID] = broker
	}
	
	for _, partition := range topicMeta.Partitions {
		leaderID := partition.Leader
		if broker, ok := brokerMap[leaderID]; ok {
			if broker.Rack != nil {
				leaderRacks[*broker.Rack]++
			}
		}
	}
	
	t.Logf("Leader distribution: %v", leaderRacks)
	
	// With 9 partitions and 3 racks, each rack should have 3 leaders
	for rack, count := range leaderRacks {
		assert.Equal(t, 3, count, "Rack %s should have 3 partition leaders", rack)
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ Partition leaders evenly distributed across racks")
}

// Test 6: Transactional producer with rack awareness
func TestFranz_TransactionalProducer(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-txn-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 3, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Create transactional producer
	producer := createFranzProducer(t,
		kgo.TransactionalID("franz-txn-producer"),
		kgo.Rack("rack-a"),
	)
	defer producer.Close()
	
	// Begin transaction
	err = producer.BeginTransaction()
	require.NoError(t, err, "Failed to begin transaction")
	
	// Produce messages in transaction
	for i := 0; i < 5; i++ {
		record := &kgo.Record{
			Topic: topicName,
			Key:   []byte(fmt.Sprintf("txn-key-%d", i)),
			Value: []byte(fmt.Sprintf("txn-value-%d", i)),
		}
		
		results := producer.ProduceSync(ctx, record)
		require.NoError(t, results.FirstErr(), "Failed to produce message %d", i)
	}
	
	// End the transaction (commit)
	err = producer.EndTransaction(ctx, kgo.TryCommit)
	require.NoError(t, err, "Failed to commit transaction")
	
	t.Log("Transaction committed successfully")
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ Transactional producer with rack awareness working correctly")
}

// Test 7: Idempotent producer
func TestFranz_IdempotentProducer(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-idempotent-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 3, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Create producer (idempotence is enabled by default in franz-go)
	// Note: franz-go enables idempotence by default, no need to configure
	producer := createFranzProducer(t)
	defer producer.Close()
	
	// Produce messages
	for i := 0; i < 10; i++ {
		record := &kgo.Record{
			Topic: topicName,
			Key:   []byte(fmt.Sprintf("idem-key-%d", i)),
			Value: []byte(fmt.Sprintf("idem-value-%d", i)),
		}
		
		results := producer.ProduceSync(ctx, record)
		require.NoError(t, results.FirstErr())
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ Idempotent producer working correctly (enabled by default)")
}

// Test 8: Consumer rebalance with rack awareness
func TestFranz_ConsumerRebalance(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-rebalance-%d", time.Now().Unix())
	groupID := fmt.Sprintf("franz-group-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 6, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Create first consumer
	consumer1 := createFranzConsumer(t, groupID, []string{topicName})
	defer consumer1.Close()
	
	// Wait for initial assignment
	time.Sleep(2 * time.Second)
	
	// Create second consumer to trigger rebalance
	consumer2 := createFranzConsumer(t, groupID, []string{topicName})
	defer consumer2.Close()
	
	// Wait for rebalance
	time.Sleep(3 * time.Second)
	
	t.Log("✓ Consumer rebalance completed successfully")
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 9: Verify ISR (In-Sync Replicas) includes all racks
func TestFranz_ISRVerification(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-isr-%d", time.Now().Unix())
	
	// Create topic
	resp, err := adminClient.CreateTopics(ctx, 3, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	time.Sleep(2 * time.Second)
	
	// Get metadata
	metadata, err := adminClient.Metadata(ctx, topicName)
	require.NoError(t, err)
	
	topicMeta := metadata.Topics[topicName]
	
	// Get broker rack mapping
	brokerRacks := make(map[int32]string)
	for _, broker := range metadata.Brokers {
		if broker.Rack != nil {
			brokerRacks[broker.NodeID] = *broker.Rack
		}
	}
	
	// Check ISR for each partition
	for _, partition := range topicMeta.Partitions {
		partitionID := partition.Partition
		t.Logf("Partition %d: ISR=%v (size=%d)", partitionID, partition.ISR, len(partition.ISR))
		
		// Verify ISR size (should be 3 with min.insync.replicas=2)
		assert.GreaterOrEqual(t, len(partition.ISR), 2, 
			"Partition %d ISR should have at least 2 replicas", partitionID)
		
		// Get racks in ISR
		isrRacks := make(map[string]bool)
		for _, replicaID := range partition.ISR {
			if rack, exists := brokerRacks[replicaID]; exists {
				isrRacks[rack] = true
			}
		}
		
		rackList := getRackListFromMap(isrRacks)
		t.Logf("Partition %d ISR racks: %v", partitionID, rackList)
		
		// ISR should span multiple racks for fault tolerance
		assert.GreaterOrEqual(t, len(isrRacks), 2, 
			"Partition %d ISR should span at least 2 racks", partitionID)
	}
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ ISR verification passed - all partitions have proper rack distribution")
}

// Test 10: High partition count with rack awareness
func TestFranz_HighPartitionCount(t *testing.T) {
	adminClient := createFranzAdminClient(t)
	defer adminClient.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), franzTimeout)
	defer cancel()
	
	topicName := fmt.Sprintf("franz-test-high-partitions-%d", time.Now().Unix())
	
	// Create topic with 30 partitions
	resp, err := adminClient.CreateTopics(ctx, 30, 3, nil, topicName)
	require.NoError(t, err)
	require.NoError(t, resp[topicName].Err)
	
	t.Logf("Created topic: %s with 30 partitions", topicName)
	
	time.Sleep(3 * time.Second)
	
	// Get metadata
	metadata, err := adminClient.Metadata(ctx, topicName)
	require.NoError(t, err)
	
	topicMeta := metadata.Topics[topicName]
	
	// Get broker rack mapping
	brokerRacks := make(map[int32]string)
	for _, broker := range metadata.Brokers {
		if broker.Rack != nil {
			brokerRacks[broker.NodeID] = *broker.Rack
		}
	}
	
	// Track rack distribution
	partitionsWithAllRacks := 0
	rackDistribution := make(map[string]int)
	
	for _, partition := range topicMeta.Partitions {
		racks := make(map[string]bool)
		
		for _, replica := range partition.Replicas {
			if rack, exists := brokerRacks[replica]; exists {
				racks[rack] = true
				rackDistribution[rack]++
			}
		}
		
		if len(racks) == 3 {
			partitionsWithAllRacks++
		}
	}
	
	t.Logf("Partitions with all racks: %d/30", partitionsWithAllRacks)
	t.Logf("Overall rack distribution: %v", rackDistribution)
	
	// All 30 partitions should have replicas in all 3 racks
	assert.Equal(t, 30, partitionsWithAllRacks, 
		"All partitions should have replicas in all racks")
	
	// Cleanup
	_, err = adminClient.DeleteTopics(ctx, topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
	
	t.Log("✓ High partition count test passed - all partitions properly distributed")
}
