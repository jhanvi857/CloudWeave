'use client'

import React, { useState, useRef, useMemo, useEffect } from 'react'
import { Canvas, useFrame } from '@react-three/fiber'
import { OrbitControls, Html, Float } from '@react-three/drei'
import * as THREE from 'three'
import { CodeBlock } from '../ui/CodeBlock'
import {
  ChevronRight,
  ChevronLeft,
  Code,
  CheckCircle2,
  Lock,
  Layers,
  HardDrive,
  Server,
  ShieldCheck,
  FileCheck,
} from 'lucide-react'

interface PipelineStage {
  id: number
  name: string
  subtitle: string
  description: string
  details: string[]
  codeSnippet: string
  codeFile: string
}

const stages: PipelineStage[] = [
  {
    id: 1,
    name: '1. S3 REST Ingestion',
    subtitle: 'SigV4 Key Verification & Byte Stream Buffer',
    description: 'Incoming S3 PUT requests arrive at any gateway node. AWS Signature Version 4 HMAC-SHA256 headers are verified against cluster credentials before streaming into pooled sync.Pool memory buffers.',
    details: [
      'Zero-allocation sync.Pool byte buffers minimize GC pressure',
      'Bounded chunk ingestion streams at >150 MB/s per node',
      'Decentralized gateway: any node acts as coordinator',
    ],
    codeFile: 'internal/api/s3_handler.go',
    codeSnippet: `// Verify AWS SigV4 Auth & Stream Payload
func (h *Handler) PutObject(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    if err := auth.VerifySigV4(r, h.secretKey); err != nil {
        http.Error(w, "SignatureDoesNotMatch", http.StatusForbidden)
        return
    }
    
    // Acquire zero-allocation buffer from pool
    buf := h.bufferPool.Get().([]byte)
    defer h.bufferPool.Put(buf)
    
    // Pass stream directly to FastCDC chunker
    manifest, err := h.coordinator.IngestStream(ctx, r.Body, r.ContentLength)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("ETag", manifest.ETag)
    w.WriteHeader(http.StatusOK)
}`,
  },
  {
    id: 2,
    name: '2. FastCDC Chunking',
    subtitle: 'Gear Rolling Hash Boundary Cut Points',
    description: 'CloudWeave applies the FastCDC algorithm using a 64-entry Gear rolling hash matrix. Byte-level boundary shifts in edited files only invalidate the edited chunk, leaving all surrounding chunks bit-for-bit identical.',
    details: [
      'Normalized chunk sizing: Min 16 KB, Avg 64 KB, Max 256 KB',
      'Gear rolling hash computes 32-bit finger marks in <2 ns/byte',
      'Eliminates 80%+ redundant storage on incremental modifications',
    ],
    codeFile: 'internal/chunk/fastcdc.go',
    codeSnippet: `// FastCDC dynamic boundary chunker with Gear matrix
func (c *Chunker) Split(data []byte) []Chunk {
    var chunks []Chunk
    n := len(data)
    cursor := 0

    for cursor < n {
        remaining := n - cursor
        if remaining <= MinChunkSize {
            chunks = append(chunks, c.createChunk(data[cursor:]))
            break
        }
        
        // Fast-path rolling hash loop
        fingerprint := uint32(0)
        splitPoint := cursor + MinChunkSize
        for splitPoint < cursor+MaxChunkSize && splitPoint < n {
            fingerprint = (fingerprint << 1) + GearMatrix[data[splitPoint]]
            if (fingerprint & MaskNormalized) == 0 {
                break // Cut boundary found
            }
            splitPoint++
        }
        chunks = append(chunks, c.createChunk(data[cursor:splitPoint]))
        cursor = splitPoint
    }
    return chunks
}`,
  },
  {
    id: 3,
    name: '3. SHA-256 CAS & Dedup',
    subtitle: 'Cryptographic Fingerprint & Deduplication Filter',
    description: 'Each FastCDC chunk is hashed with SHA-256 to create its immutable Content-Addressed Storage (CAS) identifier. If a chunk hash already exists in the cluster manifest index, physical disk I/O is skipped.',
    details: [
      'Deterministic SHA-256 hex digest acts as global chunk ID',
      'In-memory chunk index checks existence in O(1) time',
      'Deduplicated uploads achieve >480 MB/s effective throughput',
    ],
    codeFile: 'internal/storage/cas.go',
    codeSnippet: `// Compute CAS Key & verify in-memory deduplication index
func (s *CASStore) IndexChunk(chunk []byte) (ChunkID, bool) {
    hasher := sha256.New()
    hasher.Write(chunk)
    id := ChunkID(hex.EncodeToString(hasher.Sum(nil)))

    s.mu.Lock()
    defer s.mu.Unlock()

    if entry, exists := s.index[id]; exists {
        entry.RefCount++
        return id, true // Deduplicated: 0 disk I/O
    }

    s.index[id] = &IndexEntry{RefCount: 1, Size: len(chunk)}
    return id, false // New unique chunk to write
}`,
  },
  {
    id: 4,
    name: '4. Consistent Hash Ring',
    subtitle: '150 Virtual Tumbler Nodes per Physical Machine',
    description: 'Chunk IDs are hashed into a 32-bit circular key space. Each storage node owns 150 virtual nodes (vnodes) distributed across the ring to prevent hot spots. Binary search on the ring returns the primary node and successor replicas.',
    details: [
      '150 vnodes per physical host ensure <4% standard deviation load',
      'Adding/removing a node only remaps 1/N of total keys',
      'Murmur3 32-bit hashing for ultra-low latency routing lookup',
    ],
    codeFile: 'internal/ring/hash_ring.go',
    codeSnippet: `// Lookup target primary node and N replicas on hash ring
func (r *HashRing) GetNodes(chunkID string, count int) []Node {
    r.mu.RLock()
    defer r.mu.RUnlock()

    val := r.hashFn([]byte(chunkID))
    idx := sort.Search(len(r.vnodes), func(i int) bool {
        return r.vnodes[i].Token >= val
    })
    if idx == len(r.vnodes) {
        idx = 0
    }

    selected := make([]Node, 0, count)
    seen := make(map[string]bool)

    for len(selected) < count && len(seen) < len(r.physicalNodes) {
        node := r.vnodes[idx].Node
        if !seen[node.ID] {
            seen[node.ID] = true
            selected = append(selected, node)
        }
        idx = (idx + 1) % len(r.vnodes)
    }
    return selected
}`,
  },
  {
    id: 5,
    name: '5. Quorum Replication (N/W/R)',
    subtitle: 'Decentralized Dynamo Quorum with mTLS Transport',
    description: 'The coordinator fans out the chunk to N=3 nodes concurrently over mutual TLS (mTLS) with connection pooling. The coordinator waits for W=2 fsync acknowledgments before marking the write successful.',
    details: [
      'Tunable consistency parameters: Default N=3, W=2, R=2',
      'Persistent HTTP/2 + mTLS keep-alive connections avoid TLS handshake overhead',
      'Asynchronous bounded worker pools prevent goroutine leaks',
    ],
    codeFile: 'internal/coordinator/quorum_write.go',
    codeSnippet: `// Parallel write fanout with W=2 acknowledgment quorum
func (c *Coordinator) WriteChunkQuorum(ctx context.Context, chunk Chunk, nodes []Node) error {
    acks := make(chan error, len(nodes))
    
    for _, node := range nodes {
        go func(target Node) {
            client := c.pool.Get(target.Address)
            err := client.PutChunk(ctx, chunk.ID, chunk.Data)
            acks <- err
        }(node)
    }

    successCount := 0
    var lastErr error
    for i := 0; i < len(nodes); i++ {
        err := <-acks
        if err == nil {
            successCount++
            if successCount >= c.W { // Quorum met
                return nil
            }
        } else {
            lastErr = err
        }
    }
    return fmt.Errorf("quorum failed: %d/%d acks (last: %v)", successCount, c.W, lastErr)
}`,
  },
  {
    id: 6,
    name: '6. WAL & Vector Clock Seal',
    subtitle: 'Synchronous Journal Manifest Durability',
    description: 'Once all chunks satisfy quorum, the object manifest and vector clock are appended to the coordinator Write-Ahead Log (WAL) with an fsync flush. On restart, manifest state is deterministically restored from the WAL.',
    details: [
      'Vector clocks track causal order and resolve concurrent updates',
      'CRC32 checksummed binary WAL entries guarantee crash consistency',
      'Returns 200 OK + ETag to S3 client',
    ],
    codeFile: 'internal/metadata/wal.go',
    codeSnippet: `// Append manifest commit to Write-Ahead Log
func (w *WAL) AppendManifest(m *Manifest) error {
    w.mu.Lock()
    defer w.mu.Unlock()

    payload, err := json.Marshal(m)
    if err != nil {
        return err
    }

    entry := WALEntry{
        Type:      EntryTypePut,
        Timestamp: time.Now().UnixNano(),
        CRC:       crc32.ChecksumIEEE(payload),
        Payload:   payload,
    }

    if err := binary.Write(w.file, binary.LittleEndian, &entry.Header); err != nil {
        return err
    }
    if _, err := w.file.Write(payload); err != nil {
        return err
    }
    return w.file.Sync()
}`,
  },
]

