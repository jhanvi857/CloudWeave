'use client'

import React from 'react'

interface ADR {
  id: string
  title: string
  status: 'Accepted' | 'Implemented'
  date: string
  context: string
  decision: string
  consequences: string
}

const adrs: ADR[] = [
  {
    id: 'ADR-001',
    title: 'Selection of Go as the Core Implementation Language',
    status: 'Accepted',
    date: '2026-08',
    context: 'Building a high-throughput distributed storage engine requires fast concurrent I/O, low memory overhead, and static cross-compilation without CGO dependencies.',
    decision: 'Use Go 1.24+ standard library and runtime primitives (goroutines, sync.Pool, net/http).',
    consequences: 'Enables single-binary distribution and lightweight Docker containers under 30MB.',
  },
  {
    id: 'ADR-002',
    title: 'Content-Addressable Storage (CAS) with SHA-256 digests',
    status: 'Accepted',
    date: '2026-08',
    context: 'Traditional object stores rely on arbitrary UUID keys. If a disk sector silently corrupts, data corruption goes undetected until user read errors.',
    decision: 'All chunk keys are strictly derived as hex(SHA-256(chunk_bytes)). Chunks are verified against their hash on disk read.',
    consequences: 'Guarantees bit-rot detection and enables automatic cross-tenant deduplication.',
  },
  {
    id: 'ADR-003',
    title: 'FastCDC Content-Defined Chunking over Fixed Slicing',
    status: 'Accepted',
    date: '2026-08',
    context: 'Fixed-size slicing (e.g., cutting every 1MB) suffers from the boundary-shift problem: inserting 1 byte invalidates 100% of chunk hashes.',
    decision: 'Adopt FastCDC rolling Gear hash algorithm with dynamic 16KB-256KB chunk boundaries.',
    consequences: 'Delivers 80%+ deduplication ratios on file updates, with 485 MB/s deduplicated upload throughput.',
  },
  {
    id: 'ADR-004',
    title: '150 Virtual Nodes per Physical Node on Consistent Hash Ring',
    status: 'Accepted',
    date: '2026-08',
    context: 'With only 3-5 physical nodes, consistent hash rings produce severe data skew if each node only occupies 1 position on the circle.',
    decision: 'Map 150 virtual nodes per physical host onto a 32-bit SHA-256 integer ring.',
    consequences: 'Achieves <5% standard deviation in chunk distribution across all storage hosts.',
  },
  {
    id: 'ADR-005',
    title: 'Decentralized Dynamo Quorum (N/W/R) for Data Path',
    status: 'Accepted',
    date: '2026-08',
    context: 'Routing all multi-GB streaming payloads through a single Raft leader creates an extreme network bottleneck.',
    decision: 'Use decentralized N/W/R quorum fan-out for chunk writes, reserving Raft exclusively for metadata consensus.',
    consequences: 'Parallelizes write traffic across all cluster nodes while guaranteeing strong consistency when W + R > N.',
  },
]

export function ADRsView() {
  return (
    <div className="w-full max-w-4xl mx-auto space-y-10 py-4 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Header */}
      <div className="text-center space-y-3 max-w-2xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>Architectural Decision Records</span>
        </div>
        <h1 className="text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          Architecture Decisions (ADRs)
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          Historical record of architectural choices, trade-offs, and invariants in CloudWeave.
        </p>
      </div>

      {/* ADRs List */}
      <div className="space-y-6">
        {adrs.map((adr) => (
          <article
            key={adr.id}
            className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4"
          >
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-[#26262B] pb-3">
              <div className="flex items-center gap-2 font-mono text-xs">
                <span className="font-bold text-[#EDE8DF] px-2 py-0.5 rounded bg-[#000000] border border-[#26262B]">
                  {adr.id}
                </span>
                <span className="text-[#8B8B8F]">{adr.date}</span>
              </div>
              <span className="px-2.5 py-0.5 rounded text-[10px] font-mono font-bold bg-[#000000] text-[#EDE8DF] border border-[#26262B] self-start sm:self-auto">
                {adr.status}
              </span>
            </div>

            <h2 className="text-lg font-serif font-bold text-[#EDE8DF]">
              {adr.title}
            </h2>

            <div className="space-y-3 font-mono text-xs">
              <div>
                <b className="text-[#A6A9AE] block uppercase text-[10px]">Context & Problem Statement:</b>
                <p className="text-[#8B8B8F] font-sans text-xs mt-0.5">{adr.context}</p>
              </div>

              <div>
                <b className="text-[#A6A9AE] block uppercase text-[10px]">Decision:</b>
                <p className="text-[#EDE8DF] font-sans text-xs mt-0.5 font-medium">{adr.decision}</p>
              </div>

              <div>
                <b className="text-[#8B8B8F] block uppercase text-[10px]">Consequences:</b>
                <p className="text-[#8B8B8F] font-sans text-xs mt-0.5">{adr.consequences}</p>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  )
}
