# CloudWeave Request & Storage Workflows

For the full detailed directory map and subsystem lifecycles, see [workflow.md](../workflow.md).

---

## 1. Object Upload Flow (`PUT /files/{ns}/{key}` or S3 `PUT /{bucket}/{key}`)

```text
Client Request
      │ (Streaming body payload)
      ▼
API Handler / S3 Handler
      │
      ├─► 1. Register chunk IDs in InFlightRegistry (prevents GC sweep race)
      ├─► 2. SplitStream (FastCDC / 1MB fixed buffers)
      │       │
      │       ▼
      │   Quorum Coordinator (WriteChunk)
      │       │
      │       ▼
      │   Parallel fan-out to top N nodes on Consistent Hash Ring
      │       │
      │       ▼
      │   Wait for W successful disk writes (f.Sync)
      │
      ├─► 3. Local WAL Commit: Append OpRecordManifest to metadata.wal (f.Sync)
      ├─► 4. Peer Broadcast: POST /internal/manifest to active cluster peers
      ├─► 5. Unregister in-flight chunk IDs
      │
      ▼
Client Response: 201 Created
```

---

## 2. Object Download Flow (`GET /files/{ns}/{key}` or S3 `GET /{bucket}/{key}`)

```text
Client Request
      │
      ▼
API Handler / S3 Handler
      │
      ├─► 1. Lookup manifest in local MetaStore (or query quorum nodes if missing)
      ├─► 2. Stream chunk IDs in order:
      │       │
      │       ▼
      │   Quorum Coordinator (ReadChunk)
      │       │
      │       ├─► Fast-path: Check local LRU chunk cache
      │       ├─► Parallel read from R replica nodes
      │       └─► SHA-256 verify: hex(sha256(data)) == chunkID
      │
      ▼
Client Response: 200 OK (Binary stream + Content-Type + X-Meta-* headers)
```

---

## 3. Background Failure Detection & Self-Healing Repair Flow

```text
Heartbeat Ticker (2s interval)
      │
      ▼
Ping /health on each peer
      │
      ├─► Success: MarkAlive (reset consecutive failure counter to 0)
      │
      └─► Failure: RecordFailure
              │
              ▼
      4 consecutive misses (>8s timeout)?
              │
              ▼ (Yes)
      MarkDead & Remove from Consistent Hash Ring
              │
              ▼
      Submit Dead Node Job to Repair Worker Pool
              │
              ▼
      Scan active manifests for chunks held on dead node
              │
              ▼
      Deterministic Coordinator Check:
      Lexicographically sort alive survivors (aliveLocs).
      Only primary survivor (aliveLocs[0] == localAddr) coordinates repair.
              │
              ▼
      Fetch chunk from local store & transfer to new target node on Ring
              │
              ▼
      Update chunk locations in MetaStore & WAL
              │
              ▼
      Cluster health restored (Replication factor N climbs back to target)
```
