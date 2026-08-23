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

### Option 1: Pull the Prebuilt Image (Recommended)

Pull the prebuilt multi-architecture image from GitHub Container Registry — works on Intel/AMD and Apple Silicon:

```bash
docker pull ghcr.io/jhanvi857/cloudweave:latest
```

Run a standalone CloudWeave instance (like MinIO, Redis, or PostgreSQL):

```bash
docker run -d \
  --name cloudweave \
  -p 9000:9000 \
  --memory=256m \
  -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" \
  -v cloudweave_data:/data \
  ghcr.io/jhanvi857/cloudweave:latest
```

> **Prebuilt images** are [published on GHCR](https://github.com/jhanvi857/CloudWeave/pkgs/container/cloudweave) for `linux/amd64` and `linux/arm64`. Every image is Trivy-scanned for vulnerabilities and smoke-tested (health check + OOM guard) before promotion.

### Option 2: Run with Docker Compose (5-Node Distributed Cluster)

Spin up a 5-node distributed cluster with automated quorum, failure detection, and self-healing:

```bash
docker compose up -d
```

### Option 3: Build from Source

```bash
# Build image locally
docker build -t cloudweave:latest .

# Run
docker run -d \
  --name cloudweave \
  -p 9000:9000 \
  --memory=256m \
  -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" \
  -v cloudweave_data:/data \
  cloudweave:latest
```

### Option 4: Run with Go (Multi-Node Cluster)

```bash
go build -o node.exe ./cmd/node

# Node 1
./node.exe -port 9000 -data ./data/node1 -api-keys "master-secret-key=admin" -peers "http://localhost:9001,http://localhost:9002"

# Node 2
./node.exe -port 9001 -data ./data/node2 -api-keys "master-secret-key=admin" -peers "http://localhost:9000,http://localhost:9002"

# Node 3
./node.exe -port 9002 -data ./data/node3 -api-keys "master-secret-key=admin" -peers "http://localhost:9000,http://localhost:9001"
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

### 1. Configure Client Credentials

> **Note on Environment Configuration:**
> - **CloudWeave Node Server (`node.exe` / Docker):** Automatically loads `.env` from the project root on startup for server configuration (`CLOUDWEAVE_*` settings).
> - **Client Tools (AWS CLI / Boto3 / SDKs):** Read AWS credentials from standard shell environment variables or `~/.aws/credentials`. Export the credentials in your shell before running AWS CLI commands:

**Shell environment variables:**
```bash
# Linux / macOS (bash / zsh):
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

## Docker & Container Deployment

### Adding CloudWeave to Your App's `docker-compose.yml`

You can drop CloudWeave directly into your existing development stack alongside Postgres, Redis, or your backend app:

```yaml
version: '3.8'

services:
  app:
    image: my-backend-app:latest
    environment:
      - S3_ENDPOINT=http://cloudweave:9000
      - AWS_ACCESS_KEY_ID=master-secret-key
      - AWS_SECRET_ACCESS_KEY=master-secret-key
    depends_on:
      - cloudweave

  cloudweave:
    image: ghcr.io/jhanvi857/cloudweave:latest
    container_name: cloudweave
    ports:
      - "9000:9000"
    mem_limit: 256m
    environment:
      - CLOUDWEAVE_API_KEYS=master-secret-key=admin
    volumes:
      - cloudweave_data:/data
    restart: unless-stopped

volumes:
  cloudweave_data:
```

### Environment Variables Reference

| Variable | Default | Description |
|---|---|---|
| `CLOUDWEAVE_PORT` | `9000` | HTTP/HTTPS port to bind |
| `CLOUDWEAVE_DATA` | `/data` | Directory for chunk & WAL storage |
| `CLOUDWEAVE_API_KEYS` | *(auto-generated)* | Initial API key mappings (e.g. `master-secret-key=admin` or `k1=ns1,k2=ns2`) |
| `CLOUDWEAVE_PEERS` | `""` | Comma-separated list of peer HTTP node addresses for multi-node cluster |
| `CLOUDWEAVE_CLUSTER_SECRET` | `""` | Shared secret header required for inter-node transport mesh |
| `CLOUDWEAVE_STORAGE_MODE` | `replication` | Storage engine mode (`replication` or `erasure`) |
| `CLOUDWEAVE_N` | `3` (cluster) / `1` (standalone) | Total replication factor |
| `CLOUDWEAVE_W` | `2` (cluster) / `1` (standalone) | Quorum write ACKs required |
| `CLOUDWEAVE_R` | `2` (cluster) / `1` (standalone) | Quorum read responses required |
| `CLOUDWEAVE_K` | `4` | Data shards count for erasure coding mode |
| `CLOUDWEAVE_M` | `2` | Parity shards count for erasure coding mode |
| `CLOUDWEAVE_TLS_CERT` | `""` | Path to TLS server certificate |
| `CLOUDWEAVE_TLS_KEY` | `""` | Path to TLS private key |
| `CLOUDWEAVE_TLS_CA` | `""` | Path to CA bundle for mutual TLS (mTLS) |
| `CLOUDWEAVE_TLS_CLIENT_AUTH`| `verify-if-given`| mTLS mode (`require`, `verify-if-given`, `none`) |

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