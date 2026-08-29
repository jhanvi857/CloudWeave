'use client'

import React from 'react'
import Link from 'next/link'
import { CodeBlock } from '../ui/CodeBlock'
import {
  Layers,
  HardDrive,
  Server,
  ShieldCheck,
  ArrowRight,
} from 'lucide-react'

interface ConceptTopic {
  id: string
  title: string
  subtitle: string
  playgroundTab: 'pipeline' | 'failure' | 'dedup' | 'erasure'
  whyItExists: string
  howItWorks: string
  tradeoffs: string
  failureBehavior: string
  lineArt: React.ReactNode
  code: string
  icon: any
}

function FastCDCLineArt() {
  return (
    <svg viewBox="0 0 460 90" className="w-full h-auto" fill="none" stroke="currentColor">
      <rect x="10" y="25" width="440" height="28" rx="3" fill="#19191C" stroke="#26262B" strokeWidth="1" />
      <line x1="110" y1="18" x2="110" y2="60" stroke="#7A1F2B" strokeWidth="2" strokeDasharray="3 3" />
      <line x1="260" y1="18" x2="260" y2="60" stroke="#7A1F2B" strokeWidth="2" strokeDasharray="3 3" />
      <line x1="370" y1="18" x2="370" y2="60" stroke="#7A1F2B" strokeWidth="2" strokeDasharray="3 3" />
      <text x="50" y="43" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Chunk 1 (16KB)</text>
      <text x="185" y="43" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Chunk 2 (64KB - Rolling Gear Cut)</text>
      <text x="315" y="43" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Chunk 3 (32KB)</text>
      <text x="415" y="43" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Chunk 4</text>
      <text x="230" y="78" fill="#8B8B8F" fontSize="9" fontFamily="monospace" textAnchor="middle">Content-defined boundaries survive 1-byte edits</text>
    </svg>
  )
}

function CASLineArt() {
  return (
    <svg viewBox="0 0 460 90" className="w-full h-auto" fill="none" stroke="currentColor">
      <rect x="20" y="25" width="100" height="30" rx="3" fill="#19191C" stroke="#26262B" />
      <text x="70" y="44" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Byte Payload</text>

      <line x1="125" y1="40" x2="165" y2="40" stroke="#7A1F2B" strokeWidth="1.5" />
      <polygon points="165,37 172,40 165,43" fill="#7A1F2B" />

      <rect x="175" y="25" width="110" height="30" rx="3" fill="#19191C" stroke="#7A1F2B" />
      <text x="230" y="44" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">SHA-256 Digest</text>

      <line x1="290" y1="40" x2="330" y2="40" stroke="#7A1F2B" strokeWidth="1.5" />
      <polygon points="330,37 337,40 330,43" fill="#7A1F2B" />

      <rect x="340" y="25" width="100" height="30" rx="3" fill="#19191C" stroke="#26262B" />
      <text x="390" y="44" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">/chunks/8f/3a...</text>
    </svg>
  )
}

function RingLineArt() {
  return (
    <svg viewBox="0 0 460 90" className="w-full h-auto" fill="none" stroke="currentColor">
      <circle cx="230" cy="45" r="32" stroke="#26262B" strokeWidth="1.5" />
      <circle cx="230" cy="45" r="32" stroke="#7A1F2B" strokeWidth="1.5" strokeDasharray="6 12" />
      <circle cx="230" cy="13" r="5" fill="#7A1F2B" />
      <text x="230" y="8" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="middle">N1 (0°)</text>
      <circle cx="262" cy="45" r="5" fill="#A6A9AE" />
      <text x="285" y="48" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="start">N2 (90°)</text>
      <circle cx="230" cy="77" r="5" fill="#A6A9AE" />
      <text x="230" y="88" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="middle">N3 (180°)</text>
      <circle cx="198" cy="45" r="5" fill="#A6A9AE" />
      <text x="175" y="48" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="end">N4 (270°)</text>
    </svg>
  )
}

