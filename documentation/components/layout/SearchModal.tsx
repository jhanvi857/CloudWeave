'use client'

import React, { useState, useEffect } from 'react'
import { Search, X, Layers, Cpu, Server, Activity, ShieldCheck, Terminal, BookOpen, ArrowRight } from 'lucide-react'

interface SearchItem {
  id: string
  title: string
  category: string
  description: string
  icon: any
}

const searchItems: SearchItem[] = [
  { id: 'playground-overview', title: 'Interactive Playground', category: 'Playground', description: 'Follow an object from S3 request to distributed storage', icon: Activity },
  { id: 'playground-upload', title: 'Upload & Chunking Simulation', category: 'Playground', description: 'Watch video.mp4 stream into FastCDC variable chunks', icon: Layers },
  { id: 'playground-ring', title: 'Consistent Hash Ring Visualizer', category: 'Playground', description: 'Explore 150 virtual nodes and deterministic key placement', icon: Server },
  { id: 'playground-quorum', title: 'Quorum Calculator (N/W/R)', category: 'Playground', description: 'Tune W and R values and observe acknowledgment consensus', icon: ShieldCheck },
  { id: 'playground-failure', title: 'Failure & Self-Healing Demo', category: 'Playground', description: 'Kill a storage node and watch automatic replica repair', icon: Activity },
  { id: 'playground-dedup', title: 'FastCDC Deduplication Playground', category: 'Playground', description: 'Test rolling hash boundary shifts and physical storage savings', icon: Layers },
  { id: 'playground-ec', title: 'Reed-Solomon Erasure Coding (K=4, M=2)', category: 'Playground', description: 'Fail 2 nodes and reconstruct original object from 4 surviving shards', icon: Cpu },
  { id: 'concept-fastcdc', title: 'FastCDC Content-Defined Chunking', category: 'Concepts', description: 'Gear rolling hash, min 16KB, avg 64KB, max 256KB chunk boundaries', icon: BookOpen },
  { id: 'concept-cas', title: 'Content-Addressable Storage (CAS)', category: 'Concepts', description: 'SHA-256 integrity, tamper-evidence, and natural deduplication', icon: BookOpen },
  { id: 'concept-consistent-hashing', title: 'Consistent Hashing & 150 VNodes', category: 'Concepts', description: 'Minimal key migration on node join/leave, uniform load distribution', icon: BookOpen },
  { id: 'concept-quorum', title: 'N / W / R Quorum Consensus', category: 'Concepts', description: 'Dynamo-style tunable durability vs availability tradeoffs', icon: BookOpen },
  { id: 'concept-replication', title: 'Replication vs Erasure Coding', category: 'Concepts', description: '3x replication storage cost vs 1.5x Reed-Solomon storage efficiency', icon: BookOpen },
  { id: 'concept-repair', title: 'Heartbeat Failure Detection & Repair', category: 'Concepts', description: '3-second timeout, anti-entropy sync, and worker pool healing', icon: BookOpen },
  { id: 'concept-wal', title: 'Write-Ahead Log (WAL) & Durability', category: 'Concepts', description: 'Synchronous fsync disk durability before client acknowledgment', icon: BookOpen },
  { id: 'concept-vc', title: 'Vector Clocks & Object Versioning', category: 'Concepts', description: 'Causal history tracking, concurrent branch detection, and ETag', icon: BookOpen },
  { id: 'arch-overview', title: 'System Architecture Diagram', category: 'Architecture', description: 'Interactive end-to-end component flow with inspector drawer', icon: Layers },
  { id: 'internals-s3', title: 'S3-Compatible HTTP API', category: 'Internals', description: 'PUT, GET, DELETE, HEAD, ListObjectsV2 compatibility', icon: Terminal },
  { id: 'internals-transport', title: 'Transport Layer & Connection Pooling', category: 'Internals', description: 'Persistent HTTP keep-alives and mutual TLS (mTLS)', icon: Server },
  { id: 'benchmarks-empirical', title: 'Empirical Benchmark Suite', category: 'Benchmarks', description: '152 MB/s upload, 121 MB/s download, 485 MB/s deduplication', icon: Activity },
  { id: 'decisions-adr', title: 'Architecture Decision Records (ADRs)', category: 'Decisions', description: 'Rationale behind Go, FastCDC, 150 vnodes, WAL, and in-memory metadata', icon: BookOpen },
  { id: 'quickstart-docker', title: 'Quick Start with Docker & Compose', category: 'Quickstart', description: 'Spin up 5-node distributed cluster locally in seconds', icon: Terminal },
  { id: 'reference-cli', title: 'cweave CLI & Go SDK Reference', category: 'Reference', description: 'Full command-line manual and native Go client library', icon: Terminal },
]

