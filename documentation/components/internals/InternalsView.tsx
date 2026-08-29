'use client'

import React from 'react'
import { CodeBlock } from '../ui/CodeBlock'

export function InternalsView() {
  return (
    <div className="w-full max-w-4xl mx-auto space-y-10 py-4 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Header */}
      <div className="text-center space-y-3 max-w-2xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>Engineering Internals</span>
        </div>
        <h1 className="text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          Go Engine & Memory Model
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          Memory layout, concurrency primitives, zero-allocation buffer pools, and test assertions.
        </p>
      </div>

      {/* 1. Concurrency & Memory Model */}
      <section className="p-8 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4">
        <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">
          Zero-Allocation Buffer Pools (<code className="font-mono text-base text-[#EDE8DF]">sync.Pool</code>)
        </h2>
        <p className="text-xs font-mono text-[#8B8B8F] leading-relaxed">
          To sustain high-throughput multi-MB file uploads without triggering Go runtime GC pauses, CloudWeave recycles byte buffers using a global <code className="text-[#EDE8DF] bg-[#000000] px-1.5 py-0.5 rounded border border-[#26262B]">sync.Pool</code>. Memory consumption is capped at ~240 MB under 50 streams.
        </p>

        <CodeBlock
          code={`// internal/chunk/pool.go
var bufferPool = sync.Pool{
    New: func() interface{} {
        b := make([]byte, MaxChunkSize)
        return &b
    },
}

func AcquireBuffer() *[]byte {
    return bufferPool.Get().(*[]byte)
}

func ReleaseBuffer(b *[]byte) {
    bufferPool.Put(b)
}`}
          language="go"
          filename="pool.go"
        />
      </section>

      {/* 2. Directory Structure */}
      <section className="p-8 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4">
        <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">Repository Package Layout</h2>
        <p className="text-xs font-mono text-[#8B8B8F]">
          The codebase strictly separates algorithmic data structures from networking and disk I/O.
        </p>

        <div className="space-y-3 pt-2 font-mono text-xs">
          <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B]">
            <b className="text-[#EDE8DF]">cmd/node/main.go</b>
            <p className="text-[#8B8B8F] font-sans text-xs mt-1">
              Storage daemon process, wires storage engine, ring registry, and mTLS transport.
            </p>
          </div>
          <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B]">
            <b className="text-[#EDE8DF]">cmd/cweave/main.go</b>
            <p className="text-[#8B8B8F] font-sans text-xs mt-1">
              Developer CLI interface for object upload, download, deduplication status, and cluster diagnostics.
            </p>
          </div>
          <div className="p-4 rounded-lg bg-[#000000] border border-[#26262B]">
            <b className="text-[#EDE8DF]">internal/chunk/cdc.go</b>
            <p className="text-[#8B8B8F] font-sans text-xs mt-1">
              FastCDC rolling Gear hash matrix and dynamic boundary chunk generator.
            </p>
          </div>
        </div>
      </section>
    </div>
  )
}