function QuorumLineArt() {
  return (
    <svg viewBox="0 0 460 90" className="w-full h-auto" fill="none" stroke="currentColor">
      <rect x="20" y="30" width="80" height="28" rx="3" fill="#19191C" stroke="#7A1F2B" />
      <text x="60" y="48" fill="#EDE8DF" fontSize="10" fontFamily="monospace" textAnchor="middle">Coordinator</text>

      <line x1="105" y1="44" x2="180" y2="20" stroke="#7A1F2B" strokeWidth="1.5" />
      <line x1="105" y1="44" x2="180" y2="44" stroke="#7A1F2B" strokeWidth="1.5" />
      <line x1="105" y1="44" x2="180" y2="68" stroke="#A6A9AE" strokeWidth="1.5" strokeDasharray="3 3" />

      <rect x="185" y="10" width="70" height="20" rx="2" fill="#19191C" stroke="#7A1F2B" />
      <text x="220" y="24" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="middle">Node 1 (ACK)</text>

      <rect x="185" y="34" width="70" height="20" rx="2" fill="#19191C" stroke="#7A1F2B" />
      <text x="220" y="48" fill="#EDE8DF" fontSize="9" fontFamily="monospace" textAnchor="middle">Node 2 (ACK)</text>

      <rect x="185" y="58" width="70" height="20" rx="2" fill="#19191C" stroke="#26262B" />
      <text x="220" y="72" fill="#8B8B8F" fontSize="9" fontFamily="monospace" textAnchor="middle">Node 3 (Peer)</text>

      <text x="350" y="48" fill="#EDE8DF" fontSize="10" fontFamily="monospace">W=2 Satisfied (Quorum)</text>
    </svg>
  )
}

