## Who Decides the Leader Broker?

**The producer does *not* decide** which broker is leader.

The **Kafka Controller** (a special broker in the cluster) is responsible for deciding:

* Which broker is the **leader** for each partition.
* Which brokers are replicas (followers).

The controller continuously monitors broker health and ensures each partition has exactly one leader broker at any time.

---

## How Leader Assignment Works (Cluster-Level)

When a topic is created, Kafka assigns:

1. **Partitions count** (e.g., 3)
2. **Replication factor** (e.g., 3)

Then Kafka distributes leaders across brokers to balance load.

### Example Cluster

```
Brokers:
  broker1
  broker2
  broker3
```

### Topic config

```
partitions = 3
replication factor = 3
```

### Possible Allocation

| Partition | Leader  | Replicas                  |
| --------- | ------- | ------------------------- |
| 0         | broker1 | broker1, broker2, broker3 |
| 1         | broker2 | broker2, broker3, broker1 |
| 2         | broker3 | broker3, broker1, broker2 |

This spreading ensures:

* Leaders are balanced across brokers.
* High availability (each partition has multiple replicas).

---

## What Happens During Broker Failure

If a leader broker goes down:

* The **controller** detects it (via ZooKeeper in older Kafka OR via Raft/KRaft in newer Kafka).
* A **replica is promoted** to become the new leader.
* Producers/consumers automatically update their metadata and continue.

No producer code changes needed.

---

## Producer’s Role

The **producer never decides a leader**.

**Producer → asks any broker → gets metadata → sees leader assignments → sends messages to the leader.**

Flow:

```
Producer → Bootstrap Broker → Get metadata → Send message directly to leader broker
```

---

## TL;DR

| Responsibility                                   | Who Handles It                                   |
| ------------------------------------------------ | ------------------------------------------------ |
| Decide which broker is leader for each partition | **Kafka Controller (cluster)**                   |
| Assign partitions to brokers                     | **Kafka Controller**                             |
| Route message to correct broker                  | **Producer (based on metadata)**                 |
| React to leader failure                          | **Kafka Controller + Producer metadata refresh** |

 ------------

## How does this work with a cluster with two racks?

When you have **multiple racks**, Kafka uses **rack awareness** to make sure data is spread across racks for fault-tolerance.
This affects **where leader and replica partitions are placed** — but still **not decided by the producer**.

---

## First: What is Rack Awareness?

Each broker is configured with the rack (or availability zone) it lives in:

```
broker.rack=RACK_A
```

Example:

| Broker | Rack |
| ------ | ---- |
| b1     | A    |
| b2     | A    |
| b3     | B    |
| b4     | B    |

Kafka uses this when placing **partition replicas**.

---

## Goal of Rack-Aware Placement

* Avoid all replicas of a partition being in the **same rack**
* Survive complete **rack failure**
* Balance leaders across racks

So if a **replication factor = 2** and you have **2 racks**, Kafka will ensure:

```
One replica on Rack A
One replica on Rack B
```

If **replication factor = 3**, and 2 racks exist, placement may look like:

```
Two replicas in Rack A
One replica in Rack B
```

(or vice-versa for balancing)

---

## Leader Election With Racks

* The **Kafka Controller** still decides the **partition leader**.
* Kafka tries to **spread leaders across racks evenly**, so no rack becomes overloaded.

Leader selection is based on:

1. Which replica is **in-sync (ISR)**
2. Rack diversity balancing

### Example Partition Placement

| Partition | Leader | Replicas (with racks) |
| --------- | ------ | --------------------- |
| P0        | b1 (A) | b1(A), b3(B)          |
| P1        | b3 (B) | b3(B), b2(A)          |
| P2        | b4 (B) | b4(B), b1(A)          |

Even distribution of leaders across racks.

---

## What Does the Producer Do in a Multi-Rack Setup?

**Exactly the same as before**:

