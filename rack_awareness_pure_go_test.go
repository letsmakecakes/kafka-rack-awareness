package kafka_rack_awareness

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	broker1 = "localhost:9092"
	broker2 = "localhost:9093"
	broker3 = "localhost:9094"
)

var brokers = []string{broker1, broker2, broker3}

// Test 1: Verify brokers are accessible
func TestPureGo_BrokersAccessible(t *testing.T) {
	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err, "Should connect to broker 1")
	defer conn.Close()

	brokerList, err := conn.Brokers()
	require.NoError(t, err, "Should get broker list")

	assert.GreaterOrEqual(t, len(brokerList), 3, "Should have at least 3 brokers")
	t.Logf("Found %d brokers", len(brokerList))

	for _, broker := range brokerList {
		t.Logf("Broker ID: %d, Host: %s, Port: %d, Rack: %s",
			broker.ID, broker.Host, broker.Port, broker.Rack)
	}
}

// Test 2: Verify rack configuration
func TestPureGo_RackConfiguration(t *testing.T) {
	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	brokerList, err := conn.Brokers()
	require.NoError(t, err)

	rackMap := make(map[string]int)
	for _, broker := range brokerList {
		if broker.Rack != "" {
			rackMap[broker.Rack]++
		}
	}

	// We expect 3 racks: rack-a, rack-b, rack-c
	assert.Equal(t, 3, len(rackMap), "Should have 3 different racks")

	for rack, count := range rackMap {
		t.Logf("Rack '%s' has %d broker(s)", rack, count)
	}
}

// Test 3: Create topic and verify replica distribution
func TestPureGo_TopicReplicaDistribution(t *testing.T) {
	topicName := fmt.Sprintf("test-rack-dist-%d", time.Now().Unix())

	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	// Create topic with 6 partitions and replication factor 3
	topicConfig := kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     6,
		ReplicationFactor: 3,
	}

	err = controllerConn.CreateTopics(topicConfig)
	require.NoError(t, err, "Should create topic")
	t.Logf("Created topic: %s", topicName)

	// Wait for topic to be ready
	time.Sleep(3 * time.Second)

	// Get metadata
	partitions, err := conn.ReadPartitions(topicName)
	require.NoError(t, err)

	// Get broker info
	brokerList, err := conn.Brokers()
	require.NoError(t, err)

	// Create broker ID to rack mapping
	brokerRacks := make(map[int]string)
	for _, broker := range brokerList {
		brokerRacks[broker.ID] = broker.Rack
	}

	// Check replica distribution
	for _, partition := range partitions {
		racks := make(map[string]bool)

		for _, replica := range partition.Replicas {
			if rack, exists := brokerRacks[replica.ID]; exists {
				racks[rack] = true
			}
		}

		t.Logf("Partition %d: Leader=%d, Replicas=%v, Racks=%v",
			partition.ID, partition.Leader.ID, getBrokerIDs(partition.Replicas), getRacksFromBrokers(partition.Replicas, brokerRacks))

		// With RF=3 and 3 racks, replicas should be in different racks
		assert.Equal(t, 3, len(racks),
			"Partition %d should have replicas in 3 different racks", partition.ID)
	}

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 4: Producer with messages
func TestPureGo_ProducerMessages(t *testing.T) {
	topicName := fmt.Sprintf("test-producer-%d", time.Now().Unix())

	// Create topic first
	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     3,
		ReplicationFactor: 3,
	})
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	// Create producer
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topicName,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Write messages
	messages := []kafka.Message{}
	for i := 0; i < 10; i++ {
		messages = append(messages, kafka.Message{
			Key:   []byte(fmt.Sprintf("key-%d", i)),
			Value: []byte(fmt.Sprintf("value-%d", i)),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = writer.WriteMessages(ctx, messages...)
	require.NoError(t, err, "Should write all messages")
	t.Logf("Successfully produced %d messages", len(messages))

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 5: Consumer reading messages
func TestPureGo_ConsumerMessages(t *testing.T) {
	topicName := fmt.Sprintf("test-consumer-%d", time.Now().Unix())

	// Create topic
	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     3,
		ReplicationFactor: 3,
	})
	require.NoError(t, err)
	time.Sleep(5 * time.Second) // Wait longer for topic to be ready

	// Produce messages
	writer := &kafka.Writer{
		Addr:  kafka.TCP(brokers...),
		Topic: topicName,
	}

	ctx := context.Background()
	for i := 0; i < 20; i++ {
		err := writer.WriteMessages(ctx, kafka.Message{
			Value: []byte(fmt.Sprintf("message-%d", i)),
		})
		require.NoError(t, err)
	}
	writer.Close()

	// Consume messages
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topicName,
		GroupID: fmt.Sprintf("test-group-%d", time.Now().Unix()),
	})
	defer reader.Close()

	messageCount := 0
	timeout := time.After(15 * time.Second)

