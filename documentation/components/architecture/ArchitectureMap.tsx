'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import {
  Layers,
  Server,
  ShieldCheck,
  Activity,
  Zap,
  Terminal,
  FileCode,
  HardDrive,
  ArrowDown,
  Info,
  CheckCircle,
} from 'lucide-react'
import { CodeBlock } from '../ui/CodeBlock'

interface ArchComponent {
  id: string
  name: string
  category: string
  purpose: string
  inputs: string
  outputs: string
  goPackage: string
  keyTypes: string[]
  icon: any
}

const archComponents: ArchComponent[] = [
  {
    id: 'clients',
    name: 'Client Interfaces & SDKs',
    category: 'Ingress Layer',
    purpose: 'Provides S3 REST client compatibility for AWS CLI, boto3, Go SDK, and the cweave CLI tool.',
    inputs: 'Raw file streams, HTTP S3 requests (PUT, GET, HEAD, DELETE)',
    outputs: 'Authenticated HTTP/1.1 and HTTP/2 stream sockets',
    goPackage: 'client/ & cmd/cweave/',
    keyTypes: ['client.Client', 'cweave.Config'],
    icon: Terminal,
  },
  {
    id: 's3-api',
    name: 'S3-Compatible API Router',
    category: 'API Layer',
    purpose: 'Translates Amazon S3 REST calls into CloudWeave internal coordinator calls with AWS SigV4 / bearer auth.',
    inputs: 'HTTP PUT /bucket/key, GET /bucket/key',
    outputs: 'Object payload stream & manifest query requests',
    goPackage: 'internal/api/',
    keyTypes: ['api.Server', 'api.Handler', 'api.AuthMiddleware'],
    icon: FileCode,
  },
  {
    id: 'chunker',
    name: 'FastCDC Content-Defined Chunker',
    category: 'Chunking & CAS',
    purpose: 'Splits incoming byte streams into variable content-defined chunks using rolling Gear hash and computes SHA-256 CAS addresses.',
    inputs: 'Raw io.Reader stream (1MB - multi-GB)',
    outputs: '[]chunk.Chunk (ID = sha256(data), Size, Payload)',
    goPackage: 'internal/chunk/',
    keyTypes: ['chunk.Chunk', 'chunk.FastCDCChunker', 'chunk.GearTable'],
    icon: Layers,
  },
  {
    id: 'ring',
    name: 'Consistent Hash Ring (150 VNodes)',
    category: 'Placement & Topology',
    purpose: 'Maps SHA-256 chunk IDs to top N storage nodes in the 32-bit ring space, providing uniform load distribution.',
    inputs: 'Chunk ID hash string (hex)',
    outputs: 'Ordered list of N target physical node addresses',
    goPackage: 'internal/ring/',
    keyTypes: ['ring.HashRing', 'ring.Node', 'ring.VNode'],
    icon: Server,
  },
  {
    id: 'quorum',
    name: 'N / W / R Quorum Coordinator',
    category: 'Consensus & Fan-Out',
    purpose: 'Dispatches chunks in parallel to N nodes; waits for W write acknowledgments or queries R nodes on read.',
    inputs: '[]chunk.Chunk, target nodes list, W threshold',
    outputs: 'Quorum ACK status, write confirmation, latency metrics',
    goPackage: 'internal/coordinator/',
    keyTypes: ['coordinator.WriteSession', 'coordinator.ReadSession'],
    icon: ShieldCheck,
  },
  {
    id: 'storage-nodes',
    name: 'Storage Node & DiskStore Engine',
    category: 'Data Plane',
    purpose: 'Persists chunks to local content-addressed disk files ($DATA_DIR/chunks/xx/xx) with in-memory 64MB LRU cache.',
    inputs: 'Chunk ID and binary byte slice',
    outputs: 'fsync() acknowledgment or binary chunk stream',
    goPackage: 'internal/storage/',
    keyTypes: ['storage.DiskStore', 'storage.LRUCache', 'storage.ChunkStore'],
    icon: HardDrive,
  },
  {
    id: 'heartbeat',
    name: 'Heartbeat Failure Detection',
    category: 'Cluster Health',
    purpose: 'Sends periodic 500ms heartbeat pings between mesh nodes. Marks dead nodes after 3-second consecutive timeout.',
    inputs: 'Gossip ping /pong roundtrips',
    outputs: 'Node status transitions: Healthy -> Suspicious -> Dead',
    goPackage: 'internal/cluster/',
    keyTypes: ['cluster.Membership', 'cluster.HeartbeatLoop'],
    icon: Activity,
  },
  {
    id: 'repair',
    name: 'Self-Healing Replica Repair Worker',
    category: 'Anti-Entropy & Recovery',
    purpose: 'Scans manifests on node failure, computes under-replicated chunks, and copies replicas to healthy nodes to restore N=3.',
    inputs: 'Node failure event, metadata manifest table',
    outputs: 'Restored chunk replicas on replacement nodes',
    goPackage: 'internal/replication/',
    keyTypes: ['replication.RepairWorker', 'replication.Reconciler'],
    icon: Zap,
  },
]

