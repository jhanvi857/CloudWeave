'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { sampleChunks, ChunkInfo } from './UploadChunkingVisualizer'
import { Server, Plus, Minus, Hash, Info, Layers } from 'lucide-react'

interface NodePos {
  id: string
  name: string
  angleDeg: number
  ip: string
  active: boolean
}

const initialNodes: NodePos[] = [
  { id: 'node-1', name: 'Node 1', angleDeg: 0, ip: '10.0.0.1:9000', active: true },
  { id: 'node-2', name: 'Node 2', angleDeg: 72, ip: '10.0.0.2:9001', active: true },
  { id: 'node-3', name: 'Node 3', angleDeg: 144, ip: '10.0.0.3:9002', active: true },
  { id: 'node-4', name: 'Node 4', angleDeg: 216, ip: '10.0.0.4:9003', active: true },
  { id: 'node-5', name: 'Node 5', angleDeg: 288, ip: '10.0.0.5:9004', active: true },
]

const radius = 140
const centerX = 200
const centerY = 200

// Generate 150 virtual node tick positions around 360 degrees
const virtualNodes = Array.from({ length: 150 }).map((_, i) => {
  const angleDeg = (i * 360) / 150
  const rad = ((angleDeg - 90) * Math.PI) / 180
  const x = Math.round((centerX + radius * Math.cos(rad)) * 100) / 100
  const y = Math.round((centerY + radius * Math.sin(rad)) * 100) / 100
  return {
    id: i,
    x,
    y,
    nodeOwner: `Node ${(i % 5) + 1}`,
  }
})

interface ConsistentHashRingProps {
  selectedChunkId?: string
  onSelectChunk?: (id: string) => void
}

