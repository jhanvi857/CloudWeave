# CloudWeave Architecture Decisions Record (ADR)

This document records the architectural decisions, trade-offs, and rationale behind the design and implementation of **CloudWeave**.

---

## Consistency & Durability Model

CloudWeave enforces a clear separation of concerns between its data plane and control plane:

```
┌──────────────────────────────────────────────────────────────────────────┐
│                             DATA PLANE                                   │
│  - Dynamo-style tunable N/W/R quorum consensus                           │
│  - Content-addressable SHA-256 chunk placement over Consistent Hash Ring │
│  - Parallel worker-bounded chunk transfers + persistent keep-alive conns │
│  - LRU chunk caching on read + deterministic self-healing repair         │
└──────────────────────────────────────────────────────────────────────────┘
                                     │
                                     ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                            CONTROL PLANE                                 │
│  - Gossip broadcast (/internal/manifest, /internal/join, /internal/leave)│
│  - Periodic background Anti-Entropy reconciliation (every 30s)           │
│  - Local append-only Write-Ahead Log (metadata.wal) with synchronous sync│
│  - Vector Clocks (VC) for causal version history & conflict detection    │
└──────────────────────────────────────────────────────────────────────────┘
```

### Durability Ordering Invariant
A write operation follows a strict, non-negotiable durability sequence:
1. **Chunk In-Flight Registration**: Incoming stream chunks are registered in the active `InFlightRegistry` to prevent race conditions during concurrent Garbage Collection sweeps.
2. **Quorum Fan-Out**: The coordinator fans out chunk data to the top $N$ nodes on the consistent hash ring in parallel.
3. **Quorum Acknowledgment**: The write proceeds only after at least $W$ nodes write the chunk to local disk and return success.
4. **Local WAL Commit**: The node writes the file manifest to `metadata.wal` and calls synchronous disk flush (`f.Sync()`).
5. **Peer Gossip Broadcast**: The committed manifest is broadcast to active cluster peers over HTTP/HTTPS `/internal/manifest`.
6. **Anti-Entropy Reconciliation**: Periodic background sync pulls any manifests missed during network partitions or downtime.
7. **Client Acknowledgment**: The client receives HTTP `201 Created` or `200 OK`.

> **Durability Invariant**: Manifest commit only occurs after $W$ chunk acknowledgments are received. Data loss is only possible if all $N$ replicas of a chunk are lost before background self-healing repair runs.

---

## 1. Programming Language & Zero-Dependency Runtime: Go (Golang)

### Context
Distributed storage nodes require high I/O throughput, low-latency network primitives, predictable memory management, and simple cross-platform deployment.

### Decision
Built CloudWeave entirely in **Go (1.22+)** with zero external C/C++ dependencies or CGO bindings.

### Rationale
1. **Native Concurrency**: Go's lightweight goroutines and channels provide a natural model for parallel chunk transfers, background heartbeats, anti-entropy sync, and worker pools.
2. **Standard Library Capabilities**: Go's standard library provides production-grade HTTP/2, TLS, hashing (`crypto/sha256`), and atomic filesystem primitives.
3. **Single Binary Distribution**: Cross-compiles to a single static binary containing the storage engine, transport server, S3 layer, CLI, and embedded web dashboard.

---

## 2. Content-Addressable Storage (CAS) & Streaming Chunking

### Context
Storing monolithic multi-gigabyte files directly creates head-of-line blocking, difficult replication recovery, and excessive memory overhead.

### Decision
- Files are split into fixed 1 MB chunks (default) or content-defined chunks.
- Each chunk's unique identifier is the hexadecimal SHA-256 hash of its raw byte payload: `ChunkID = hex(SHA256(data))`.
- All uploads and downloads use constant-buffer streaming pipelines (`chunk.SplitStream`).