const conceptsData: ConceptTopic[] = [
  {
    id: 'fastcdc',
    title: 'FastCDC Content-Defined Chunking',
    subtitle: 'Rolling Gear hash over byte streams to eliminate byte-shift boundary corruption',
    playgroundTab: 'dedup',
    whyItExists:
      'Fixed-size chunking (e.g. 1MB blocks) fails completely when 1 byte is inserted at the beginning of a file: all subsequent boundaries shift by 1 byte, destroying 100% of deduplication. FastCDC declares chunk boundaries based on content fingerprints.',
    howItWorks:
      'CloudWeave implements FastCDC in Go using a 256-entry Gear lookup table. As bytes stream in, it updates fp = (fp << 1) + gearTable[b]. When (fp & mask) == 0 within min (16KB) and max (256KB) bounds, a boundary cut is declared.',
    tradeoffs:
      'Requires ~5-8% more CPU cycles per ingested byte than static slicing, but yields 80%+ deduplication reuse on modified object versions.',
    failureBehavior:
      'If an unaligned stream is ingested, FastCDC enforces the hard maximum chunk size (256KB) to prevent unbounded buffer growth.',
    lineArt: <FastCDCLineArt />,
    code: `// internal/chunk/fastcdc.go
func (c *Chunker) Split(data []byte) []Chunk {
    var chunks []Chunk
    cursor := 0
    n := len(data)

    for cursor < n {
        remaining := n - cursor
        if remaining <= MinChunkSize {
            chunks = append(chunks, c.createChunk(data[cursor:]))
            break
        }
        fp := uint32(0)
        split := cursor + MinChunkSize
        for split < cursor+MaxChunkSize && split < n {
            fp = (fp << 1) + GearMatrix[data[split]]
            if (fp & MaskNormalized) == 0 {
                break
            }
            split++
        }
        chunks = append(chunks, c.createChunk(data[cursor:split]))
        cursor = split
    }
    return chunks
}`,
    icon: Layers,
  },
  {
    id: 'cas',
    title: 'Content-Addressed Storage (CAS) & Deduplication',
    subtitle: 'Self-verifying chunk IDs derived directly from SHA-256 content hashes',
    playgroundTab: 'dedup',
    whyItExists:
      'Traditional storage engines assign arbitrary UUIDs to objects. If a disk sector silently corrupts (bit rot), the system cannot detect it. In CAS, chunk ID = hex(SHA256(data)). The data is self-verifying and tamper-evident.',
    howItWorks:
      'Every chunk is saved to $DATA_DIR/chunks/ab/cd/abcdef1234... on disk. When multiple files or versions upload identical chunks, CloudWeave detects the matching SHA-256 key and reuses the existing storage block without writing duplicate bytes to disk.',
    tradeoffs:
      'SHA-256 compute overhead is minimal on modern CPUs with hardware SHA extensions (>1.5 GB/s throughput).',
    failureBehavior:
      'Corrupted chunks immediately fail the SHA-256 verification check on read, prompting the coordinator to fetch from an alternate replica and queue background repair.',
    lineArt: <CASLineArt />,
    code: `// internal/storage/diskstore.go
func (s *DiskStore) Put(chunkID string, data []byte) error {
    h := sha256.Sum256(data)
    computed := hex.EncodeToString(h[:])
    if computed != chunkID {
        return fmt.Errorf("CAS mismatch: expected %s, got %s", chunkID, computed)
    }
    return s.atomicWriteFile(s.chunkPath(chunkID), data)
}`,
    icon: HardDrive,
  },
  {
    id: 'ring',
    title: 'Consistent Hashing with 150 Virtual Nodes',
    subtitle: 'Deterministic chunk placement minimizing key movement when nodes join or leave',
    playgroundTab: 'pipeline',
    whyItExists:
      'Modulo hashing (key % N) remaps ~100% of all keys when node count changes. Consistent hashing ensures adding or removing 1 node only relocates 1/N fraction of keys, keeping 90%+ of cluster data in place.',
    howItWorks:
      '150 virtual nodes per physical node are mapped onto a 32-bit circular integer ring. For key lookup, binary search locates the first virtual node clockwise from the key hash.',
    tradeoffs:
      'Virtual nodes require a small in-memory slice for binary search (O(log(150N)) < 1 microsecond).',
    failureBehavior:
      'When a node goes offline, only its virtual nodes disappear; queries immediately fall through to the next physical nodes on the ring.',
    lineArt: <RingLineArt />,
    code: `// internal/ring/ring.go
func (r *HashRing) GetNodesForKey(key string, n int) []Node {
    hash := hashKey(key)
    idx := sort.Search(len(r.vnodes), func(i int) bool {
        return r.vnodes[i].hash >= hash
    })
    return r.collectNextNDistinctNodes(idx, n)
}`,
    icon: Server,
  },
  {
    id: 'quorum',
    title: 'Dynamo-Style Quorum (N / W / R)',
    subtitle: 'Decentralized quorum consistency without a single leader bottleneck',
    playgroundTab: 'pipeline',
    whyItExists:
      'Single-leader consensus (like Raft) forces all large data streams through the leader node. Dynamo quorums allow any coordinator node to fan out writes and reads directly to storage nodes.',
    howItWorks:
      'For write quorum (N=3, W=2), the coordinator fans out chunks to 3 nodes concurrently and returns success once 2 nodes acknowledge. For read quorum (R=2), it queries 2 nodes and reconciles versions via vector clocks.',
    tradeoffs:
      'Requires vector clock reconciliation on conflicting concurrent writes.',
    failureBehavior:
      'Tolerates N - W node failures for writes (1 failure under N=3, W=2) and N - R node failures for reads.',
    lineArt: <QuorumLineArt />,
    code: `// internal/coordinator/write.go
func (c *Coordinator) WriteChunk(ctx context.Context, chunk Chunk, nodes []Node) error {
    acks := make(chan error, len(nodes))
    for _, node := range nodes {
        go func(n Node) {
            acks <- c.transport.SendChunk(ctx, n.Address, chunk)
        }(node)
    }
    return c.waitQuorum(acks, c.W)
}`,
    icon: ShieldCheck,
  },
]

