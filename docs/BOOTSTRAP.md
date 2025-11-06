# Kafka Broker Discovery and Message Routing

This document explains how Kafka clients interact with brokers in a clustered setup, and clarifies what happens when you only configure a single broker in your application.

## Overview

In a Kafka cluster, multiple brokers work together to store and serve data. When writing a Kafka client (for example, in Go), it is common to provide only **one broker address** in the producer configuration.

This broker is known as a **bootstrap broker**.

```
brokers := []string{"broker1:9092"} // Only one broker listed
````

## What the Bootstrap Broker Does

The bootstrap broker's purpose is only to help the client **discover the cluster**. When the producer connects, it requests **metadata**:

* List of all brokers in the cluster
* Topic and partition information
* Which broker is the **leader** for each partition

Once the metadata is received, the producer now knows the **entire cluster layout** — not just the bootstrap broker.

## Producing Messages

When sending (producing) messages, the producer does **not** simply send all data to the bootstrap broker.

Instead, it:

1. Determines which **partition** the message belongs to (round-robin or key-based hashing).
2. Looks up which broker is the **leader** for that partition.
3. Sends the message **directly to the leader broker**, regardless of which broker was originally configured.

### Example

| Partition | Leader Broker |
| --------- | ------------- |
| 0         | Broker 2      |
| 1         | Broker 1      |
| 2         | Broker 3      |

If a message is assigned to **Partition 2**, the producer will automatically send it to **Broker 3**, even if you originally connected to **Broker 1** as the bootstrap broker.

## Summary

* Providing only one broker in config is **normal and correct**.
* That broker is used only for **metadata discovery**.
* Kafka producers automatically route messages to the **correct broker** (partition leader).
* You do **not** need to list all brokers for producing to work.

### Final Key Point

> The broker you configure in your Go application is **not** where messages are necessarily written. After metadata retrieval, messages are produced directly to the broker that leads the target partition.