### Rationale
1. **Self-Verifying Integrity**: A chunk's name *is* its cryptographic hash. Any silent bit rot or disk corruption immediately fails verification (`sha256(data) == chunkID`).
2. **Natural Deduplication**: If two tenants or two versions upload identical chunks, they produce identical chunk IDs and map to the exact same storage block without duplicate disk consumption.
3. **Bounded Memory Safety**: Streaming chunks through a bounded worker pool guarantees that memory usage remains $O(\text{chunk size} \times \text{workers})$ rather than $O(\text{file size})$.

---

## 3. FastCDC Rolling Hash for Content-Defined Chunking & Deduplication

### Context
Fixed-size chunking (e.g., exact 1 MB boundaries) suffers from the *byte-shift problem*: inserting a single byte at the beginning of a file changes all subsequent chunk boundaries, causing 100% deduplication failure.

### Decision
- Implemented **FastCDC / Gear Hash** rolling content-defined chunking (`internal/chunk/cdc.go`).
- Target average chunk size of 64 KB with minimum (16 KB) and maximum (256 KB) bounds.

### Rationale
1. **Boundary Resilience**: Content-defined chunking limits boundary displacement from insertions/deletions, so unchanged regions often retain their original chunk boundaries and hashes.
2. **High Throughput**: The Gear hash algorithm uses a 256-entry precomputed lookup table with single-cycle bit shifts (`fp = (fp << 1) + gearTable[b]`), operating significantly faster than traditional Rabin fingerprints.

---

## 4. Consistent Hashing Ring with Virtual Nodes (150 VNodes)

### Context
In a distributed storage cluster, chunks must be distributed deterministically across nodes. Modulo hashing ($N \pmod{\text{node count}}$) remaps almost all keys when a node joins or leaves, triggering massive data migration.

### Decision
- Implemented a consistent hash ring with **150 virtual nodes** per physical node (`internal/ring/ring.go`).
- Ring positions are derived from `SHA-256(nodeID#vnodeIndex)` mapped to a 32-bit integer space.

### Rationale
1. **Minimal Key Remapping**: When a node is added or removed, only $1/N$ fraction of keys migrate on average.
2. **Uniform Load Balancing**: 150 virtual nodes per physical host smooth out the hash space distribution, preventing hot spots.
3. **Logarithmic Lookup**: Finding the replica set for any chunk ID executes in $O(\log(\text{vnodes}))$ time using binary search over sorted ring hashes.

---

## 5. Tunable Quorum Consensus Model ($N, W, R$)

### Context
Distributed storage requires a balance between strong consistency, write latency, and high availability in the presence of node network partitions or failures.

### Decision
- Implemented Dynamo-style configurable quorums:
  - **N** = Total replicas per chunk (default: 3).
  - **W** = Write quorum acknowledgments required (default: 2).
  - **R** = Read quorum acknowledgments required (default: 2).

### Rationale
1. **Strong Consistency Guarantee ($W + R > N$)**: When $W=2, R=2, N=3$, the write quorum and read quorum are mathematically guaranteed to overlap on at least one node containing the latest chunk version.
2. **Fault Tolerance**: The system tolerates $N - W = 1$ node failure during writes and $N - R = 1$ node failure during reads without downtime.
3. **Any-Node Coordination**: Any node can receive a client request and act as the coordinator, dispatching chunk operations across the ring.

---

## 6. Failure Detection: Heartbeats with Flap-Damping & Consecutive Miss Thresholds

### Context
In distributed environments, transient network hiccups, temporary garbage collection pauses, or CPU spikes can cause a single ping to drop. Instantly marking nodes dead on 1-2 missed pings triggers false-positive node evictions and wasteful repair storms.

### Decision
- Implemented periodic background heartbeat checks (`internal/cluster/heartbeat.go`).
- Default: interval = 2s, dead timeout = 8s, requiring **4 consecutive failed heartbeats** before marking a node dead.
- Flap-damping: Any single successful ping immediately resets the node's consecutive failure counter.

### Rationale
1. **GC & Latency Spikes Resilience**: A 1-2 second GC pause will not falsely evict healthy nodes.
2. **Decisive Eviction on Real Death**: Persistent unreachability across 4 consecutive checks reliably evicts dead nodes and initiates repair.

