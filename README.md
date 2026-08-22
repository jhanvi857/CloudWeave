# CloudWeave

### A local, S3-compatible distributed object store for learning and prototyping distributed storage.

CloudWeave lets you run a **multi-node object-storage cluster on your own machine** and use it like S3.

Upload a file through AWS CLI or `boto3`, CloudWeave distributes its chunks across nodes, replicates them using configurable quorum rules, and repairs replicas when a node fails.

The main idea is simple: **instead of treating object storage as a black box, CloudWeave lets you see and experiment with what happens inside a distributed storage system.**

---

## Why CloudWeave?

Use it to:

- **Learn distributed object storage** - follow an object from S3 request → chunking → distribution → replication.
- **Prototype S3 applications locally** without depending on cloud storage.
- **Experiment with failures** - kill a node and watch detection and replica repair.
- **Explore distributed-storage algorithms** such as consistent hashing, quorum reads/writes, replication, anti-entropy, and erasure coding.
- **Study the implementation** - the system is intentionally small and written from scratch in Go.

### Key features

- S3-compatible API + AWS CLI / `boto3`
- FastCDC content-defined chunking + SHA-256 deduplication
- Consistent hashing (150 virtual nodes)
- Configurable `N/W/R` quorum
- Replication **or** Reed-Solomon erasure coding
- Automatic failure detection and replica repair
- WAL-backed metadata durability
- Vector-clock object versioning
- Embedded live cluster dashboard
- Native Go SDK + `cweave` CLI

---

## Quickstart & Failure Demo

### Option 1: Run with Go

```bash
go build -o node.exe ./cmd/node

./node.exe -port 9000 -data-dir ./data/node1 -admin-key "master-secret-key" -peers "http://localhost:9001,http://localhost:9002"

./node.exe -port 9001 -data-dir ./data/node2 -admin-key "master-secret-key" -peers "http://localhost:9000,http://localhost:9002"

./node.exe -port 9002 -data-dir ./data/node3 -admin-key "master-secret-key" -peers "http://localhost:9000,http://localhost:9001"
```

### Option 2: Run with Docker Compose

```bash
docker compose up -d
```

### Open the Dashboard

Open **http://localhost:9000/dashboard** in your browser.

Upload an object, then click **"Simulate Failure"** to watch the cluster detect node failure and self-heal in real time:

```text
Node failure
     ↓
Heartbeat detection
     ↓
Node removed from topology
     ↓
Under-replicated chunks detected
     ↓
Replica repair
     ↓
Cluster becomes healthy again
```

---

## Use It Like S3

### 1. Configure Credentials

CloudWeave supports built-in `.env` file loading on startup or standard shell variables:

**Option A: `.env` file (in project root):**
```env
AWS_ACCESS_KEY_ID=master-secret-key
AWS_SECRET_ACCESS_KEY=master-secret-key
```

**Option B: Shell environment variables:**
```bash
# Linux / macOS (bash):
export AWS_ACCESS_KEY_ID="master-secret-key"
export AWS_SECRET_ACCESS_KEY="master-secret-key"

# Windows (PowerShell):
$env:AWS_ACCESS_KEY_ID="master-secret-key"
$env:AWS_SECRET_ACCESS_KEY="master-secret-key"
```

### 2. Run AWS CLI / Python S3 Commands

```bash
aws --endpoint-url http://localhost:9000 s3 mb s3://demo
aws --endpoint-url http://localhost:9000 s3 cp video.mp4 s3://demo/
aws --endpoint-url http://localhost:9000 s3 ls s3://demo
```

Or with Python (`boto3`):

```python
import boto3

s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:9000",
    aws_access_key_id="master-secret-key",
    aws_secret_access_key="master-secret-key",
    region_name="us-east-1",
)

s3.upload_file("video.mp4", "demo", "video.mp4")
```

---

## Architecture

```text
                AWS CLI / boto3 / Go SDK
                         │
                         ▼
                  S3-Compatible API
                         │
                         ▼
                Streaming Chunker
                         │
                  FastCDC + SHA-256
                         │
                         ▼
                 Consistent Hash Ring
                         │
                    N/W/R Quorum
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
       Node 1          Node 2          Node 3
          │              │              │
          └──────────────┼──────────────┘
                         │
                  Failure Detection
                         │
                         ▼
                   Replica Repair
```

Each node stores chunks locally. Any node can coordinate a request, while the cluster handles placement, replication, failure detection, and repair.

---

## Storage Modes

| Mode | Example | Trade-off |
|---|---|---|
| **Replication** | `N=3` | Faster, 3× storage |
| **Erasure Coding** | `K=4, M=2` | ~1.5× storage, higher recovery cost |

Enable erasure coding:

```bash
./node.exe -storage-mode=erasure -k=4 -m=2
```

---

## Benchmarks

Run benchmarks locally:

```bash
go test -bench . -count=5 ./test/benchmark
```

Measured locally on Intel Core Ultra 5 125H (Windows 64-bit, 3-node cluster, N=3, W=2, R=2) over 5-iteration median runs:

| Metric | Result | Notes |
|---|---:|:---|
| **Upload throughput** | **~152 MB/s** | Parallel 8-worker streaming writes |
| **Download throughput** | **~121 MB/s** | Post-write quorum read reassembly |
| **Warm cache read** | **~114 MB/s** | In-memory 64MB LRU cache hit |
| **Concurrent requests** | **~333 req/s** | Pooled persistent keep-alive connections |
| **Deduplicated upload** | **~485 MB/s** | FastCDC chunk match bypasses disk I/O |

See [`benchmarks.md`](benchmarks.md) for full benchmark breakdowns, memory scaling tables, and methodology.

---

## Limitations

CloudWeave is a **learning/prototyping system**, not a production replacement for S3 or MinIO.

Current limitations:

- **Metadata Scale**: Metadata uses an in-memory map backed by append-only WAL (~150K keys per 256MB RAM).
- **Anti-Entropy**: Anti-entropy currently exchanges full manifest state every 30s (hash-tree diffing is a planned follow-up for >100K objects).
- **Scope**: Multi-datacenter topologies and production-scale clusters are outside current scope.

---

## Documentation

- [`decisions.md`](decisions.md) - architecture decisions record (ADR) and consistency model
- [`workflow.md`](workflow.md) - request and storage flows
- [`benchmarks.md`](benchmarks.md) - performance benchmarks and empirical measurements
- [`docs/API.md`](docs/API.md) - developer API reference

## License

See [`LICENSE`](LICENSE).