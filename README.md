# CloudWeave

CloudWeave is a high-performance, multi-tenant distributed object storage system written in Go. Drawing architectural design principles from Amazon DynamoDB, Amazon S3, and Apache Cassandra, CloudWeave implements an Amazon S3-compatible protocol surface (AWS SigV4 authentication, bucket management, ListObjectsV2, multipart uploads), content-addressable streaming chunking, consistent hashing with virtual nodes, configurable N/W/R quorum consensus, automated heartbeat failure detection, Write-Ahead Logging (WAL) durability, vector clocks, Reed-Solomon erasure coding, Prometheus metrics, mark-and-sweep garbage collection, HTTP byte-range requests, Raft metadata consensus, multi-tenant namespace isolation, SHA-256 hashed API key authentication, dynamic cluster membership (join and leave), end-to-end TLS mesh encryption, an importable Go Client SDK, a dedicated CLI binary (`cweave`), and zero-knowledge convergent client-side encryption.

---

## Prerequisites

- **Go**: Version 1.22 or higher
- **Dependencies**: Standard Go library and internal packages (zero C/C++ runtime dependencies)
- **Docker and Docker Compose**: Optional, for containerized cluster deployment

---

## System Architecture

```mermaid
flowchart TD
    Client["Native Client (Go SDK / HTTP / CLI)"] -->|HTTPS + Bearer Key| API["API Router & Auth Middleware"]
    S3Client["AWS CLI / boto3 / S3 Tools"] -->|S3 Protocol + SigV4 Auth| S3API["S3 Compatibility Layer (internal/s3)"]
    S3API -->|SigV4 Auth Check| Auth["SHA-256 Auth Engine"]
    S3API -->|Bucket & Object Ops| API
    API -->|Validate Key Hash| Auth
    API -->|SplitStream / Reassemble| Chunker["Streaming Chunker (SHA-256)"]
    API -->|Causal Versioning| VC["Vector Clock Engine"]
    API -->|Prometheus Metrics| Metrics["Metrics Exporter (/metrics)"]
    API -->|Propose Transaction| Raft["Raft Consensus Engine"]
    API -->|Trigger Sweep| GC["Mark-and-Sweep Garbage Collector"]
    API -->|Topology Discovery| Discovery["GET /cluster/nodes"]
    
    Auth -->|Record/Delete Key| WAL["Write-Ahead Log (metadata.wal)"]
    Raft -->|Local Log Write| WAL
    WAL -->|Replay Committed State| Meta["Metadata Store"]

    Chunker --> Coordinator["Any-Node Quorum Coordinator (N, W, R)"]
    Coordinator --> Ring["Consistent Hash Ring (Virtual Nodes)"]
    Coordinator --> Node1["Storage Node 1 (DiskStore / Erasure Shards)"]
    Coordinator --> Node2["Storage Node 2 (DiskStore / Erasure Shards)"]
    Coordinator --> Node3["Storage Node 3 (DiskStore / Erasure Shards)"]
    Coordinator --> Node4["Storage Node 4 (DiskStore / Erasure Shards)"]
    Coordinator --> Node5["Storage Node 5 (DiskStore / Erasure Shards)"]

    GC -->|Purge Orphan Chunks| Node1
    GC -->|Purge Orphan Chunks| Node2

    Cluster["Cluster Failure Detector"] -->|HTTP/HTTPS Heartbeat| Node1
    Cluster -->|HTTP/HTTPS Heartbeat| Node2
    Cluster -->|HTTP/HTTPS Heartbeat| Node3
    Cluster -->|HTTP/HTTPS Heartbeat| Node4
    Cluster -->|HTTP/HTTPS Heartbeat| Node5

    Cluster -->|Remove Dead Node| Ring
    Cluster -->|Trigger Dead Node Event| Repair["Self-Healing Repair Manager"]
    Repair -->|Copy Missing Replicas| Node1
    Repair -->|Copy Missing Replicas| Node2
    Repair -->|Copy Missing Replicas| Node3
    Repair -->|Copy Missing Replicas| Node4
    Repair -->|Copy Missing Replicas| Node5
    Repair -->|Update Manifest Locations| Meta
```