---

## 7. Self-Healing Replication with Deterministic Coordinator Election

### Context
When a storage node fails permanently, its chunks must be re-replicated to restore the target replication factor $N$. If every surviving node independently launches duplicate transfers of the same chunks, the cluster suffers severe network saturation.

### Decision
- Implemented background self-healing repair (`internal/replication/repair.go`, `worker.go`).
- **Deterministic Repair Coordinator**: For each under-replicated chunk, only the primary surviving replica (the first node in `aliveLocs`) coordinates the repair transfer. All secondary survivors skip that chunk.
- Repair transfers are processed through a bounded background worker pool.

### Rationale
1. **Zero Duplicate Bandwidth**: Eliminates redundant parallel transfers of identical chunks.
2. **Automatic Self-Healing**: Replication factor automatically climbs back to $N$ without operator intervention.
3. **Bounded Background Work**: The worker pool prevents repair operations from overwhelming client traffic.

---

## 8. In-Memory Metadata Store & Honest Scaling Limits

### Context
Fast metadata lookup is critical for streaming throughput.

### Decision
- In-memory thread-safe manifest map (`internal/metadata/store.go`) backed by append-only Write-Ahead Logging.
- Honest memory capacity: Supports **~150,000 object keys per 256 MB RAM**.
- Explicit path forward: Embedded LSM-tree storage (PebbleDB / BoltDB) for deployments exceeding 10M keys.

---

## 9. Write-Ahead Logging (WAL) for Crash Durability

### Context
In-memory metadata is lost on process termination unless durably persisted.

### Decision
- Implemented binary append-only Write-Ahead Log (`internal/metadata/wal.go`).
- Operations (`OpRecordManifest`, `OpDeleteManifest`, `OpRecordKey`, `OpDeleteKey`) are written to `metadata.wal` and flushed (`f.Sync()`) before client write confirmations.
- On startup, the node replays `metadata.wal` to rebuild in-memory state.

---

## 10. In-Flight Chunk Registry & Mark-and-Sweep Garbage Collection

### Context
Deleting an object removes its metadata manifest. The raw chunk files on disk become unreferenced orphans. If GC sweeps during an in-flight upload, it could delete newly written chunks before the manifest is committed.

### Decision
- Implemented `InFlightRegistry` (`internal/storage/inflight.go`) that tracks chunk IDs participating in active streaming and S3 multipart uploads.
- Mark Phase: Collects all chunk IDs across all manifests (all namespaces, all versions), in-flight upload sessions, and active multipart upload parts.
- Sweep Phase: Sweeps disk stores and purges unreferenced chunks.

### Rationale
1. **Zero Race Window**: In-flight uploads are never swept mid-stream.
2. **Reclaimed Storage**: Reclaims disk capacity after object deletion or aborted uploads.

---

## 11. Amazon S3 Compatibility & Native AWS SigV4 / `aws-chunked` Verification

### Context
Supporting the Amazon S3 standard enables instant compatibility with enterprise tools (AWS CLI, `boto3`, Terraform, `rclone`, Cyberduck, `restic`).

### Decision
- Implemented S3 REST protocol layer (`internal/s3`):
  - Bucket operations: `PUT /{bucket}`, `GET /` (ListBuckets), `DELETE /{bucket}`, `HEAD /{bucket}`.
  - Object operations: `PUT /{bucket}/{key}`, `GET /{bucket}/{key}`, `HEAD /{bucket}/{key}`, `DELETE /{bucket}/{key}`.
  - S3 Pagination: `GET /{bucket}?list-type=2` (ListObjectsV2 with prefix, delimiter, continuation tokens).
  - Multipart Uploads: `InitiateMultipartUpload`, `UploadPart`, `CompleteMultipartUpload`, `AbortMultipartUpload`.
  - **AWS Signature Version 4 (SigV4)** and **`aws-chunked` streaming HMAC verification**.

---