/* ========================================================================= */
/* STAGE 1: S3 INGESTION & MECHANICAL VAULT UNLOCK                           */
/* ========================================================================= */
function Stage1S3Ingestion() {
  const doorWheelRef = useRef<THREE.Group>(null)
  const streamRef = useRef<THREE.Group>(null)
  const boltRefs = useRef<THREE.Mesh[]>([])

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.8) % 3.0 // 3-second cycle

    // Wheel rotation on unlock
    if (doorWheelRef.current) {
      if (t < 1.0) {
        doorWheelRef.current.rotation.z = THREE.MathUtils.lerp(0, Math.PI / 2, t)
      } else {
        doorWheelRef.current.rotation.z = Math.PI / 2
      }
    }

    // Locking bolts slide outward
    boltRefs.current.forEach((bolt, i) => {
      if (bolt) {
        const angle = (i * Math.PI) / 2
        const dist = t < 1.0 ? THREE.MathUtils.lerp(0.8, 1.3, t) : 1.3
        bolt.position.x = Math.cos(angle) * dist
        bolt.position.y = Math.sin(angle) * dist
      }
    })

    // Byte stream particles flow through
    if (streamRef.current) {
      const p = (t - 0.8) / 2.2
      if (p > 0) {
        streamRef.current.position.z = THREE.MathUtils.lerp(2.5, -2.5, p)
        streamRef.current.visible = true
      } else {
        streamRef.current.visible = false
      }
    }
  })

  return (
    <group>
      {/* Outer Heavy Vault Portal Ring */}
      <mesh position={[0, 0, 0]}>
        <torusGeometry args={[1.5, 0.15, 16, 48]} />
        <meshStandardMaterial color="#26262B" metalness={0.9} roughness={0.2} />
      </mesh>

      {/* Rotating Wheel Gear Hub */}
      <group ref={doorWheelRef} position={[0, 0, 0]}>
        <mesh>
          <cylinderGeometry args={[0.7, 0.7, 0.15, 32]} />
          <meshStandardMaterial color="#19191C" metalness={0.9} roughness={0.2} />
        </mesh>
        <mesh position={[0, 0, 0.08]}>
          <torusGeometry args={[0.55, 0.05, 16, 32]} />
          <meshStandardMaterial color="#7A1F2B" metalness={0.95} roughness={0.1} />
        </mesh>
        {[0, Math.PI / 4, Math.PI / 2, (3 * Math.PI) / 4].map((rot, i) => (
          <mesh key={i} rotation={[0, 0, rot]}>
            <boxGeometry args={[1.2, 0.06, 0.1]} />
            <meshStandardMaterial color="#A6A9AE" metalness={0.9} roughness={0.15} />
          </mesh>
        ))}
      </group>

      {/* 4 Radial Vault Locking Bolts */}
      {[0, 1, 2, 3].map((i) => (
        <mesh
          key={i}
          ref={(el) => {
            if (el) boltRefs.current[i] = el
          }}
          rotation={[0, 0, (i * Math.PI) / 2]}
        >
          <cylinderGeometry args={[0.08, 0.08, 0.5, 16]} />
          <meshStandardMaterial color="#7A1F2B" metalness={0.9} roughness={0.15} />
        </mesh>
      ))}

      {/* Ingested Byte Stream (Stream Particles) */}
      <group ref={streamRef} position={[0, 0, 2.5]}>
        {[-0.3, 0, 0.3].map((offset, i) => (
          <mesh key={i} position={[offset, (i % 2 === 0 ? 0.1 : -0.1), 0]}>
            <sphereGeometry args={[0.09, 16, 16]} />
            <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3} />
          </mesh>
        ))}
      </group>

      <Html position={[0, 1.8, 0]} center distanceFactor={8}>
        <div className="px-2.5 py-1 rounded bg-[#000000]/95 border border-[#7A1F2B] font-mono text-[9px] text-[#EDE8DF] shadow-md whitespace-nowrap">
          <span className="font-bold text-[#EDE8DF]">SigV4 Auth OK</span> · Ingestion Stream
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE 2: FASTCDC CONTENT-DEFINED FRACTURING                              */
/* ========================================================================= */
function Stage2FastCDCChunking() {
  const shardRefs = useRef<THREE.Group[]>([])
  const laserRef = useRef<THREE.Mesh>(null)

  const shardSpecs = [
    { x: -1.35, width: 0.45, label: '16 KB', color: '#19191C' },
    { x: -0.65, width: 0.85, label: '64 KB (Gear Cut)', color: '#7A1F2B' },
    { x: 0.15, width: 0.55, label: '32 KB', color: '#19191C' },
    { x: 0.95, width: 0.75, label: '48 KB', color: '#19191C' },
  ]

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.8) % 3.0

    // Laser cut sweep
    if (laserRef.current) {
      if (t < 1.0) {
        laserRef.current.position.x = THREE.MathUtils.lerp(-1.8, 1.8, t)
        laserRef.current.visible = true
      } else {
        laserRef.current.visible = false
      }
    }

    // Shards fracture and drift apart
    shardRefs.current.forEach((ref, i) => {
      if (ref) {
        if (t < 1.0) {
          ref.position.y = 0
          ref.position.z = 0
        } else {
          const p = (t - 1.0) / 2.0
          ref.position.y = THREE.MathUtils.lerp(0, (i % 2 === 0 ? 0.25 : -0.25), p)
          ref.position.z = THREE.MathUtils.lerp(0, 0.2, p)
        }
      }
    })
  })

  return (
    <group>
      {/* Laser Cut Plane */}
      <mesh ref={laserRef} position={[-1.8, 0, 0]}>
        <boxGeometry args={[0.04, 1.6, 1.0]} />
        <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3} transparent opacity={0.8} />
      </mesh>

      {/* Fractured Variable Sized Shards */}
      {shardSpecs.map((spec, i) => (
        <group
          key={i}
          ref={(el) => {
            if (el) shardRefs.current[i] = el
          }}
          position={[spec.x, 0, 0]}
        >
          <mesh>
            <boxGeometry args={[spec.width, 1.1, 0.6]} />
            <meshStandardMaterial
              color={spec.color}
              metalness={0.85}
              roughness={0.25}
              emissive={spec.color === '#7A1F2B' ? '#421118' : '#000000'}
              emissiveIntensity={spec.color === '#7A1F2B' ? 0.6 : 0.2}
            />
          </mesh>

          <mesh>
            <boxGeometry args={[spec.width + 0.02, 1.12, 0.62]} />
            <meshStandardMaterial color="#A6A9AE" wireframe transparent opacity={0.4} />
          </mesh>

          <Html position={[0, 0.8, 0]} center distanceFactor={8}>
            <div className="px-1.5 py-0.5 rounded bg-[#000000]/95 border border-[#26262B] font-mono text-[8px] text-[#EDE8DF] whitespace-nowrap">
              {spec.label}
            </div>
          </Html>
        </group>
      ))}

      <Html position={[0, -1.2, 0]} center distanceFactor={8}>
        <div className="px-2.5 py-1 rounded bg-[#000000]/95 border border-[#7A1F2B] font-mono text-[9px] text-[#EDE8DF] shadow-md whitespace-nowrap">
          Rolling Gear Hash Boundary Cuts
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE 3: SHA-256 CAS & DEDUPLICATION MERGE                               */
/* ========================================================================= */
function Stage3CASDedup() {
  const dupRef = useRef<THREE.Group>(null)
  const [isDeduplicated, setIsDeduplicated] = useState<boolean>(false)

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.7) % 3.0

    if (dupRef.current) {
      if (t < 1.2) {
        // Incoming duplicate moves toward the matching stored chunk
        const p = t / 1.2
        dupRef.current.position.x = THREE.MathUtils.lerp(1.8, 0, p)
        dupRef.current.position.y = THREE.MathUtils.lerp(0.8, 0, p)
        dupRef.current.scale.setScalar(1)
        setIsDeduplicated(false)
      } else {
        // Merge & dissolve with 0 I/O callout
        const p = (t - 1.2) / 1.8
        dupRef.current.scale.setScalar(Math.max(0, 1 - p * 1.5))
        setIsDeduplicated(true)
      }
    }
  })

  return (
    <group>
      {/* Existing Stored Chunks in CAS Directory */}
      <group position={[-1.2, 0, 0]}>
        <mesh>
          <boxGeometry args={[0.7, 0.9, 0.45]} />
          <meshStandardMaterial color="#19191C" metalness={0.85} roughness={0.25} />
        </mesh>
        <Html position={[0, 0.65, 0]} center distanceFactor={8}>
          <div className="px-1.5 py-0.5 rounded bg-[#000000] border border-[#26262B] font-mono text-[7px] text-[#8B8B8F]">
            8f3a...91c2
          </div>
        </Html>
      </group>

      {/* Target Stored Chunk that will match */}
      <group position={[0, 0, 0]}>
        <mesh>
          <boxGeometry args={[0.7, 0.9, 0.45]} />
          <meshStandardMaterial
            color="#7A1F2B"
            metalness={0.9}
            roughness={0.2}
            emissive="#421118"
            emissiveIntensity={isDeduplicated ? 0.8 : 0.3}
          />
        </mesh>
        <Html position={[0, 0.65, 0]} center distanceFactor={8}>
          <div className="px-1.5 py-0.5 rounded bg-[#000000] border border-[#7A1F2B] font-mono text-[7px] text-[#EDE8DF] font-bold">
            17bd...82a1 (CAS)
          </div>
        </Html>
      </group>

      {/* Incoming Duplicate Chunk */}
      <group ref={dupRef} position={[1.8, 0.8, 0]}>
        <mesh>
          <boxGeometry args={[0.65, 0.85, 0.4]} />
          <meshStandardMaterial
            color="#EDE8DF"
            emissive="#7A1F2B"
            emissiveIntensity={1.5}
            transparent
            opacity={0.85}
          />
        </mesh>
        <Html position={[0, 0.65, 0]} center distanceFactor={8}>
          <div className="px-1.5 py-0.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-mono text-[7px] font-bold shadow-xs">
            Duplicate 17bd...
          </div>
        </Html>
      </group>

      <Html position={[0, -1.0, 0]} center distanceFactor={8}>
        <div className={`px-2.5 py-1 rounded font-mono text-[9px] shadow-md whitespace-nowrap transition-all ${isDeduplicated ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold border border-[#EDE8DF]' : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B]'
          }`}>
          {isDeduplicated ? '✓ SHA-256 CAS Hit · 0 Disk I/O (Deduplicated)' : 'Computing SHA-256 Digest...'}
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE 4: CONSISTENT HASH RING & ROUTING                                   */
/* ========================================================================= */
function Stage4HashRingRouting() {
  const packetRef = useRef<THREE.Mesh>(null)
  const RING_R = 1.6

  const nodes = [
    { angle: 0, label: 'N1', role: 'Replica' },
    { angle: 60, label: 'N2', role: 'Primary' },
    { angle: 120, label: 'N3', role: 'Replica' },
    { angle: 180, label: 'N4', role: 'Standby' },
    { angle: 240, label: 'N5', role: 'Standby' },
    { angle: 300, label: 'N6', role: 'Replica' },
  ]

  const targetRad = (60 * Math.PI) / 180
  const targetX = Math.cos(targetRad) * RING_R
  const targetY = Math.sin(targetRad) * RING_R

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.8) % 2.5
    if (packetRef.current) {
      if (t < 0.4) {
        packetRef.current.position.set(0, 0, 0)
      } else {
        const p = (t - 0.4) / 1.6
        packetRef.current.position.x = THREE.MathUtils.lerp(0, targetX, p)
        packetRef.current.position.y = THREE.MathUtils.lerp(0, targetY, p)
      }
    }
  })

  return (
    <group>
      {/* 32-Bit Ring Torus */}
      <mesh>
        <torusGeometry args={[RING_R, 0.04, 16, 48]} />
        <meshStandardMaterial color="#26262B" metalness={0.8} />
      </mesh>

      {/* Ring Nodes */}
      {nodes.map((n, i) => {
        const rad = (n.angle * Math.PI) / 180
        const x = Math.cos(rad) * RING_R
        const y = Math.sin(rad) * RING_R
        const isPrimary = n.role === 'Primary'

        return (
          <group key={i} position={[x, y, 0]}>
            <mesh>
              <cylinderGeometry args={[0.18, 0.18, 0.22, 16]} />
              <meshStandardMaterial
                color={isPrimary ? '#7A1F2B' : '#A6A9AE'}
                metalness={0.9}
                roughness={0.2}
                emissive={isPrimary ? '#421118' : '#000000'}
                emissiveIntensity={isPrimary ? 0.7 : 0.2}
              />
            </mesh>
            <Html position={[0, 0.35, 0]} center distanceFactor={8}>
              <span className={`text-[8px] font-mono font-bold px-1 rounded ${isPrimary ? 'bg-[#7A1F2B] text-[#EDE8DF]' : 'bg-[#000000] text-[#A6A9AE]'
                }`}>
                {n.label}
              </span>
            </Html>
          </group>
        )
      })}

      {/* Traveling Chunk Packet */}
      <mesh ref={packetRef} position={[0, 0, 0]}>
        <sphereGeometry args={[0.1, 16, 16]} />
        <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3.5} />
      </mesh>

      <Html position={[0, -1.2, 0]} center distanceFactor={8}>
        <div className="px-2.5 py-1 rounded bg-[#000000]/95 border border-[#7A1F2B] font-mono text-[9px] text-[#EDE8DF] shadow-md whitespace-nowrap">
          Hash(Chunk) → Routed to Primary Tumbler N2
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE 5: QUORUM REPLICATION (N=3, W=2, R=2)                               */
/* ========================================================================= */
function Stage5QuorumReplication() {
  const ack1Ref = useRef<THREE.Mesh>(null)
  const ack2Ref = useRef<THREE.Mesh>(null)
  const trail3Ref = useRef<THREE.Mesh>(null)
  const [quorumMet, setQuorumMet] = useState<boolean>(false)

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.8) % 3.0

    // ACK 1 from Node 1 (Fast ACK)
    if (ack1Ref.current) {
      if (t < 0.6) {
        ack1Ref.current.position.set(-1.4, 0.7, 0)
      } else {
        const p = Math.min(1, (t - 0.6) / 0.8)
        ack1Ref.current.position.x = THREE.MathUtils.lerp(-1.4, 0, p)
        ack1Ref.current.position.y = THREE.MathUtils.lerp(0.7, 0, p)
      }
    }

    // ACK 2 from Node 2 (Fast ACK)
    if (ack2Ref.current) {
      if (t < 0.8) {
        ack2Ref.current.position.set(0, 1.2, 0)
      } else {
        const p = Math.min(1, (t - 0.8) / 0.8)
        ack2Ref.current.position.y = THREE.MathUtils.lerp(1.2, 0, p)
      }
    }

    // Trail 3 to Node 3 (Slow Peer Trail)
    if (trail3Ref.current) {
      const p = (t / 3.0)
      trail3Ref.current.position.x = THREE.MathUtils.lerp(0, 1.4, p)
      trail3Ref.current.position.y = THREE.MathUtils.lerp(0, -0.7, p)
    }

    setQuorumMet(t >= 1.6)
  })

  return (
    <group>
      {/* Central Coordinator */}
      <group position={[0, 0, 0]}>
        <mesh>
          <cylinderGeometry args={[0.3, 0.3, 0.25, 24]} />
          <meshStandardMaterial color="#7A1F2B" metalness={0.9} roughness={0.2} emissive="#421118" emissiveIntensity={0.6} />
        </mesh>
        <Html position={[0, 0.4, 0]} center distanceFactor={8}>
          <span className="text-[8px] font-mono font-bold px-1.5 py-0.5 rounded bg-[#000000] text-[#EDE8DF] border border-[#7A1F2B]">
            Coordinator
          </span>
        </Html>
      </group>

      {/* 3 Target Replica Nodes (N=3) */}
      {[
        { pos: [-1.4, 0.7, 0] as [number, number, number], label: 'Replica 1 (ACK 1)' },
        { pos: [0, 1.2, 0] as [number, number, number], label: 'Replica 2 (ACK 2)' },
        { pos: [1.4, -0.7, 0] as [number, number, number], label: 'Replica 3 (Peer)' },
      ].map((n, i) => (
        <group key={i} position={n.pos}>
          <mesh>
            <cylinderGeometry args={[0.22, 0.22, 0.2, 16]} />
            <meshStandardMaterial color="#A6A9AE" metalness={0.85} roughness={0.2} />
          </mesh>
          <Html position={[0, 0.35, 0]} center distanceFactor={8}>
            <span className="text-[7px] font-mono text-[#8B8B8F] px-1 rounded bg-[#000000] border border-[#26262B] whitespace-nowrap">
              {n.label}
            </span>
          </Html>
        </group>
      ))}

      {/* Returning ACK 1 */}
      <mesh ref={ack1Ref} position={[-1.4, 0.7, 0]}>
        <sphereGeometry args={[0.08, 16, 16]} />
        <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3} />
      </mesh>

      {/* Returning ACK 2 */}
      <mesh ref={ack2Ref} position={[0, 1.2, 0]}>
        <sphereGeometry args={[0.08, 16, 16]} />
        <meshStandardMaterial color="#EDE8DF" emissive="#7A1F2B" emissiveIntensity={3} />
      </mesh>

      {/* In-Flight Trail 3 */}
      <mesh ref={trail3Ref} position={[0, 0, 0]}>
        <sphereGeometry args={[0.06, 16, 16]} />
        <meshStandardMaterial color="#A6A9AE" emissive="#A6A9AE" emissiveIntensity={1.5} />
      </mesh>

      <Html position={[0, -1.2, 0]} center distanceFactor={8}>
        <div className={`px-2.5 py-1 rounded font-mono text-[9px] shadow-md whitespace-nowrap transition-all ${quorumMet ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold border border-[#EDE8DF]' : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B]'
          }`}>
          {quorumMet ? '✓ W=2 ACKs Confirmed · Quorum Met' : 'Waiting for W=2 ACKs...'}
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE 6: WAL COMMIT & VECTOR CLOCK SEAL STAMP                            */
/* ========================================================================= */
function Stage6WALSeal() {
  const stampRef = useRef<THREE.Group>(null)
  const [isSealed, setIsSealed] = useState<boolean>(false)

  useFrame(({ clock }) => {
    const t = (clock.getElapsedTime() * 0.8) % 3.0

    if (stampRef.current) {
      if (t < 1.2) {
        // Stamp descends onto the ledger block
        const p = t / 1.2
        stampRef.current.position.y = THREE.MathUtils.lerp(1.6, 0.35, p)
        setIsSealed(false)
      } else {
        // Stamp locked on the journal
        stampRef.current.position.y = 0.35
        setIsSealed(true)
      }
    }
  })

  return (
    <group>
      {/* WAL Journal Ledger Block */}
      <mesh position={[0, -0.2, 0]}>
        <boxGeometry args={[2.0, 0.4, 1.4]} />
        <meshStandardMaterial color="#19191C" metalness={0.9} roughness={0.2} />
      </mesh>
      <mesh position={[0, -0.2, 0]}>
        <boxGeometry args={[2.02, 0.42, 1.42]} />
        <meshStandardMaterial color="#26262B" wireframe />
      </mesh>

      {/* Descending Garnet/Steel Seal Stamp */}
      <group ref={stampRef} position={[0, 1.6, 0]}>
        <mesh position={[0, 0.2, 0]}>
          <cylinderGeometry args={[0.45, 0.55, 0.3, 32]} />
          <meshStandardMaterial color="#7A1F2B" metalness={0.95} roughness={0.15} emissive="#421118" emissiveIntensity={0.6} />
        </mesh>
        <mesh position={[0, 0.4, 0]}>
          <cylinderGeometry args={[0.2, 0.2, 0.2, 16]} />
          <meshStandardMaterial color="#A6A9AE" metalness={0.9} roughness={0.1} />
        </mesh>
        <mesh position={[0, 0.05, 0]}>
          <torusGeometry args={[0.42, 0.04, 16, 32]} />
          <meshStandardMaterial color="#EDE8DF" metalness={0.95} roughness={0.1} />
        </mesh>
      </group>

      <Html position={[0, -0.9, 0]} center distanceFactor={8}>
        <div className={`px-2.5 py-1 rounded font-mono text-[9px] shadow-md whitespace-nowrap transition-all ${isSealed ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold border border-[#EDE8DF]' : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B]'
          }`}>
          {isSealed ? '✓ WAL fsync Committed · Vector Clock Sealed (200 OK)' : 'Flushing WAL Journal...'}
        </div>
      </Html>
    </group>
  )
}