---

## Core System Components

1. **Content-Addressable Streaming Chunking (`internal/chunk`)**
   Files are processed as streams and split into fixed-size data blocks (1 MB default). Each block is assigned a unique content-based identifier derived via SHA-256 hashing. Uploads and downloads use stream pipelines with a constant 1 MB buffer, preventing memory overflow during multi-gigabyte file transfers.

2. **Consistent Hash Ring (`internal/ring`)**
   Implements consistent hashing using 150 virtual nodes per physical node. This ensures balanced key distribution across cluster members and minimizes key migration during node joins or failures.

3. **Configurable Quorum Consensus (`internal/coordinator`)**
   Enforces tunable consistency across operations using three parameters:
   - **N** (Replication Factor): Total number of replicas assigned per chunk.
   - **W** (Write Quorum): Minimum number of successful write acknowledgments required for a write operation to succeed.
   - **R** (Read Quorum): Minimum number of successful node reads required for a read operation to succeed.

4. **Any-Node Coordination Guarantee (`internal/api`, `internal/coordinator`)**
   Clients can execute requests (`PUT`, `GET`, `DELETE`, Range `GET`) against any node in the cluster. The entry node coordinates chunk distribution across consistent hash rings and peer replication transparently.

5. **Multi-Tenant Namespacing and Custom Metadata (`internal/auth`, `internal/metadata`)**
   Objects are isolated within tenant namespaces (`X-Namespace` header or `/files/<namespace>/<key>`), preventing key collisions across tenants. Custom metadata headers (`X-Meta-<Key>: <Value>`) and `Content-Type` are stored in manifests and returned on retrieval.

6. **SHA-256 Hashed Credential Security and Dynamic Key Management (`internal/auth`, `internal/api`)**
   Admin endpoints (`POST /admin/keys`, `DELETE /admin/keys`, `GET /admin/keys`) allow dynamic key generation and revocation. Server generates 24-byte random keys (`crypto/rand`) returned once to the client upon creation. Keys are hashed using SHA-256 before storage; plaintext keys are never stored on disk or in the WAL.

7. **End-to-End TLS Encryption (`cmd/node`, `internal/transport`)**
   Supports TLS encryption (`-tls-cert`, `-tls-key`, `-tls-ca`) across both client-facing HTTP APIs and inter-node mesh transport (`PutChunk`, `GetChunk`, `/internal/manifest`, join and leave heartbeats).

8. **Dynamic Cluster Membership and Topology Discovery (`internal/cluster`, `client`)**
   Nodes join (`POST /admin/join`) or leave (`POST /admin/leave`) the cluster live without process restarts. External SDK clients query `GET /cluster/nodes` or enable background discovery (`EnableAutoDiscovery: true`) to automatically learn about topology updates.

9. **Importable Go Client SDK (`client/`)**
   Full-featured Go SDK providing simple object operations (`Put`, `Get`, `RangeGet`, `Delete`), automatic round-robin endpoint balancing, failover retries, and background topology discovery.

10. **Storage Architecture: Dual-Mode Engine (`internal/storage`, `internal/erasure`)**
    Supports Full Replication Mode (`replication`, default N=3, W=2, R=2) and Reed-Solomon Erasure Coding Mode (`-storage-mode=erasure -k=4 -m=2`), splitting data blocks into K=4 data shards and M=2 parity shards over Galois Field GF(2^8).

11. **Write-Ahead Logging and Durability (`internal/metadata`)**
    The Write-Ahead Log (`metadata.wal`) records metadata transactions (`OpRecordManifest`, `OpUpdateLocations`, `OpDeleteManifest`, `OpRecordKey`, `OpDeleteKey`) synchronously to disk before confirming writes. On restart, WAL replays log entries with zero data loss.