1. Connects to **any bootstrap broker** (could be in any rack).
2. Fetches metadata (which includes leader + replica racks).
3. Sends message to the **leader broker**, regardless of rack.

Producers do *not* care about racks — the cluster routing logic handles it.

```
Producer → Leader Broker (which may be in Rack A or Rack B)
```

Data transfer across racks happens automatically.

---

## What About Consumer Reads?

Consumers also read from the **leader**, not a follower.
So reads may cross racks too — unless **Rack Aware Read** mode is enabled (optional KIP-392 feature).

---

## Rack Failure Scenario

If **Rack A goes down**:

* All leader partitions in Rack A lose leaders.
* Kafka Controller promotes **in-sync replicas in Rack B** to **become new leaders**.
* Producers fetch updated metadata and continue.

Cluster stays operational (as long as `replication.factor >= number_of_racks`).

---

## TL;DR

| Responsibility                   | Who Handles It                                                |
| -------------------------------- | ------------------------------------------------------------- |
| Spread replicas across racks     | Kafka Controller + Rack-aware assignment                      |
| Choose leader partition per rack | Kafka Controller                                              |
| Route messages to the leader     | Producer (after metadata lookup)                              |
| Continue on rack failure         | Controller promotes a new leader; producer refreshes metadata |

---------

## What if one broker goes down in a rack that has a leader partition?

If a broker goes down in a rack and that broker was the **leader** for one or more partitions, Kafka will **automatically promote a follower replica** (as long as replicas are in-sync) to become the new leader.

This happens **without your producer or consumer needing to change anything**.

---

## Step-by-Step What Happens

### 1. Broker Goes Down

The broker hosting the **leader** partition becomes unavailable.

### 2. Kafka Controller Detects the Failure

The **controller broker** constantly monitors broker heartbeats.
If a broker stops responding, the controller marks it **dead**.

### 3. Controller Promotes a New Leader

For every partition that had its leader on that broker, the controller:

* Looks at the **ISR (In-Sync Replica) list**
* Chooses the **next replica** as the new leader

Because of **rack-aware replication**, there is already a replica in the **other rack**, so the cluster stays healthy.

```
Old Leader (Rack A, Broker Down) ❌
New Leader (Rack B, In-Sync Replica) ✅
```

### 4. Producers & Consumers Update Metadata

Clients periodically refresh metadata or receive metadata refresh errors.

Then they **automatically send traffic to the new leader broker**.

No manual rerouting required.

---

## Example

### Cluster Layout (Before Failure)

| Partition | Leader | Followers | Rack Spread |
| --------- | ------ | --------- | ----------- |
| P0        | b1 (A) | b3 (B)    | A ↔ B       |

### b1 in Rack A goes down

* Controller sees b1 is gone
* `b3` is already in-sync

### Cluster Layout (After Leader Rebalancing)

| Partition | Leader | Followers                     | Rack Spread        |
| --------- | ------ | ----------------------------- | ------------------ |
| P0        | b3 (B) | (none in A until b1 recovers) | B only temporarily |

Service continues without downtime (as long as ISR was healthy).

---

## If the Replica Was **Not** In-Sync (ISR Dropped)?

If no follower is in-sync:

| Scenario                                    | Result                                                                                                           |
| ------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `unclean.leader.election = false` (default) | Partition becomes **unavailable** until leader broker returns → preserves data correctness.                      |
| `unclean.leader.election = true`            | Kafka may promote an **out-of-sync replica**, causing **possible data loss**, but keeps the partition available. |

### In production, you usually keep:

```
unclean.leader.election=false
```

*This prioritizes correctness over availability.*

---

## Key Takeaways

| Event                 | Behavior                                                        |
| --------------------- | --------------------------------------------------------------- |
| Leader broker fails   | Controller promotes an ISR follower replica                     |
| Producer behavior     | Automatically updates metadata & writes to new leader           |
| Consumer behavior     | Automatically re-balances to new leader                         |
| Rack-awareness effect | Ensures replicas exist on another rack, so failover still works |
