Here's the build order, broken into phases — each one should fully work and be testable before you move to the next:

**Phase 0 — Setup**
- `go.mod`, basic folder skeleton, empty `cmd/node/main.go`
- Nothing to test yet, just scaffolding.

**Phase 1 — Chunking (no networking, no ring)**
- `internal/chunk/chunk.go` — `Chunk` struct: ID (hash of content), data, index
- `internal/chunk/chunker.go` — `Split(file) []Chunk` and `Reassemble([]Chunk) []byte`
- `internal/chunk/chunker_test.go`
- **Done when:** split a file, shuffle the chunks, reassemble, output byte-for-byte matches input.

**Phase 2 — Consistent hash ring (still no networking)**
- `internal/ring/ring.go` — ring struct, `AddNode`, `RemoveNode`, `GetNodesForKey(key, N) []Node` (with virtual nodes)
- `internal/ring/ring_test.go`
- **Done when:** you can add/remove nodes and measure that removing one node only remaps a small fraction of keys (write a test that asserts this quantitatively — good portfolio talking point too).

**Phase 3 — Single node: storage + transport**
- `internal/storage/diskstore.go` — `Put(chunkID, data)`, `Get(chunkID) []byte`, writes to local disk
- `internal/transport/server.go` — gRPC (or plain TCP to start, simpler) server exposing store/fetch
- `internal/transport/client.go` — client to call another node's transport server
- `cmd/node/main.go` — starts one node, wires storage + transport
- Add a `.proto` file if using gRPC: `internal/transport/chunk.proto`
- **Done when:** run one node, send a chunk to it over the network, fetch it back.

**Phase 4 — Metadata / manifest**
- `internal/metadata/manifest.go` — `Manifest{FileID, ChunkIDs, Locations}`
- `internal/metadata/store.go` — in-memory map, `RecordPlacement`, `Lookup(fileID)`
- **Done when:** manifest correctly tracks "file X → these chunk IDs → these nodes" and survives being queried repeatedly.

**Phase 5 — Client-facing API (still single node, no replication)**
- `internal/api/handlers.go` — `PUT /file`, `GET /file`
- `internal/api/router.go`
- **Done when:** `curl -T bigfile.mp4 localhost:8080/files/x` then `curl localhost:8080/files/x -o out.mp4` gives back an identical file, all on one node — chunk → store → manifest → reassemble on read.

**Phase 6 — Multi-node quorum (the real core)**
- `internal/coordinator/write.go` — fan out chunk to N nodes via ring, wait for W acks
- `internal/coordinator/read.go` — query R nodes, pick newest version
- Config: hardcode a static list of peer node addresses for now (real membership comes next phase)
- **Done when:** run 3–5 node processes locally, PUT through any one of them, confirm the chunk actually landed on the correct N nodes per the ring, GET returns correctly.

**Phase 7 — Cluster membership + failure detection**
- `internal/cluster/membership.go` — known nodes, join/leave
- `internal/cluster/heartbeat.go` — periodic ping loop, dead-node timeout
- **Done when:** kill one node process, confirm the others mark it dead within your timeout window (log it, don't need repair yet).

**Phase 8 — Repair / self-healing**
- `internal/replication/repair.go` — on node-death event, scan manifest for chunks that lived there, compute under-replicated set
- `internal/replication/worker.go` — small worker pool goroutine(s) that copy chunks from a surviving replica to a new/healthy node
- **Done when:** the actual demo — PUT a file, `kill -9` a node mid-cluster, wait, confirm replication count climbs back to N automatically and GET still works throughout.

**Phase 9 — Durability**
- Add a WAL to `internal/metadata/store.go` (or a new `wal.go`)
- **Done when:** restart the "coordinator" node process, manifest reloads from WAL instead of starting empty.

**Phase 10 — Packaging for demo**
- `scripts/docker-compose.yml` — spin up 5 nodes in containers
- `test/integration/failure_test.go` — scripted version of the Phase 8 demo (automated PUT → kill → assert → verify repair)
- `README.md` — architecture diagram + how to run the demo

**Stretch (only after all of the above works):** vector clocks instead of timestamps for conflict resolution, erasure coding instead of full replication, sharded/replicated metadata (Raft), Prometheus metrics per node.

Build strictly in this order — phases 1–2 have zero networking so you can nail the algorithms in isolation before debugging distributed behavior on top of them, which is where most people doing this project waste time.