/* ========================================================================= */
/* STAGE SELECTOR 3D SCENE WITH CROSSFADE                                    */
/* ========================================================================= */
function PipelineStage3DVisual({ stageId }: { stageId: number }) {
  return (
    <group>
      <ambientLight intensity={0.6} />
      <directionalLight position={[5, 8, 5]} intensity={1.5} color="#EDE8DF" />
      <pointLight position={[0, 0, 0]} intensity={2.0} color="#7A1F2B" distance={6} />

      {stageId === 1 && <Stage1S3Ingestion />}
      {stageId === 2 && <Stage2FastCDCChunking />}
      {stageId === 3 && <Stage3CASDedup />}
      {stageId === 4 && <Stage4HashRingRouting />}
      {stageId === 5 && <Stage5QuorumReplication />}
      {stageId === 6 && <Stage6WALSeal />}
    </group>
  )
}

export function Pipeline3DScroll() {
  const [activeStageId, setActiveStageId] = useState<number>(1)
  const [showCode, setShowCode] = useState<boolean>(true)
  const [prefersReducedMotion, setPrefersReducedMotion] = useState<boolean>(false)

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    setPrefersReducedMotion(mediaQuery.matches)
    const handler = (e: MediaQueryListEvent) => setPrefersReducedMotion(e.matches)
    mediaQuery.addEventListener('change', handler)
    return () => mediaQuery.removeEventListener('change', handler)
  }, [])

  const currentStage = stages.find((s) => s.id === activeStageId) || stages[0]

  return (
    <div className="w-full flex flex-col space-y-8 font-sans">
      {/* Stage Navigator Tabs */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2 w-full">
        {stages.map((stage) => {
          const isCurrent = stage.id === activeStageId
          return (
            <button
              key={stage.id}
              onClick={() => setActiveStageId(stage.id)}
              className={`p-3 rounded-lg border text-left transition-all cursor-pointer font-mono ${isCurrent
                  ? 'bg-[#19191C] border-[#7A1F2B] text-[#EDE8DF] shadow-xs ring-1 ring-[#7A1F2B]'
                  : 'bg-[#000000] border-[#26262B] text-[#8B8B8F] hover:border-[#34343A] hover:text-[#EDE8DF]'
                }`}
            >
              <div className="flex items-center justify-between text-[11px]">
                <span className={isCurrent ? 'text-[#EDE8DF] font-bold' : 'text-[#8B8B8F]'}>
                  Stage 0{stage.id}
                </span>
                {isCurrent && <span className="w-1.5 h-1.5 rounded-full bg-[#7A1F2B]" />}
              </div>
              <div className="text-xs font-semibold mt-1 truncate">
                {stage.name.split('. ')[1]}
              </div>
            </button>
          )
        })}
      </div>

      {/* Main Interactive Stage Display */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
        {/* Left: 3D Concept-Accurate Stage Mechanism Visualization */}
        <div className="lg:col-span-5 flex flex-col h-[380px] lg:h-auto min-h-[380px] bg-[#19191C] border border-[#26262B] rounded-xl overflow-hidden relative">
          {prefersReducedMotion ? (
            <div className="w-full h-full flex flex-col items-center justify-center p-6 text-center bg-[#000000]">
              <span className="text-sm font-serif font-bold text-[#EDE8DF]">{currentStage.name}</span>
              <span className="text-xs font-mono text-[#8B8B8F] mt-1">{currentStage.subtitle}</span>
            </div>
          ) : (
            <Canvas camera={{ position: [0, 0, 4.2], fov: 45 }}>
              <color attach="background" args={['#000000']} />
              <PipelineStage3DVisual stageId={activeStageId} />
              <OrbitControls enableZoom={false} enablePan={false} />
            </Canvas>
          )}

          <div className="absolute bottom-3 left-3 right-3 flex items-center justify-between font-mono text-[10px] text-[#8B8B8F] bg-[#000000]/90 px-3 py-1.5 rounded border border-[#26262B]">
            <span>Concept-Accurate 3D Model</span>
            <span className="text-[#EDE8DF] font-semibold">Stage {activeStageId} of 6</span>
          </div>
        </div>

        {/* Right: Narrative Explanation & Code Accordion */}
        <div className="lg:col-span-7 flex flex-col justify-between bg-[#19191C] border border-[#26262B] rounded-xl p-6 space-y-6">
          <div className="space-y-4">
            <div className="space-y-1">
              <div className="inline-flex items-center gap-1.5 text-xs font-mono text-[#A6A9AE] font-semibold">
                <span>STAGE 0{currentStage.id}</span>
                <span>·</span>
                <span>{currentStage.subtitle}</span>
              </div>
              <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">
                {currentStage.name}
              </h2>
            </div>

            <p className="text-sm font-sans text-[#EDE8DF]/90 leading-relaxed">
              {currentStage.description}
            </p>

            {/* Bullet Highlights */}
            <div className="space-y-2 pt-2">
              {currentStage.details.map((detail, idx) => (
                <div key={idx} className="flex items-start gap-2.5 text-xs font-mono text-[#8B8B8F]">
                  <CheckCircle2 className="w-3.5 h-3.5 text-[#7A1F2B] shrink-0 mt-0.5" />
                  <span>{detail}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Expandable "Read the Code" Accordion with Colorful CodeBlock */}
          <div className="border border-[#26262B] rounded-lg overflow-hidden bg-[#000000]">
            <button
              onClick={() => setShowCode(!showCode)}
              className="w-full px-4 py-2.5 flex items-center justify-between text-xs font-mono text-[#EDE8DF] hover:bg-[#19191C] transition-colors cursor-pointer border-b border-[#26262B]"
            >
              <div className="flex items-center gap-2">
                <Code className="w-3.5 h-3.5 text-[#A6A9AE]" />
                <span className="font-semibold">{currentStage.codeFile}</span>
              </div>
              <span className="text-[11px] text-[#8B8B8F]">
                {showCode ? 'Hide Code' : 'Read Go Code'}
              </span>
            </button>

            {showCode && (
              <div className="p-3 text-xs">
                <CodeBlock code={currentStage.codeSnippet} language="go" filename={currentStage.codeFile} />
              </div>
            )}
          </div>

          {/* Step Navigation Controls */}
          <div className="flex items-center justify-between border-t border-[#26262B] pt-4 font-mono text-xs">
            <button
              disabled={activeStageId === 1}
              onClick={() => setActiveStageId((prev) => Math.max(1, prev - 1))}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded border border-[#26262B] text-[#8B8B8F] hover:text-[#EDE8DF] hover:border-[#34343A] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
              <span>Previous Stage</span>
            </button>

            <span className="text-[11px] text-[#8B8B8F]">
              {activeStageId} / {stages.length}
            </span>

            <button
              disabled={activeStageId === stages.length}
              onClick={() => setActiveStageId((prev) => Math.min(stages.length, prev + 1))}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-semibold hover:bg-[#912735] disabled:opacity-30 disabled:cursor-not-allowed transition-colors cursor-pointer"
            >
              <span>Next Stage</span>
              <ChevronRight className="w-3.5 h-3.5 stroke-[2.5]" />
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