## 12. One-Way (Hash-Only) Credential Storage & Multi-Tenancy

### Context
Shared infrastructure requires isolation between organizations. Plaintext API keys must never be stored on disk or in logs.

### Decision
- Every key manifest is strictly scoped by namespace: `ScopedKey = namespace + "/" + fileID`.
- API keys are generated using cryptographically secure random bytes (`crypto/rand`).
- Plaintext keys are returned to the client **once** upon creation; only the **SHA-256 hash** (`KeyHash`) is stored in RAM and persisted in the WAL.
- Admin endpoints (`/admin/*`) enforce strict `IsAdmin = true` validation.

---

## 13. End-to-End Transport Security (mTLS & Mesh Cluster Secret)

### Decision
- **TLS & Mutual TLS (mTLS)**: Supports client and inter-node HTTPS encryption with configurable CA certificate validation (`-tls-cert`, `-tls-key`, `-tls-ca`).
- **Internal Cluster Secret**: Inter-node RPCs (`/chunks/*`, `/internal/*`) require a shared secret header (`X-Cluster-Secret`) verified using constant-time comparison (`crypto/subtle`).

---

## 14. Client-Side Argon2id & Convergent Encryption

### Decision
- Built client-side encryption into the Go Client SDK (`client/crypto.go`):
  - **Cipher**: AES-256-GCM.
  - **Key Derivation**: Argon2id (3 iterations, 32 MB RAM, 2 threads).
  - **Convergent Encryption Option**: Deterministic HMAC-SHA256 derivation of salt and nonce from plaintext and passphrase, allowing global deduplication and client-side encryption to operate together seamlessly.

---

## 15. Vector Clocks & Object Versioning Stack

### Decision
- Each manifest maintains a logical **Vector Clock** (`map[string]uint64`) tracking causal update histories across nodes.
- Overwriting an existing key archives previous revisions under unique version IDs (`GET /files/{key}?versions=true`, `GET /files/{key}?version_id={id}`).

---

## 16. Embedded Single-Binary Web Dashboard (`go:embed`)

### Decision
- Implemented a single-page dashboard (`internal/api/dashboard.html`) embedded directly into the Go binary using `//go:embed`.
- Accessible at `http://localhost:9000/dashboard` with live node health, real-time Prometheus statistics, and a "Kill Node" demo button.

---

## 17. Anti-Entropy Periodic Metadata Reconciliation

### Decision
- Implemented background anti-entropy sync (`internal/cluster/anti_entropy.go`).
- Nodes exchange manifest states every 30 seconds over `GET /internal/manifest`.
- Automatically pulls missing manifests on node restart or partition healing without requiring client requests.

### Rationale & Scale Honesty
1. **Self-Healing Partition Recovery**: A node can be killed, restarted with an empty WAL, and automatically catch up to full manifest state without any client traffic.
2. **Current Implementation vs Merkle Diffing**: Current anti-entropy performs a full manifest state exchange (`GET /internal/manifest`) every 30s across active peers. This is simple, robust, and completely self-healing for clusters up to tens of thousands of objects. Key-range digest exchange or Merkle tree diffing is a planned follow-up for deployments scaling beyond 100K objects.

---

## 18. Chunk Cache Invalidation vs. Metadata Versioning

### Clarification
- **Chunk-Data Cache (`DiskStore.cache`)**: Because chunks are content-addressed and immutable by SHA-256 hash (`ChunkID = hex(SHA256(chunkData))`), chunk-data cache invalidation on object overwrite or deletion is strictly **memory hygiene** (promptly freeing memory allocated to unreferenced chunk bytes) rather than a stale-read correctness fix.
- **Metadata Layer (`metaStore`)**: Read correctness is enforced at the metadata layer. `metaStore.RecordPlacement` immediately updates the in-memory manifest mapping `(namespace, fileID)` to the new version's chunk ID list, advances the vector clock, and flushes to `metadata.wal` before acknowledging writes. Subsequent reads immediately fetch the new chunk list.
