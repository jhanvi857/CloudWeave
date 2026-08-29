'use client'

import React from 'react'
import { CodeBlock } from '../ui/CodeBlock'

export function ArchitectureView() {
  return (
    <div className="w-full max-w-4xl mx-auto space-y-12 py-4 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Title */}
      <div className="text-center space-y-3 max-w-2xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>System Architecture</span>
        </div>
        <h1 className="text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          How CloudWeave is Built
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          A deep dive into the subsystems, data flows, and invariants that power CloudWeave.
        </p>
      </div>

      {/* Layer Diagram */}
      <section className="p-8 rounded-xl border border-[#26262B] bg-[#19191C] space-y-6">
        <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">5-Tier Layered Architecture</h2>

        <div className="space-y-4 font-mono text-xs">
          {/* Layer 1: Ingress */}
          <div className="p-5 rounded-lg border border-[#26262B] bg-[#000000] space-y-2">
            <div className="flex items-center justify-between text-[#EDE8DF] font-bold">
              <span className="text-sm">1. Ingress & S3 Compatibility Layer</span>
              <span className="text-xs text-[#A6A9AE]">internal/api</span>
            </div>
            <p className="text-[#8B8B8F] font-sans text-xs leading-relaxed">
              HTTP server exposing standard S3 REST protocol. Validates AWS SigV4 signatures, parses HTTP multipart / streaming PUT/GET bodies, and handles bucket routing.
            </p>
          </div>

          {/* Layer 2: Coordinator & FastCDC */}
          <div className="p-5 rounded-lg border border-[#26262B] bg-[#000000] space-y-2">
            <div className="flex items-center justify-between text-[#EDE8DF] font-bold">
              <span className="text-sm">2. Coordinator & Chunking Engine</span>
              <span className="text-xs text-[#A6A9AE]">internal/coordinator · internal/chunk</span>
            </div>
            <p className="text-[#8B8B8F] font-sans text-xs leading-relaxed">
              Runs FastCDC rolling Gear hash to slice objects into 16KB-256KB CAS blocks. Computes SHA-256 digests and initiates parallel write fan-out.
            </p>
          </div>

          {/* Layer 3: Consistent Hash Ring */}
          <div className="p-5 rounded-lg border border-[#26262B] bg-[#000000] space-y-2">
            <div className="flex items-center justify-between text-[#EDE8DF] font-bold">
              <span className="text-sm">3. 150-Virtual-Node Consistent Hash Ring</span>
              <span className="text-xs text-[#A6A9AE]">internal/ring</span>
            </div>
            <p className="text-[#8B8B8F] font-sans text-xs leading-relaxed">
              Distributes 150 vnodes per physical machine onto a 32-bit integer ring. Binary search maps each SHA-256 chunk key to primary and N=3 replica nodes with minimal key migration.
            </p>
          </div>

          {/* Layer 4: Storage & CAS */}
          <div className="p-5 rounded-lg border border-[#26262B] bg-[#000000] space-y-2">
            <div className="flex items-center justify-between text-[#EDE8DF] font-bold">
              <span className="text-sm">4. Content-Addressable Storage (CAS)</span>
              <span className="text-xs text-[#A6A9AE]">internal/storage</span>
            </div>
            <p className="text-[#8B8B8F] font-sans text-xs leading-relaxed">
              Disk store persisting chunks under hex-prefixed directory hierarchies. Implements atomic writes, SHA-256 integrity verification, and in-memory LRU read caching.
            </p>
          </div>

          {/* Layer 5: Durability & WAL */}
          <div className="p-5 rounded-lg border border-[#26262B] bg-[#000000] space-y-2">
            <div className="flex items-center justify-between text-[#EDE8DF] font-bold">
              <span className="text-sm">5. Write-Ahead Log (WAL) & Vector Clocks</span>
              <span className="text-xs text-[#A6A9AE]">internal/metadata</span>
            </div>
            <p className="text-[#8B8B8F] font-sans text-xs leading-relaxed">
              Maintains sequential append-only WAL with CRC32 checksums for manifest durability across node restarts. Tracks causal version ordering via vector clocks.
            </p>
          </div>
        </div>
      </section>
    </div>
  )
}