12. **Object Deletion and Garbage Collection (`internal/gc`, `internal/api`)**
    `DELETE /files/{key}` removes metadata records. Automated Mark-and-Sweep Garbage Collection (`POST /admin/gc`) snapshots active manifest references and sweeps local disk stores to purge unreferenced orphan chunks.

13. **HTTP Byte-Range Requests (`internal/api`)**
    Supports standard HTTP byte-range requests (`Range: bytes=start-end`) returning HTTP `206 Partial Content` for media seeking and resumable downloads.

14. **Vector Clocks (`internal/vectorclock`)**
    Tracks logical vector clocks (`map[string]uint64`) to resolve concurrent multi-master update conflicts (`Before`, `After`, `Concurrent`, `Equal`).

15. **Prometheus Metrics Exporter (`internal/metrics`)**
    Exposes operational counters and gauges at `GET /metrics` (`cloudweave_file_uploads_total`, `cloudweave_file_downloads_total`, `cloudweave_repaired_chunks_total`, `cloudweave_active_nodes`).

16. **Raft Metadata Consensus Engine (`internal/consensus`)**
    Implements a Raft-backed replicated log state machine to achieve distributed consensus across metadata store nodes.

17. **Embedded Real-Time Web Dashboard (`internal/api`)**
    Single-page responsive dashboard UI embedded directly in the node binary (`GET /dashboard` on port 9000), visualizing live node health, active topologies, real-time Prometheus statistics, and an admin-gated simulate kill button (`POST /admin/kill`) demonstrating automatic cluster failover and self-healing.

18. **File Versioning (`internal/metadata`, `client`)**
    Supports vector clock manifest history retention. Overwritten keys archive prior versions under unique version IDs (`GET /files/{key}?versions=true`, `GET /files/{key}?version_id={id}`).

19. **Client-Side Argon2id and Convergent Encryption (`client`)**
    Provides zero-knowledge client-side encryption using AES-256-GCM and memory-hard Argon2id key derivation. Includes Convergent Encryption mode, deriving a deterministic HMAC salt/nonce from plaintext so identical encrypted chunks produce matching ciphertext hashes, allowing deduplication and client-side encryption to operate seamlessly together.

20. **Command Line Interface `cweave` (`cmd/cweave`)**
    Dedicated CLI binary supporting `cweave put <file>`, `cweave get <key>`, `cweave versions <key>`, `cweave rm <key>`, and `cweave ls`.

21. **Content-Defined Chunking and Deduplication (`internal/chunk`)**
    Implements FastCDC / Gear hash rolling content-defined chunking. Identical content blocks share identical SHA-256 chunk IDs across different files, eliminating duplicate block storage on disk.

22. **Amazon S3 Compatibility Layer and AWS SigV4 Auth (`internal/s3`)**
    Provides an S3 protocol surface mounted alongside the native CloudWeave API. Speaks full Amazon S3 protocol (`PUT /{bucket}/{key}`, `GET /{bucket}/{key}`, `HEAD /{bucket}/{key}`, `DELETE /{bucket}/{key}`, `PUT /{bucket}`, `GET /` ListBuckets, `DELETE /{bucket}`, `GET /{bucket}?list-type=2` ListObjectsV2 with prefix/delimiter/continuation tokens, and multipart uploads). Authenticates via AWS Signature Version 4 (SigV4) using issued CloudWeave API keys. Standard tools like AWS CLI, `boto3`, Cyberduck, `rclone`, and `restic` point directly to CloudWeave via `--endpoint-url http://localhost:9000`.

23. **Per-Chunk Signature Validation (`internal/s3`)**
    Implements rolling SigV4 HMAC chain verification for S3 streaming requests (`aws-chunked` transfer encoding). Each incoming payload chunk signature is validated against the derived signing key and prior chunk signature, rejecting forged payload chunks with HTTP `403 SignatureDoesNotMatch`.