export function ConsistentHashRing({
  selectedChunkId = 'chunk-a',
  onSelectChunk,
}: ConsistentHashRingProps) {
  const [nodes, setNodes] = useState<NodePos[]>(initialNodes)
  const [currentChunkId, setCurrentChunkId] = useState(selectedChunkId)

  const activeChunk = sampleChunks.find((c) => c.id === currentChunkId) || sampleChunks[0]

  // Map chunk hashes to ring angle (deterministic)
  const chunkAngleMap: Record<string, number> = {
    'chunk-a': 54, // Maps to Node 2 (angle 72)
    'chunk-b': 195, // Maps to Node 4 (angle 216)
    'chunk-c': 345, // Maps to Node 1 (angle 0 / 360)
    'chunk-d': 120, // Maps to Node 3 (angle 144)
  }

  const chunkAngle = chunkAngleMap[activeChunk.id] ?? 45

  const toggleNode = (id: string) => {
    setNodes((prev) =>
      prev.map((n) => (n.id === id ? { ...n, active: !n.active } : n))
    )
  }

  const radius = 140
  const centerX = 200
  const centerY = 200

  return (
    <div className="flex flex-col lg:flex-row items-center justify-between gap-6 p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs">
      {/* 1. Left Ring SVG Canvas */}
      <div className="relative flex items-center justify-center w-full max-w-[400px] aspect-square">
        <svg viewBox="0 0 400 400" className="w-full h-full">
          {/* Subtle Background Ring Circles */}
          <circle
            cx={centerX}
            cy={centerY}
            r={radius}
            fill="none"
            stroke="#E2E8F0"
            strokeWidth="3"
            strokeDasharray="4 4"
          />
          <circle
            cx={centerX}
            cy={centerY}
            r={radius + 18}
            fill="none"
            stroke="#F1F5F9"
            strokeWidth="1"
          />

          {/* 150 Virtual Node Dots */}
          {virtualNodes.map((vn) => (
            <circle
              key={vn.id}
              cx={vn.x}
              cy={vn.y}
              r="1.5"
              fill="#90E0EF"
              opacity="0.6"
            />
          ))}

          {/* Active Chunk Placement Line & Marker */}
          {(() => {
            const rad = ((chunkAngle - 90) * Math.PI) / 180
            const x = centerX + radius * Math.cos(rad)
            const y = centerY + radius * Math.sin(rad)
            return (
              <g>
                <motion.line
                  x1={centerX}
                  y1={centerY}
                  x2={x}
                  y2={y}
                  stroke="#00B4D8"
                  strokeWidth="2"
                  strokeDasharray="3 3"
                  initial={{ pathLength: 0 }}
                  animate={{ pathLength: 1 }}
                  transition={{ duration: 0.5 }}
                />
                <circle cx={x} cy={y} r="7" fill="#00B4D8" />
                <circle cx={x} cy={y} r="11" fill="none" stroke="#00B4D8" strokeWidth="1.5" className="animate-ping" />
              </g>
            )
          })()}

          {/* 5 Physical Storage Nodes */}
          {nodes.map((node) => {
            const rad = ((node.angleDeg - 90) * Math.PI) / 180
            const x = centerX + radius * Math.cos(rad)
            const y = centerY + radius * Math.sin(rad)

            const isPrimary = activeChunk.primaryNode === node.name
            const isReplica = activeChunk.replicas.includes(node.name)

            return (
              <g key={node.id} className="cursor-pointer" onClick={() => toggleNode(node.id)}>
                {/* Node Outer Ring */}
                <circle
                  cx={x}
                  cy={y}
                  r="24"
                  fill={isPrimary ? '#0077B6' : isReplica ? '#CAF0F8' : node.active ? '#FFFFFF' : '#FEECEB'}
                  stroke={isPrimary ? '#03045E' : isReplica ? '#00B4D8' : node.active ? '#CBD5E1' : '#E5484D'}
                  strokeWidth={isPrimary || isReplica ? '2.5' : '1.5'}
                />

                {/* Node Label Text */}
                <text
                  x={x}
                  y={y + 4}
                  textAnchor="middle"
                  fontSize="10"
                  fontFamily="JetBrains Mono, monospace"
                  fontWeight="bold"
                  fill={isPrimary ? '#FFFFFF' : node.active ? '#03045E' : '#E5484D'}
                >
                  {node.name.replace('Node ', 'N')}
                </text>

                {/* Status Dot */}
                <circle
                  cx={x + 16}
                  cy={y - 16}
                  r="4"
                  fill={node.active ? (isPrimary ? '#22A06B' : '#00B4D8') : '#E5484D'}
                />
              </g>
            )
          })}

          {/* Center Hub Indicator */}
          <circle cx={centerX} cy={centerY} r="28" fill="#F8FAFC" stroke="#CBD5E1" strokeWidth="1.5" />
          <text
            x={centerX}
            y={centerY + 3}
            textAnchor="middle"
            fontSize="9"
            fontFamily="JetBrains Mono, monospace"
            fontWeight="bold"
            fill="#0077B6"
          >
            150 VN
          </text>
        </svg>

        {/* Floating Ring Key Indicator */}
        <div className="absolute bottom-2 left-2 bg-white/90 backdrop-blur-xs px-2.5 py-1 rounded-md border border-[#E2E8F0] text-[10px] font-mono text-[#64748B]">
          <span className="text-[#0077B6] font-bold">SHA-256</span> 32-bit Integer Space
        </div>
      </div>

      {/* 2. Right Interactive Chunk & Placement Details */}
      <div className="flex-1 w-full space-y-4">
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-xs font-mono font-bold uppercase text-[#64748B]">
              Select Active Chunk
            </span>
            <span className="text-xs font-mono text-[#0077B6]">
              N=3 Quorum Placement
            </span>
          </div>

          {/* Chunk Selector Pills */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
            {sampleChunks.map((chunk) => {
              const isSelected = chunk.id === activeChunk.id
              return (
                <button
                  key={chunk.id}
                  onClick={() => {
                    setCurrentChunkId(chunk.id)
                    onSelectChunk?.(chunk.id)
                  }}
                  className={`p-2.5 rounded-lg border text-left font-mono transition-all ${
                    isSelected
                      ? 'border-[#00B4D8] bg-[#CAF0F8] text-[#03045E] shadow-2xs font-bold'
                      : 'border-[#E2E8F0] bg-white text-[#475569] hover:bg-[#F8FAFC]'
                  }`}
                >
                  <div className="text-xs">{chunk.name}</div>
                  <div className="text-[10px] text-[#64748B]">{chunk.sizeMb} MB</div>
                </button>
              )
            })}
          </div>
        </div>

        {/* Placement Info Card */}
        <div className="p-4 rounded-xl border border-[#E2E8F0] bg-[#F8FAFC] space-y-3 font-mono text-xs">
          <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-2">
            <span className="text-[#64748B]">Active Chunk Key</span>
            <span className="font-bold text-[#03045E]">{activeChunk.hash}</span>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="p-2.5 rounded-lg bg-white border border-[#CBD5E1]">
              <span className="text-[10px] text-[#0077B6] font-bold block uppercase">Primary Replica</span>
              <span className="text-sm font-bold text-[#03045E]">{activeChunk.primaryNode}</span>
              <span className="text-[10px] text-[#64748B] block mt-0.5">First successor on ring</span>
            </div>

            <div className="p-2.5 rounded-lg bg-white border border-[#CBD5E1]">
              <span className="text-[10px] text-[#00B4D8] font-bold block uppercase">Replicas (N=3)</span>
              <span className="text-sm font-bold text-[#03045E]">{activeChunk.replicas.join(', ')}</span>
              <span className="text-[10px] text-[#64748B] block mt-0.5">Next distinct physical nodes</span>
            </div>
          </div>

          <div className="flex items-center gap-2 text-[11px] text-[#475569] pt-1">
            <Info className="w-4 h-4 text-[#0077B6] shrink-0" />
            <span>
              150 virtual nodes per physical server ensure balanced distribution with variance &lt; 5%.
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
