'use client'

import React, { useState } from 'react'
import { BookOpen, CheckCircle, HelpCircle } from 'lucide-react'

interface ADR {
  id: string
  title: string
  context: string
  decision: string
  reasoning: string[]
  tradeoffs: string[]
}

const adrList: ADR[] = [
  {
    id: 'adr-go',
    title: '1. Programming Language: Go (Golang 1.22+)',
    context: 'Distributed storage nodes require high I/O throughput, low-latency network primitives, predictable memory management, and cross-platform compilation.',
    decision: 'Built CloudWeave entirely in Go with zero external C/C++ dependencies or CGO bindings.',
    reasoning: [
      'Native Concurrency: Lightweight goroutines and channels provide a natural model for parallel chunk fan-out, background heartbeats, and worker pools.',
      'Standard Library: Production-grade HTTP/2, TLS, and crypto primitives (SHA-256) without external packages.',
      'Single Binary Distribution: Cross-compiles to a single static binary containing the storage engine, transport server, and embedded web dashboard.',
    ],
    tradeoffs: [
      'Go GC pauses (mitigated via sync.Pool constant-buffer reuse to keep allocations near zero in hot streaming path).',
    ],
  },
  {
    id: 'adr-fastcdc',
    title: '2. FastCDC Rolling Hash for Content-Defined Chunking',
    context: 'Fixed-size chunking (e.g. 1MB boundaries) suffers from the byte-shift problem: inserting a single byte destroys 100% of subsequent chunk deduplication.',
    decision: 'Implemented FastCDC / Gear rolling hash content-defined chunking with target 64KB average chunk size (16KB min, 256KB max).',
    reasoning: [
      'Boundary Resilience: Modifying a byte only alters the immediate chunk boundary, preserving identical SHA-256 hashes for all other chunks.',
      'High Throughput: Gear hash uses a 256-entry lookup table with single-cycle bit shifts, operating much faster than Rabin fingerprints.',
    ],
    tradeoffs: [
      'Variable chunk sizes require dynamic buffer management rather than static array slices.',
    ],
  },
  {
    id: 'adr-vnodes',
    title: '3. Consistent Hashing Ring with 150 Virtual Nodes',
    context: 'In distributed storage, chunks must be placed deterministically. Modulo hashing (key % N) causes ~100% data migration on node join/leave.',
    decision: 'Implemented a consistent hash ring with 150 virtual nodes per physical node mapped onto a 32-bit SHA-256 integer space.',
    reasoning: [
      'Minimal Remapping: Adding/removing 1 node only migrates 1/N fraction of keys.',
      'Uniform Balance: 150 virtual nodes reduce load distribution variance across servers to < 5%.',
    ],
    tradeoffs: [
      'Requires storing 150 * N virtual node tokens in memory and running binary search (O(log(150N)) < 1 microsecond).',
    ],
  },
  {
    id: 'adr-quorum',
    title: '4. Dynamo-Style Tunable N / W / R Quorum Consensus',
    context: 'Single-leader consensus (e.g. Raft) routes all data traffic through a single leader node, creating a severe bottleneck for multi-gigabyte object streaming.',
    decision: 'Decentralized Dynamo-style quorum consensus where any node acts as coordinator for incoming client writes and reads.',
    reasoning: [
      'Linear Scalability: Data throughput scales horizontally across all cluster nodes.',
      'Tunable Durability: Setting W=2 on N=3 guarantees strong durability while tolerating 1 node failure during writes.',
    ],
    tradeoffs: [
      'Requires anti-entropy repair and vector clocks to resolve concurrent write branches.',
    ],
  },
  {
    id: 'adr-wal',
    title: '5. Append-Only Write-Ahead Log (WAL) with Synchronous Fsync',
    context: 'Node crashes or power outages must not cause committed object manifests to be lost or corrupted.',
    decision: 'All manifest commits append JSON entries to metadata.wal with synchronous f.Sync() before returning HTTP 201 Created.',
    reasoning: [
      'Strict Durability Invariant: A write only succeeds after W chunk acknowledgments are received AND the local WAL is flushed to disk.',
      'Instant Crash Recovery: Replaying metadata.wal restores full manifest state into RAM on startup.',
    ],
    tradeoffs: [
      'Synchronous fsync adds ~1-3ms disk latency per manifest commit (amortized over multi-MB chunk uploads).',
    ],
  },
]