---

## Performance Benchmarks

CloudWeave includes a repeatable benchmark suite (`test/benchmark/benchmark_test.go`). Performance measurements over 5-iteration empirical runs (`go test -bench . -count=5 ./test/benchmark`):

| Metric | Baseline | Empirical Range / Median | Category and Architectural Analysis |
| :--- | :--- | :--- | :--- |
| **Upload Throughput (2MB)** | ~21.9 MB/s | **92.25 to 214.38 MB/s** (~151.61 MB/s) | **Over 4.4x to 9.5x Throughput Gain**: Bypasses global mutex lock contention around disk I/O and optimizes atomic file move operations. |
| **Download (2MB Post-Write Read)** | ~10.14 MB/s | **109.62 to 231.18 MB/s** (~120.90 MB/s) | **OS Page Cache Sensitive**: Uncached disk tier (~109 MB/s) vs OS page cache hit (~231 MB/s). |
| **Download (10MB Post-Write Read)** | ~10.14 MB/s | **128.98 to 149.62 MB/s** (~142.78 MB/s) | **OS Page Cache Sensitive**: Varies dynamically with kernel filesystem page cache residency. |
| **Download (Warm Cache - LRU RAM)** | ~10.14 MB/s | **75.06 to 177.57 MB/s** (~94.68 MB/s) | **Over 7.4x to 17.5x Gain**: In-memory LRU cache hit latency (~11.8 to 27.9 ms/op). |
| **Concurrent Load Latency** | ~47 req/sec | **289.2 to 558.8 req/sec** (~333 req/sec) | **Over 6.1x to 11.8x Gain**: Persistent HTTP connection pooling with `http.Transport` keep-alives. |

*Architectural Takeaway*: Eliminating global mutex lock contention around disk I/O and bypassing slow temporary file move sequences delivered a 4.4x to 9.5x boost in upload throughput, unlocking parallel chunk writes. Persistent HTTP connection pooling (`http.Transport`) eliminates per-request TCP/TLS handshake overhead.

For complete empirical analysis and bottleneck breakdown, see [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md).

---

## Payload Streaming Memory Safety and Metric Distinction

CloudWeave's HTTP and S3 handlers process incoming uploads and downloads using zero-buffering constant-RAM streaming (`chunk.SplitStream`). Request and response payloads are processed in 1 MB chunk increments, keeping payload memory bounded regardless of file size.

### Continuous In-Flight Container Memory Verification (`docker run --memory=256m`)

Live black-box verification was executed against a running CloudWeave Docker container limited strictly to 256 MB RAM (`docker run --memory=256m`) on standard S3 port 9000. Memory was sampled continuously every 50ms by a background thread poller while 50 concurrent workers executed active streaming uploads (1,000 MB total active concurrent payload transfer):

- **Peak In-Flight Physical Container RSS**: **121.6 MiB / 256 MiB (47.51% of container limit)** measured during active 50-worker concurrent payload transfer.
- **In-Flight Memory Headroom**: Container retained **134.4 MB (52.5% free memory headroom)** under `--memory=256m` during peak load.
- **Post-Burst Baseline**: Once the active 50-stream upload completed, container RSS immediately returned to **23.09 MiB** (9.02% of limit).
- **Zero OOM Events**: Container state remained active with `OOMKilled=false`.