interface SearchModalProps {
  isOpen: boolean
  onClose: () => void
  onSelect: (id: string) => void
}

export function SearchModal({ isOpen, onClose, onSelect }: SearchModalProps) {
  const [query, setQuery] = useState('')

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        if (isOpen) onClose()
        else {
          setQuery('')
          // toggle handled by parent
        }
      }
      if (e.key === 'Escape' && isOpen) {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  const filtered = searchItems.filter(
    (item) =>
      item.title.toLowerCase().includes(query.toLowerCase()) ||
      item.description.toLowerCase().includes(query.toLowerCase()) ||
      item.category.toLowerCase().includes(query.toLowerCase())
  )

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 p-4 bg-[#03045E]/20 backdrop-blur-xs">
      <div
        className="w-full max-w-xl rounded-2xl border border-[#CBD5E1] bg-white shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-3 px-4 py-3.5 border-b border-[#E2E8F0] bg-[#F8FAFC]">
          <Search className="w-5 h-5 text-[#0077B6]" />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search concepts, playground simulations, benchmarks..."
            className="w-full bg-transparent text-sm text-[#03045E] placeholder-[#94A3B8] outline-hidden font-medium"
            autoFocus
          />
          <button
            onClick={onClose}
            className="rounded p-1 text-[#64748B] hover:bg-[#E2E8F0] hover:text-[#0077B6] cursor-pointer"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="max-h-96 overflow-y-auto p-2 divide-y divide-[#F1F5F9]">
          {filtered.length === 0 ? (
            <div className="py-12 text-center text-sm text-[#64748B]">
              No results found for &ldquo;<span className="text-[#0077B6] font-medium">{query}</span>&rdquo;
            </div>
          ) : (
            filtered.map((item) => {
              const Icon = item.icon
              return (
                <button
                  key={item.id}
                  onClick={() => {
                    onSelect(item.id)
                    onClose()
                  }}
                  className="w-full flex items-start gap-3.5 p-3 rounded-xl text-left hover:bg-[#CAF0F8]/50 transition-colors group cursor-pointer"
                >
                  <div className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-[#0077B6]/20 bg-[#CAF0F8] text-[#0077B6] group-hover:bg-[#0077B6] group-hover:text-white transition-colors">
                    <Icon className="w-4 h-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-sm text-[#03045E]">{item.title}</span>
                      <span className="text-[10px] font-mono font-medium px-1.5 py-0.5 rounded bg-[#F1F5F9] text-[#64748B]">
                        {item.category}
                      </span>
                    </div>
                    <p className="text-xs text-[#64748B] line-clamp-1 mt-0.5">{item.description}</p>
                  </div>
                  <ArrowRight className="w-4 h-4 text-[#94A3B8] group-hover:text-[#0077B6] self-center shrink-0 transition-colors" />
                </button>
              )
            })
          )}
        </div>

        <div className="flex items-center justify-between px-4 py-2.5 bg-[#F8FAFC] border-t border-[#E2E8F0] text-[11px] font-mono text-[#64748B]">
          <div className="flex items-center gap-3">
            <span>
              <kbd className="px-1.5 py-0.5 rounded border border-[#CBD5E1] bg-white text-[10px]">↑</kbd>{' '}
              <kbd className="px-1.5 py-0.5 rounded border border-[#CBD5E1] bg-white text-[10px]">↓</kbd> Navigate
            </span>
            <span>
              <kbd className="px-1.5 py-0.5 rounded border border-[#CBD5E1] bg-white text-[10px]">ESC</kbd> Close
            </span>
          </div>
          <span className="text-[#0077B6] font-bold">CloudWeave Engine</span>
        </div>
      </div>
    </div>
  )
}
