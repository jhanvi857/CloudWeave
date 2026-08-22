# CloudWeave Architecture & Consistency Model

For the complete Architecture Decisions Record (ADR), see [decisions.md](../decisions.md).

---

## Consistency & Durability Model

CloudWeave enforces a clear separation between its data plane and control plane:

- **Control Plane Consistency**: Gossip-replicated metadata across active nodes (`/internal/manifest`, `/internal/join`, `/internal/leave`, `/internal/revoke-key`) backed by periodic background **Anti-Entropy reconciliation** (every 30s) and a local append-only **Write-Ahead Log (WAL)** on each node.
- **Data Plane Consistency**: Dynamo-style $N/W/R$ quorum with content-addressed chunk placement over a consistent hash ring (150 virtual nodes).
- **Durability Sequence Invariant**:
  $$\text{Chunk Write} \longrightarrow W\text{ Quorum Acks} \longrightarrow \text{Local WAL Commit (fsync)} \longrightarrow \text{Peer Gossip Broadcast} \longrightarrow \text{Client 201 Created}$$
  *Manifest commit only happens after $W$ chunk acks; data loss is only possible if all $N$ replicas of a chunk are lost before background self-healing repair runs.*

---

## Core System Subsystems

1. **Streaming Chunker (`internal/chunk`)**: Splits files into SHA-256 content-addressed chunks using 1MB fixed buffers or FastCDC rolling hash content-defined boundaries (Deduplication).
2. **Consistent Hash Ring (`internal/ring`)**: 150 virtual nodes per physical host smooth partition distribution, ensuring only $1/N$ keys remap during node topology changes.
3. **Quorum Coordinator (`internal/coordinator`)**: Any node acts as coordinator, fanning out chunk reads/writes in parallel across $N$ ring nodes and verifying $W$ write / $R$ read acknowledgments.
4. **Metadata & WAL (`internal/metadata`)**: In-memory metadata map with causal vector clocks, backed by synchronous `metadata.wal` disk flushes.
5. **Anti-Entropy Reconciler (`internal/cluster/anti_entropy.go`)**: Runs periodic background sync every 30s over `GET /internal/manifest` to reconcile partitions and catch up restarting nodes.
6. **Failure Detection & Flap-Damping (`internal/cluster`)**: Requires 4 consecutive failed heartbeats before declaring a node dead, avoiding false evictions during GC pauses.
7. **Deterministic Self-Healing Repair (`internal/replication`)**: Lexicographical primary survivor election coordinates re-replication to healthy nodes with zero duplicate transfer bandwidth.
8. **In-Flight Registry & Mark-and-Sweep GC (`internal/gc`, `internal/storage`)**: Active upload sessions reserve chunk IDs in memory, preventing GC from purging chunks during concurrent streaming uploads.
9. **Amazon S3 Compatibility Surface (`internal/s3`)**: Full AWS SigV4 auth, ListObjectsV2 pagination, multipart uploads, and `aws-chunked` per-chunk HMAC signature validation.