### `m.HeapAlloc` vs `RSS` Metric Accounting
Application heap allocation (`m.HeapAlloc` = 240.74 MB at 50 streams) and physical container RSS (121.6 MiB in Docker) represent two distinct system layers:
- **`m.HeapAlloc` (Application Heap Layer)**: Measures Go runtime heap allocations, which include short-lived chunk garbage buffers awaiting standard GC sweep cycles.
- **Physical Container RSS (Kernel OS Page Layer)**: Measures physical resident memory pages mapped by OS page tables inside the Linux cgroup (`docker stats` / `memory.current`).
- Both metrics are real: `m.HeapAlloc` reaches ~240 MB in application heap space during 50 concurrent streams before GC sweeps, while physical container RSS stays at 121.6 MiB (47.5% of the 256 MB cap), confirming that the physical memory footprint remains well within container boundaries during peak load.

### Operator Guidance: In-Memory Metadata Index Scaling

The metadata store (`internal/metadata/store.go`) maintains active file manifests in an in-memory map backed by WAL replay for sub-microsecond lookup latency. At ~1 KB per manifest record, a 256 MB container can hold up to ~150,000 active key manifests alongside node baseline memory. For multi-tenant production clusters managing hundreds of millions of keys, replace the in-memory `map` in `metadata.Store` with an embedded disk-backed key-value engine (such as PebbleDB or BoltDB).

---

## CLI Configuration Flags

The node executable (`cmd/node/main.go`) supports runtime flags and environment variables:

| Flag | Env Var | Default | Description |
| :--- | :--- | :--- | :--- |
| `-port` | `CLOUDWEAVE_PORT` | `9000` | Port for HTTP API, S3 API, and inter-node transport |
| `-data` | `CLOUDWEAVE_DATA` | `./data` | Directory path for local chunk storage |
| `-peers` | `CLOUDWEAVE_PEERS` | `""` | Comma-separated list of peer node HTTP/HTTPS addresses |
| `-wal` | `CLOUDWEAVE_WAL` | `<data>/metadata.wal` | Path to Write-Ahead Log file for metadata durability |
| `-api-keys` | `CLOUDWEAVE_API_KEYS` | `""` | Comma-separated initial static keys (`key=ns1;ns2` or `key=admin`) |
| `-storage-mode` | `CLOUDWEAVE_STORAGE_MODE` | `replication` | Storage strategy (`replication` or `erasure`) |
| `-k` | `CLOUDWEAVE_K` | `4` | Number of data shards K for erasure coding mode |
| `-m` | `CLOUDWEAVE_M` | `2` | Number of parity shards M for erasure coding mode |
| `-n` | `CLOUDWEAVE_N` | `3` | Replication factor N (number of replicas per chunk) |
| `-w` | `CLOUDWEAVE_W` | `2` | Write quorum W (minimum ACKs required for write success) |
| `-r` | `CLOUDWEAVE_R` | `2` | Read quorum R (minimum successful reads required for GET) |
| `-tls-cert` | `CLOUDWEAVE_TLS_CERT` | `""` | Path to TLS certificate file |
| `-tls-key` | `CLOUDWEAVE_TLS_KEY` | `""` | Path to TLS private key file |
| `-tls-ca` | `CLOUDWEAVE_TLS_CA` | `""` | Path to CA bundle file for node-to-node TLS verification |
| `-tls-insecure-skip-verify` | `CLOUDWEAVE_TLS_SKIP_VERIFY` | `false` | Skip TLS certificate verification for development |

---

## Getting Started

### 1. Build and Run Tests

Run the complete unit test suite across all packages:
```bash
go test -v ./...
```

Run the automated cluster integration tests:
```bash
go test -v ./test/integration/...
```

---

### 2. Docker Compose Cluster Deployment

Launch a 5-node cluster backed by persistent named Docker volumes in one command:

```bash
docker-compose up --build
```

On first boot, Node 1 generates a cryptographically random initial admin key and outputs it to the container log:
`[SECURITY] First boot detected: Generated initial admin API Key: cw_key_...`

Alternatively, pass your own custom key via environment variable:
`CLOUDWEAVE_API_KEYS="my-secure-key=admin" docker-compose up --build`

---

### 3. Local Multi-Node Setup (CLI)