export function ArchitectureMap() {
  const [selected, setSelected] = useState<ArchComponent>(archComponents[2]) // default FastCDC

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs">
        <div className="flex items-center gap-2">
          <span className="rounded-md bg-[#CAF0F8] px-2.5 py-1 text-[10px] font-mono font-bold text-[#0077B6] uppercase">
            System Architecture
          </span>
          <span className="text-xs font-mono text-[#64748B]">Click any component to inspect</span>
        </div>
        <h2 className="text-2xl font-bold tracking-tight text-[#03045E] mt-2">
          One Request, Several Deliberate Boundaries.
        </h2>
        <p className="text-xs sm:text-sm text-[#64748B] mt-1 max-w-2xl">
          CloudWeave separates S3 client semantics from content-addressed chunking, quorum coordination, and local disk storage.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Interactive Vertical Architecture Diagram */}
        <div className="lg:col-span-7 space-y-3 font-mono text-xs">
          {archComponents.map((comp, idx) => {
            const isSelected = selected.id === comp.id
            const Icon = comp.icon

            return (
              <React.Fragment key={comp.id}>
                <div
                  onClick={() => setSelected(comp)}
                  className={`p-4 rounded-xl border cursor-pointer transition-all ${
                    isSelected
                      ? 'border-[#00B4D8] bg-[#CAF0F8] shadow-md ring-2 ring-[#00B4D8]/30'
                      : 'border-[#CBD5E1] bg-white hover:border-[#90E0EF] hover:bg-[#F8FAFC]'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div
                        className={`flex h-8 w-8 items-center justify-center rounded-lg ${
                          isSelected ? 'bg-[#0077B6] text-white' : 'bg-[#F1F5F9] text-[#0077B6]'
                        }`}
                      >
                        <Icon className="w-4 h-4" />
                      </div>
                      <div>
                        <span className="font-bold text-sm text-[#03045E]">{comp.name}</span>
                        <p className="text-[10px] text-[#64748B]">{comp.category}</p>
                      </div>
                    </div>

                    <span className="text-[11px] font-bold text-[#0077B6] bg-white px-2 py-0.5 rounded border border-[#CBD5E1]">
                      {comp.goPackage}
                    </span>
                  </div>
                </div>

                {idx < archComponents.length - 1 && (
                  <div className="flex justify-center">
                    <div className="h-4 w-0.5 bg-[#CBD5E1]" />
                  </div>
                )}
              </React.Fragment>
            )
          })}
        </div>

        {/* Right: Component Inspector Side Drawer */}
        <div className="lg:col-span-5">
          <div className="sticky top-20 p-5 rounded-2xl border border-[#CBD5E1] bg-white shadow-sm space-y-4 font-mono text-xs">
            <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-3">
              <span className="text-[10px] font-bold uppercase text-[#0077B6]">
                Component Inspector
              </span>
              <span className="text-[11px] font-bold text-[#22A06B] bg-[#E8F8F0] px-2 py-0.5 rounded border border-[#A8E6CF]">
                {selected.category}
              </span>
            </div>

            <div className="space-y-3">
              <div>
                <h3 className="font-bold text-base text-[#03045E]">{selected.name}</h3>
                <p className="text-xs text-[#475569] leading-relaxed mt-1 font-sans">
                  {selected.purpose}
                </p>
              </div>

              <div className="p-3 rounded-lg bg-[#F8FAFC] border border-[#E2E8F0] space-y-2">
                <div>
                  <span className="text-[10px] text-[#64748B] block uppercase font-bold">Input Stream:</span>
                  <span className="text-[#03045E] font-semibold text-[11px]">{selected.inputs}</span>
                </div>
                <div>
                  <span className="text-[10px] text-[#64748B] block uppercase font-bold">Output Artifacts:</span>
                  <span className="text-[#0077B6] font-semibold text-[11px]">{selected.outputs}</span>
                </div>
              </div>

              <div>
                <span className="text-[10px] text-[#64748B] block uppercase font-bold">Go Package Path:</span>
                <div className="mt-1 p-2 rounded bg-[#F1F5F9] text-[#03045E] font-bold border border-[#E2E8F0]">
                  cloudWeave/{selected.goPackage}
                </div>
              </div>

              <div>
                <span className="text-[10px] text-[#64748B] block uppercase font-bold">Core Structs & Interfaces:</span>
                <div className="flex flex-wrap gap-1.5 mt-1.5">
                  {selected.keyTypes.map((t) => (
                    <span
                      key={t}
                      className="px-2 py-0.5 rounded bg-[#CAF0F8] text-[#0077B6] font-bold text-[10px] border border-[#90E0EF]"
                    >
                      {t}
                    </span>
                  ))}
                </div>
              </div>
            </div>

            <div className="pt-3 border-t border-[#E2E8F0] text-[10px] text-[#64748B] flex items-center gap-1.5">
              <CheckCircle className="w-3.5 h-3.5 text-[#22A06B]" />
              <span>Zero external CGO dependencies</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