consumerLoop:
	for messageCount < 20 {
		select {
		case <-timeout:
			break consumerLoop
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			msg, err := reader.ReadMessage(ctx)
			cancel()

			if err != nil {
				continue
			}

			messageCount++
			t.Logf("Consumed message from partition %d: %s", msg.Partition, string(msg.Value))
		}
	}

	assert.Equal(t, 20, messageCount, "Should consume all 20 messages")

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 6: Leader distribution across racks
func TestPureGo_LeaderDistribution(t *testing.T) {
	topicName := fmt.Sprintf("test-leaders-%d", time.Now().Unix())

	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	// Create topic with 9 partitions
	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     9,
		ReplicationFactor: 3,
	})
	require.NoError(t, err)
	time.Sleep(3 * time.Second)

	// Get partitions
	partitions, err := conn.ReadPartitions(topicName)
	require.NoError(t, err)

	// Get broker racks
	brokerList, err := conn.Brokers()
	require.NoError(t, err)

	brokerRacks := make(map[int]string)
	for _, broker := range brokerList {
		brokerRacks[broker.ID] = broker.Rack
	}

	// Count leaders per rack
	leadersByRack := make(map[string]int)
	for _, partition := range partitions {
		rack := brokerRacks[partition.Leader.ID]
		leadersByRack[rack]++
		t.Logf("Partition %d leader is broker %d in rack %s",
			partition.ID, partition.Leader.ID, rack)
	}

	// Verify leaders are distributed
	assert.Equal(t, 3, len(leadersByRack), "Leaders should be in all 3 racks")

	for rack, count := range leadersByRack {
		t.Logf("Rack %s has %d leaders", rack, count)
		assert.GreaterOrEqual(t, count, 2, "Each rack should have at least 2 leaders")
	}

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 7: High partition count
func TestPureGo_HighPartitionCount(t *testing.T) {
	topicName := fmt.Sprintf("test-high-part-%d", time.Now().Unix())

	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     30,
		ReplicationFactor: 3,
	})
	require.NoError(t, err)
	time.Sleep(3 * time.Second)

	partitions, err := conn.ReadPartitions(topicName)
	require.NoError(t, err)

	brokerList, err := conn.Brokers()
	require.NoError(t, err)

	brokerRacks := make(map[int]string)
	for _, broker := range brokerList {
		brokerRacks[broker.ID] = broker.Rack
	}

	wellDistributed := 0
	for _, partition := range partitions {
		racks := make(map[string]bool)
		for _, replica := range partition.Replicas {
			if rack, exists := brokerRacks[replica.ID]; exists {
				racks[rack] = true
			}
		}

		if len(racks) == 3 {
			wellDistributed++
		}
	}

	assert.Equal(t, 30, wellDistributed,
		"All 30 partitions should have replicas across 3 racks")
	t.Logf("Partitions well-distributed across racks: %d/30", wellDistributed)

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Test 8: Single partition with rack awareness
func TestPureGo_SinglePartition(t *testing.T) {
	topicName := fmt.Sprintf("test-single-part-%d", time.Now().Unix())

	conn, err := kafka.Dial("tcp", broker1)
	require.NoError(t, err)
	defer conn.Close()

	controller, err := conn.Controller()
	require.NoError(t, err)

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	require.NoError(t, err)
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 3,
	})
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	partitions, err := conn.ReadPartitions(topicName)
	require.NoError(t, err)
	assert.Equal(t, 1, len(partitions), "Should have exactly 1 partition")

	brokerList, err := conn.Brokers()
	require.NoError(t, err)

	brokerRacks := make(map[int]string)
	for _, broker := range brokerList {
		brokerRacks[broker.ID] = broker.Rack
	}

	partition := partitions[0]
	racks := make(map[string]bool)
	for _, replica := range partition.Replicas {
		if rack, exists := brokerRacks[replica.ID]; exists {
			racks[rack] = true
		}
	}

	assert.Equal(t, 3, len(racks), "Single partition should have replicas in all 3 racks")
	t.Logf("Single partition replicas: %v, Racks: %v",
		getBrokerIDs(partition.Replicas), getRacksFromBrokers(partition.Replicas, brokerRacks))

	// Cleanup
	err = controllerConn.DeleteTopics(topicName)
	if err != nil {
		t.Logf("Warning: Failed to delete topic: %v", err)
	}
}

// Helper function to get broker IDs from broker list
func getBrokerIDs(brokers []kafka.Broker) []int {
	ids := []int{}
	for _, broker := range brokers {
		ids = append(ids, broker.ID)
	}
	return ids
}

// Helper function to get racks from broker list
func getRacksFromBrokers(brokers []kafka.Broker, brokerRacks map[int]string) []string {
	racks := []string{}
	for _, broker := range brokers {
		if rack, exists := brokerRacks[broker.ID]; exists {
			racks = append(racks, rack)
		}
	}
	sort.Strings(racks)
	return racks
}