Start a 3-node cluster manually (on first boot, Node 1 outputs your initial random admin key):

**Terminal 1 (Node 1):**
```powershell
go run cmd/node/main.go -port 9000 -data ./data-node1 -peers http://localhost:9001,http://localhost:9002
```

**Terminal 2 (Node 2):**
```powershell
go run cmd/node/main.go -port 9001 -data ./data-node2 -peers http://localhost:9000,http://localhost:9002
```

**Terminal 3 (Node 3):**
```powershell
go run cmd/node/main.go -port 9002 -data ./data-node3 -peers http://localhost:9000,http://localhost:9001
```

---

### 4. Basic API Verification

#### Issue a Tenant API Key (`POST /admin/keys`):
```bash
curl -X POST http://localhost:9000/admin/keys \
  -H "Authorization: Bearer <YOUR_INITIAL_ADMIN_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"namespaces": ["tenant1"], "is_admin": false}'
# Returns: {"key": "cw_key_...", "key_hash": "...", "namespaces": ["tenant1"], "is_admin": false}
```

#### Upload Object (`PUT /files/{key}`):
```bash
curl -X PUT http://localhost:9000/files/tenant1/documents/hello.txt \
  -H "Authorization: Bearer cw_key_..." \
  -H "Content-Type: text/plain" \
  -H "X-Meta-Author: Alice" \
  --data-binary "Hello CloudWeave Distributed World!"
```

#### Download Object (`GET /files/{key}`):
```bash
curl -X GET http://localhost:9000/files/tenant1/documents/hello.txt \
  -H "Authorization: Bearer cw_key_..."
```

#### HTTP Byte-Range Request (`Range: bytes=0-11`):
```bash
curl -X GET http://localhost:9000/files/tenant1/documents/hello.txt \
  -H "Authorization: Bearer cw_key_..." \
  -H "Range: bytes=0-11"
```

#### Discover Active Topology (`GET /cluster/nodes`):
```bash
curl -X GET http://localhost:9000/cluster/nodes \
  -H "Authorization: Bearer cw_key_..."
```

---

### 5. Amazon S3 Protocol and AWS CLI Verification

#### Configure AWS CLI Credentials:
```bash
export AWS_ACCESS_KEY_ID="cw_key_..."
export AWS_SECRET_ACCESS_KEY="cw_key_..."
export AWS_DEFAULT_REGION="us-east-1"
```

#### Bucket and Object Operations via AWS CLI:
```bash
# 1. Create S3 Bucket
aws s3 mb s3://my-bucket --endpoint-url http://localhost:9000

# 2. Upload Object
aws s3 cp document.txt s3://my-bucket/ --endpoint-url http://localhost:9000

# 3. List Objects (ListObjectsV2)
aws s3 ls s3://my-bucket/ --endpoint-url http://localhost:9000

# 4. Download Object
aws s3 cp s3://my-bucket/document.txt out.txt --endpoint-url http://localhost:9000

# 5. Multipart Upload (for large files >8MB)
aws s3 cp largefile.bin s3://my-bucket/ --endpoint-url http://localhost:9000

# 6. Delete Object
aws s3 rm s3://my-bucket/document.txt --endpoint-url http://localhost:9000
```

#### Python `boto3` SDK Integration:
```python
import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='cw_key_...',
    aws_secret_access_key='cw_key_...',
    region_name='us-east-1'
)

# Create bucket, PutObject, GetObject, ListObjectsV2
s3.create_bucket(Bucket='boto3-bucket')
s3.put_object(Bucket='boto3-bucket', Key='data.txt', Body=b'CloudWeave S3 API')
res = s3.list_objects_v2(Bucket='boto3-bucket')
content = s3.get_object(Bucket='boto3-bucket', Key='data.txt')['Body'].read()
```

---

## Developer API Documentation

For the complete API reference, error codes, authentication formats, and Go SDK guide, see [`docs/API.md`](docs/API.md).