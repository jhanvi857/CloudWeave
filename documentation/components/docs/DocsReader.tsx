'use client'

import React, { useState } from 'react'
import { CodeBlock } from '../ui/CodeBlock'
import { Layers, HardDrive, Server, ShieldCheck, Zap, Cpu } from 'lucide-react'

interface DocsReaderProps {
  initialTopic?: string
}

export function DocsReader({ initialTopic = 'fastcdc' }: DocsReaderProps) {
  const [topic, setTopic] = useState<string>(initialTopic)

  const topics = [
    { id: 'fastcdc', label: 'FastCDC Chunking' },
    { id: 'cas', label: 'SHA-256 CAS & Dedup' },
    { id: 'ring', label: 'Consistent Hash Ring' },
    { id: 'quorum', label: 'N / W / R Quorum' },
    { id: 'erasure', label: 'Erasure Coding (RS)' },
    { id: 'repair', label: 'Failure Detection & Repair' },
    { id: 'wal', label: 'WAL & Durability' },
  ]

  return (
    <div className="w-full max-w-4xl mx-auto space-y-8 py-4 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Topic Switcher Bar */}
      <div className="flex items-center gap-1.5 overflow-x-auto pb-2 border-b border-[#26262B] font-mono text-xs">
        {topics.map((t) => (
          <button
            key={t.id}
            onClick={() => setTopic(t.id)}
            className={`px-3 py-1.5 rounded shrink-0 font-medium transition-colors cursor-pointer ${
              topic === t.id
                ? 'bg-[#19191C] text-[#EDE8DF] border border-[#7A1F2B] font-bold'
                : 'text-[#8B8B8F] hover:text-[#EDE8DF] hover:bg-[#19191C]'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Article Content */}
      {topic === 'fastcdc' && (
        <article className="space-y-6 text-sm text-[#8B8B8F] leading-relaxed">
          <div>
            <h1 className="text-2xl font-serif font-bold text-[#EDE8DF] tracking-tight">FastCDC Content-Defined Chunking</h1>
            <p className="text-xs font-mono text-[#A6A9AE] mt-1">Rolling Gear Hash · Boundary Resilience · Variable 16KB-256KB Chunks</p>
          </div>

          <p>
            Fixed-size chunking (such as cutting an object every 1 MB) breaks completely when a single byte is inserted or deleted at the beginning of a file. Every subsequent boundary shifts, causing 100% deduplication failure. FastCDC solves this by computing a rolling Gear hash over the byte stream to declare chunk cut points based on content patterns rather than fixed offsets.
          </p>

          <div className="p-4 rounded-xl border border-[#26262B] bg-[#19191C] space-y-2 font-mono text-xs">
            <div className="font-bold text-[#EDE8DF]">How CloudWeave Implements FastCDC</div>
            <p className="text-[#8B8B8F]">
              As bytes stream from the HTTP request into CloudWeave, the chunker updates a 64-bit fingerprint:
              <code className="text-[#EDE8DF] ml-1">fp = (fp &lt;&lt; 1) + gearTable[b]</code>. When the lower bits match the mask threshold within the minimum (16 KB) and maximum (256 KB) size window, a chunk is finalized and hashed with SHA-256.
            </p>
          </div>

          <CodeBlock
            code={`// internal/chunk/cdc.go
func FastCDC(data []byte) []Chunk {
    var chunks []Chunk
    fp := uint64(0)
    start := 0
    for i, b := range data {
        fp = (fp << 1) + gearTable[b]
        size := i - start
        if size >= MinChunkSize && ((fp & Mask) == 0 || size >= MaxChunkSize) {
            chunkData := data[start : i+1]
            hash := sha256.Sum256(chunkData)
            chunks = append(chunks, Chunk{
                ID:   hex.EncodeToString(hash[:]),
                Data: chunkData,
                Size: len(chunkData),
            })
            start = i + 1
            fp = 0
        }
    }
    return chunks
}`}
            language="go"
            filename="internal/chunk/cdc.go"
          />
        </article>
      )}

      {topic === 'cas' && (
        <article className="space-y-6 text-sm text-[#8B8B8F] leading-relaxed">
          <div>
            <h1 className="text-2xl font-serif font-bold text-[#EDE8DF] tracking-tight">Content-Addressable Storage (CAS)</h1>
            <p className="text-xs font-mono text-[#A6A9AE] mt-1">Cryptographic Integrity · Self-Verifying Blocks · Zero-I/O Deduplication</p>
          </div>

          <p>
            In CloudWeave, chunk identifiers are not random UUIDs; each chunk ID is strictly the hexadecimal SHA-256 hash of its binary payload: <code className="text-[#EDE8DF]">ID = hex(sha256(data))</code>. This guarantees that data is self-verifying.
          </p>

          <CodeBlock
            code={`// internal/storage/diskstore.go
func (s *DiskStore) Put(chunkID string, data []byte) error {
    h := sha256.Sum256(data)
    computed := hex.EncodeToString(h[:])
    if computed != chunkID {
        return fmt.Errorf("CAS mismatch: expected %s, got %s", chunkID, computed)
    }
    return s.atomicWriteFile(s.chunkPath(chunkID), data)
}`}
            language="go"
            filename="internal/storage/diskstore.go"
          />
        </article>
      )}
    </div>
  )
}