export function ConceptsView() {
  return (
    <div className="w-full max-w-4xl mx-auto space-y-12 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Title */}
      <div className="text-center space-y-3 max-w-2xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>Distributed Storage Foundations</span>
        </div>
        <h1 className="text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          Core Concepts & Mechanics
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          Deep architectural explanations of the algorithms and durability models behind CloudWeave.
        </p>
      </div>

      {/* Concepts List */}
      <div className="space-y-8">
        {conceptsData.map((concept) => {
          const Icon = concept.icon
          return (
            <div
              key={concept.id}
              id={concept.id}
              className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-6 scroll-mt-24"
            >
              {/* Header & Playground Link */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-[#26262B] pb-4">
                <div className="flex items-center gap-3">
                  <div className="w-9 h-9 rounded bg-[#000000] border border-[#26262B] flex items-center justify-center text-[#A6A9AE]">
                    <Icon className="w-4 h-4" />
                  </div>
                  <div>
                    <h2 className="font-serif font-bold text-lg text-[#EDE8DF]">
                      {concept.title}
                    </h2>
                    <p className="text-xs font-mono text-[#8B8B8F]">
                      {concept.subtitle}
                    </p>
                  </div>
                </div>

                <Link
                  href={`/playground`}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded border border-[#26262B] hover:border-[#7A1F2B] bg-[#000000] text-xs font-mono text-[#EDE8DF] transition-colors cursor-pointer self-start sm:self-auto shrink-0"
                >
                  <span>See it live</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </Link>
              </div>

              {/* Line Art Diagram */}
              <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B]">
                {concept.lineArt}
              </div>

              {/* Technical Breakdown Grid */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs">
                <div className="p-3.5 rounded bg-[#000000] border border-[#26262B] space-y-1">
                  <span className="text-[10px] text-[#A6A9AE] font-bold uppercase tracking-wider block">
                    Why it exists
                  </span>
                  <p className="text-xs font-sans text-[#EDE8DF]/90 leading-relaxed">
                    {concept.whyItExists}
                  </p>
                </div>

                <div className="p-3.5 rounded bg-[#000000] border border-[#26262B] space-y-1">
                  <span className="text-[10px] text-[#A6A9AE] font-bold uppercase tracking-wider block">
                    How it works
                  </span>
                  <p className="text-xs font-sans text-[#EDE8DF]/90 leading-relaxed">
                    {concept.howItWorks}
                  </p>
                </div>

                <div className="p-3.5 rounded bg-[#000000] border border-[#26262B] space-y-1">
                  <span className="text-[10px] text-[#8B8B8F] font-bold uppercase tracking-wider block">
                    Trade-offs
                  </span>
                  <p className="text-xs font-sans text-[#EDE8DF]/90 leading-relaxed">
                    {concept.tradeoffs}
                  </p>
                </div>

                <div className="p-3.5 rounded bg-[#000000] border border-[#26262B] space-y-1">
                  <span className="text-[10px] text-[#8B8B8F] font-bold uppercase tracking-wider block">
                    Failure behavior
                  </span>
                  <p className="text-xs font-sans text-[#EDE8DF]/90 leading-relaxed">
                    {concept.failureBehavior}
                  </p>
                </div>
              </div>

              {/* Go Implementation Snippet */}
              <div className="space-y-2">
                <div className="text-[11px] font-mono text-[#8B8B8F]">
                  Go Implementation Snippet:
                </div>
                <CodeBlock code={concept.code} language="go" />
              </div>
            </div>
          )
        })}
      </div>

      {/* Folded Architecture Section */}
      <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-6">
        <div className="border-b border-[#26262B] pb-3 space-y-1">
          <div className="text-xs font-mono text-[#A6A9AE] font-semibold">
            ARCHITECTURE DEEP-DIVE
          </div>
          <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">
            System Layers & Zero-Trust Protocol
          </h2>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs">
          <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B] space-y-2">
            <span className="font-bold text-[#EDE8DF] block text-sm">Write Path Mechanics</span>
            <p className="text-xs font-sans text-[#8B8B8F] leading-relaxed">
              1. Client issues S3 PUT with SigV4 auth header.<br />
              2. Gateway streams body into sync.Pool buffer and runs FastCDC.<br />
              3. Chunks hashed with SHA-256; deduplication index checked.<br />
              4. Unseen chunks routed via consistent ring to N=3 nodes.<br />
              5. W=2 fsync acks satisfy write quorum; manifest written to WAL.
            </p>
          </div>

          <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B] space-y-2">
            <span className="font-bold text-[#EDE8DF] block text-sm">Read Path Mechanics</span>
            <p className="text-xs font-sans text-[#8B8B8F] leading-relaxed">
              1. Client issues S3 GET with SigV4 auth.<br />
              2. Manifest looked up in memory cache (or reconstructed from WAL).<br />
              3. Chunks fetched from nearest healthy replica.<br />
              4. In-memory LRU cache checked before disk I/O.<br />
              5. SHA-256 CAS hash verified before streaming to client.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