export function DecisionsView() {
  const [selectedADR, setSelectedADR] = useState<ADR>(adrList[0])

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs">
        <div className="flex items-center gap-2">
          <span className="rounded-md bg-[#CAF0F8] px-2.5 py-1 text-[10px] font-mono font-bold text-[#0077B6] uppercase">
            Architecture Decisions
          </span>
          <span className="text-xs font-mono text-[#64748B]">ADR Records & Trade-offs</span>
        </div>
        <h2 className="text-2xl font-bold tracking-tight text-[#03045E] mt-2">
          Why CloudWeave is Built This Way
        </h2>
        <p className="text-xs sm:text-sm text-[#64748B] mt-1 max-w-2xl">
          Detailed technical rationale, architectural constraints, and engineering trade-offs behind each core design decision.
        </p>

        {/* ADR Selector */}
        <div className="flex flex-wrap gap-2 mt-4 pt-3 border-t border-[#E2E8F0]">
          {adrList.map((adr) => {
            const isSelected = selectedADR.id === adr.id
            return (
              <button
                key={adr.id}
                onClick={() => setSelectedADR(adr)}
                className={`px-3 py-1.5 rounded-lg text-xs font-mono font-semibold transition-all ${
                  isSelected
                    ? 'bg-[#0077B6] text-white shadow-2xs'
                    : 'bg-[#F8FAFC] text-[#475569] border border-[#CBD5E1] hover:bg-[#CAF0F8] hover:text-[#03045E]'
                }`}
              >
                {adr.title.split(':')[0]}
              </button>
            )
          })}
        </div>
      </div>

      {/* ADR Content Card */}
      <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs space-y-6 font-mono text-xs">
        <div>
          <h3 className="text-lg font-bold text-[#03045E]">{selectedADR.title}</h3>
        </div>

        <div className="space-y-4">
          <div className="p-4 rounded-xl bg-[#F8FAFC] border border-[#E2E8F0] space-y-1.5">
            <span className="text-[10px] font-bold text-[#64748B] uppercase block">Context & Problem</span>
            <p className="text-xs text-[#475569] font-sans leading-relaxed">{selectedADR.context}</p>
          </div>

          <div className="p-4 rounded-xl bg-[#CAF0F8]/50 border border-[#90E0EF] space-y-1.5">
            <span className="text-[10px] font-bold text-[#0077B6] uppercase block">Architectural Decision</span>
            <p className="text-xs text-[#03045E] font-sans font-medium leading-relaxed">{selectedADR.decision}</p>
          </div>

          <div className="p-4 rounded-xl bg-[#F8FAFC] border border-[#E2E8F0] space-y-2">
            <span className="text-[10px] font-bold text-[#22A06B] uppercase block">Technical Reasoning</span>
            <ul className="space-y-1.5 text-xs text-[#475569] font-sans">
              {selectedADR.reasoning.map((r, i) => (
                <li key={i} className="flex items-start gap-2">
                  <CheckCircle className="w-3.5 h-3.5 text-[#22A06B] shrink-0 mt-0.5" />
                  <span>{r}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="p-4 rounded-xl bg-[#FFF8E6] border border-[#FDE68A] space-y-2">
            <span className="text-[10px] font-bold text-[#D97706] uppercase block">Accepted Trade-offs</span>
            <ul className="space-y-1.5 text-xs text-[#475569] font-sans">
              {selectedADR.tradeoffs.map((t, i) => (
                <li key={i} className="flex items-start gap-2">
                  <span className="text-[#D97706] font-bold">•</span>
                  <span>{t}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}
