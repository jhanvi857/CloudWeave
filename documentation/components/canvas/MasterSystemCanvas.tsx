'use client'

import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Play,
  Pause,
  RotateCcw,
  Gauge,
  Check,
  X,
  FileVideo,
  Layers,
  Server,
  ShieldCheck,
  Zap,
} from 'lucide-react'

export interface ChunkItem {
  id: string
  name: string
  sizeMb: number
  hashShort: string
  angleDeg: number
  primaryNode: number
  replicas: number[]
}

const chunksData: ChunkItem[] = [
  { id: 'a', name: 'Chunk A', sizeMb: 16, hashShort: '8f3a...91c2', angleDeg: 54, primaryNode: 2, replicas: [3, 4] },
  { id: 'b', name: 'Chunk B', sizeMb: 16, hashShort: '17bd...82a1', angleDeg: 195, primaryNode: 4, replicas: [5, 1] },
  { id: 'c', name: 'Chunk C', sizeMb: 16, hashShort: 'a92c...11ef', angleDeg: 330, primaryNode: 1, replicas: [2, 3] },
  { id: 'd', name: 'Chunk N', sizeMb: 16, hashShort: '7b3f...91c8', angleDeg: 120, primaryNode: 3, replicas: [1, 5] },
]

// Ring center and radius in 900x560 coordinate space
const CX = 450
const CY = 345
const RING_R = 125

// 5 Node Cartesian positions around the ring
const nodeCoords = [
  { id: 1, name: 'Node 1', x: 450, y: 195, role: 'Replica', cap: '2.1 GB / 10 GB', chunks: 15, barW: 42 },
  { id: 2, name: 'Node 2', x: 595, y: 255, role: 'Peer', cap: '2.4 GB / 10 GB', chunks: 15, barW: 48 },
  { id: 3, name: 'Node 3', x: 550, y: 440, role: 'Primary', cap: '2.1 GB / 10 GB', chunks: 14, barW: 42 },
  { id: 4, name: 'Node 4', x: 350, y: 440, role: 'Peer', cap: '2.0 GB / 10 GB', chunks: 13, barW: 40 },
  { id: 5, name: 'Node 5', x: 305, y: 255, role: 'Replica', cap: '1.8 GB / 10 GB', chunks: 11, barW: 36 },
]

// 120 tick marks for consistent hash ring
const virtualTicks = Array.from({ length: 90 }).map((_, i) => {
  const angle = (i * 360) / 90
  const rad = ((angle - 90) * Math.PI) / 180
  const x = Math.round((CX + RING_R * Math.cos(rad)) * 100) / 100
  const y = Math.round((CY + RING_R * Math.sin(rad)) * 100) / 100
  return { id: i, x, y }
})

export function MasterSystemCanvas() {
  const [step, setStep] = useState<number>(3)
  const [isPlaying, setIsPlaying] = useState<boolean>(true)
  const [speed, setSpeed] = useState<number>(1.0)
  const [selectedChunkId, setSelectedChunkId] = useState<string>('d')
  const [inspectedNode, setInspectedNode] = useState<number | null>(null)

  const activeChunk = chunksData.find((c) => c.id === selectedChunkId) || chunksData[3]

  const stepsList = [
    { num: 1, label: 'Upload' },
    { num: 2, label: 'Chunking' },
    { num: 3, label: 'Distribution' },
    { num: 4, label: 'Replication' },
    { num: 5, label: 'Complete' },
  ]

  const stepDescriptions = [
    'Stage 1: S3 PUT /video.mp4 (2.4 GB) stream arrives via AWS SigV4 REST gateway',
    'Stage 2: FastCDC Gear rolling hash computes dynamic chunk boundaries (16 KB - 256 KB)',
    'Stage 3: SHA-256 CAS chunk IDs are routed to primary storage nodes via 150-vnode hash ring',
    'Stage 4: Asynchronous quorum fan-out replicates chunk copies to N=3 distinct physical hosts',
    'Stage 5: Quorum achieved (W=2/2 fsync ACKs) · Object manifest atomically committed to WAL',
  ]

  useEffect(() => {
    let timer: NodeJS.Timeout
    if (isPlaying) {
      const duration = 2800 / speed
      timer = setTimeout(() => {
        setStep((prev) => (prev >= 5 ? 1 : prev + 1))
      }, duration)
    }
    return () => clearTimeout(timer)
  }, [isPlaying, step, speed])

  const activeRad = ((activeChunk.angleDeg - 90) * Math.PI) / 180
  const activePinX = Math.round((CX + RING_R * Math.cos(activeRad)) * 100) / 100
  const activePinY = Math.round((CY + RING_R * Math.sin(activeRad)) * 100) / 100

  return (
    <div className="w-full flex flex-col bg-white rounded-2xl border-2 border-[#CBD5E1] shadow-sm select-none font-sans overflow-hidden">
      {/* 1. Header Toolbar with Step Tabs */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between px-5 py-4 border-b border-[#E2E8F0] bg-[#F8FAFC] gap-3">
        <div>
          <div className="flex items-center gap-2">
            <span className="flex h-2 w-2 rounded-full bg-[#00B4D8] animate-pulse" />
            <span className="text-[11px] font-mono font-bold uppercase tracking-wider text-[#0077B6]">
              STEP {step} OF 5 · {stepsList[step - 1].label}
            </span>
          </div>
          <h3 className="text-base sm:text-lg font-bold text-[#03045E] mt-0.5">
            {step === 1 && '1. S3 Client Request Ingestion'}
            {step === 2 && '2. FastCDC Content-Defined Chunking'}
            {step === 3 && '3. Consistent Hash Ring Placement'}
            {step === 4 && '4. Multi-Node Mesh Replication (N=3)'}
            {step === 5 && '5. Write Quorum (W=2) & WAL Commit'}
          </h3>
        </div>

        {/* Stepper Buttons */}
        <div className="flex items-center gap-1.5 font-mono text-xs overflow-x-auto">
          {stepsList.map((s) => {
            const isCurr = step === s.num
            const isDone = step > s.num
            return (
              <button
                key={s.num}
                onClick={() => setStep(s.num)}
                className={`flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg font-bold transition-all cursor-pointer ${
                  isCurr
                    ? 'bg-[#0077B6] text-white shadow-xs'
                    : isDone
                    ? 'bg-[#CAF0F8] text-[#0077B6] hover:bg-[#B3E8F5]'
                    : 'bg-white border border-[#CBD5E1] text-[#64748B] hover:bg-[#F1F5F9]'
                }`}
              >
                <span>{isDone ? <Check className="w-3 h-3 inline" /> : s.num}</span>
                <span className="hidden md:inline">{s.label}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* 2. Main High-Precision SVG Canvas */}
      <div className="w-full relative bg-[#FFFFFF] p-2 sm:p-4">
        <svg
          viewBox="0 0 900 550"
          className="w-full h-auto max-h-[580px]"
          style={{ shapeRendering: 'geometricPrecision' }}
        >
          <defs>
            {/* Soft Shadow Filter */}
            <filter id="cardShadow" x="-10%" y="-10%" width="120%" height="130%">
              <feDropShadow dx="0" dy="2" stdDeviation="3" floodOpacity="0.08" floodColor="#03045E" />
            </filter>
            <filter id="activeGlow" x="-20%" y="-20%" width="140%" height="140%">
              <feDropShadow dx="0" dy="0" stdDeviation="4" floodOpacity="0.35" floodColor="#00B4D8" />
            </filter>
          </defs>

          {/* ============================================================ */}
          {/* TOP SECTION: S3 INGRESS & FASTCDC                            */}
          {/* ============================================================ */}

          {/* S3 Ingress Card (Top Center) */}
          <g transform="translate(370, 15)">
            <rect
              width="160"
              height="44"
              rx="10"
              fill="#FFFFFF"
              stroke="#0077B6"
              strokeWidth="1.5"
              filter="url(#cardShadow)"
            />
            <circle cx="22" cy="22" r="12" fill="#CAF0F8" />
            <polygon points="19,16 27,22 19,28" fill="#0077B6" />
            <text x="42" y="20" fontSize="11" fontFamily="Inter, sans-serif" fontWeight="bold" fill="#03045E">
              video.mp4
            </text>
            <text x="42" y="34" fontSize="9" fontFamily="JetBrains Mono, monospace" fill="#64748B">
              2.4 GB Stream (S3 PUT)
            </text>
          </g>

          {/* Downward line from Ingress to FastCDC */}
          <line x1="450" y1="59" x2="450" y2="80" stroke="#00B4D8" strokeWidth="2" strokeDasharray="3 3" />

          {/* Stage 1 Traveling Packet */}
          {step === 1 && (
            <motion.circle
              r="5"
              fill="#00B4D8"
              animate={{ cx: [450, 450], cy: [59, 80] }}
              transition={{ repeat: Infinity, duration: 0.8 / speed, ease: 'linear' }}
            />
          )}

          {/* FastCDC Engine Badge */}
          <g transform="translate(350, 80)">
            <rect
              width="200"
              height="30"
              rx="8"
              fill="#CAF0F8"
              stroke="#00B4D8"
              strokeWidth="1.5"
            />
            <text
              x="100"
              y="19"
              textAnchor="middle"
              fontSize="10"
              fontFamily="JetBrains Mono, monospace"
              fontWeight="bold"
              fill="#0077B6"
            >
              FastCDC Gear Hash (16-256 KB)
            </text>
          </g>

          {/* Downward branching line to Chunks */}
          <line x1="450" y1="110" x2="450" y2="128" stroke="#00B4D8" strokeWidth="2" strokeDasharray="3 3" />

          {/* ============================================================ */}
          {/* CHUNK ROW (4 CAS Chunks)                                     */}
          {/* ============================================================ */}
          {chunksData.map((chunk, idx) => {
            const chunkX = 175 + idx * 145
            const chunkY = 128
            const isSelected = selectedChunkId === chunk.id

            return (
              <g
                key={chunk.id}
                transform={`translate(${chunkX}, ${chunkY})`}
                className="cursor-pointer"
                onClick={() => setSelectedChunkId(chunk.id)}
              >
                {/* Chunk Card Box */}
                <rect
                  width="120"
                  height="44"
                  rx="8"
                  fill={isSelected ? '#CAF0F8' : '#FFFFFF'}
                  stroke={isSelected ? '#0077B6' : '#CBD5E1'}
                  strokeWidth={isSelected ? '2' : '1.5'}
                  filter={isSelected ? 'url(#activeGlow)' : 'url(#cardShadow)'}
                />
                <text x="10" y="18" fontSize="10" fontFamily="Inter, sans-serif" fontWeight="bold" fill="#03045E">
                  {chunk.name}
                </text>
                <text x="75" y="18" fontSize="9" fontFamily="JetBrains Mono, monospace" fill="#64748B">
                  {chunk.sizeMb} MB
                </text>
                <text x="10" y="34" fontSize="8.5" fontFamily="JetBrains Mono, monospace" fill="#0077B6">
                  {chunk.hashShort}
                </text>
                {isSelected && (
                  <circle cx="108" cy="14" r="4" fill="#0077B6" />
                )}
              </g>
            )
          })}

          {/* Downward line from selected chunk to ring */}
          <line x1="450" y1="172" x2="450" y2="210" stroke="#00B4D8" strokeWidth="2" strokeDasharray="3 3" />

          {/* ============================================================ */}
          {/* CONSISTENT HASH RING & TOPOLOGY                              */}
          {/* ============================================================ */}

          {/* Base Ring Circle */}
          <circle
            cx={CX}
            cy={CY}
            r={RING_R}
            fill="none"
            stroke="#90E0EF"
            strokeWidth="3"
          />

          {/* 90 Virtual Node Tick Dots */}
          {virtualTicks.map((vt) => (
            <circle
              key={vt.id}
              cx={vt.x}
              cy={vt.y}
              r="1.5"
              fill="#0077B6"
              opacity="0.45"
            />
          ))}

          {/* Active Key Placement Beacon on the Ring */}
          <g>
            <circle cx={activePinX} cy={activePinY} r="7" fill="#00B4D8" opacity="0.3" />
            <circle cx={activePinX} cy={activePinY} r="4" fill="#0077B6" />
          </g>

          {/* Central Ring Hub */}
          <circle cx={CX} cy={CY} r="38" fill="#CAF0F8" stroke="#0077B6" strokeWidth="2" filter="url(#cardShadow)" />
          <text x={CX} y={CY - 6} textAnchor="middle" fontSize="9" fontFamily="Inter, sans-serif" fontWeight="bold" fill="#03045E">
            Consistent
          </text>
          <text x={CX} y={CY + 7} textAnchor="middle" fontSize="9" fontFamily="Inter, sans-serif" fontWeight="bold" fill="#03045E">
            Hash Ring
          </text>
          <text x={CX} y={CY + 18} textAnchor="middle" fontSize="7.5" fontFamily="JetBrains Mono, monospace" fill="#0077B6">
            150 VNodes
          </text>

          {/* Step 2/3: Traveling Particle from Chunk to Ring Hub */}
          {(step === 2 || step === 3) && (
            <motion.circle
              r="5"
              fill="#00B4D8"
              animate={{ cx: [450, CX], cy: [172, CY] }}
              transition={{ repeat: Infinity, duration: 1.0 / speed, ease: 'easeInOut' }}
            />
          )}

          {/* Step 3, 4, 5: Radial Network Flow Lines from Ring to Nodes */}
          {step >= 3 &&
            nodeCoords.map((node) => {
              const isPrimary = node.id === activeChunk.primaryNode
              const isReplica = activeChunk.replicas.includes(node.id)
              const isTarget = isPrimary || (step >= 4 && isReplica)

              return (
                <g key={`mesh-flow-${node.id}`}>
                  <line
                    x1={CX}
                    y1={CY}
                    x2={node.x}
                    y2={node.y}
                    stroke={isTarget ? (isPrimary ? '#0077B6' : '#00B4D8') : '#E2E8F0'}
                    strokeWidth={isTarget ? '2.5' : '1'}
                    strokeDasharray={isTarget ? '4 4' : '2 2'}
                  />
                  {isTarget && (
                    <motion.circle
                      r="5"
                      fill={isPrimary ? '#0077B6' : '#00B4D8'}
                      animate={{ cx: [CX, node.x], cy: [CY, node.y] }}
                      transition={{ repeat: Infinity, duration: 1.1 / speed, ease: 'easeInOut' }}
                    />
                  )}
                </g>
              )
            })}

          {/* Step 5: Return Acks to Ring Hub (Quorum Achieved) */}
          {step === 5 && (
            <g>
              <motion.circle
                r="6"
                fill="#22A06B"
                animate={{ cx: [550, CX], cy: [440, CY] }}
                transition={{ repeat: Infinity, duration: 0.9 / speed, ease: 'easeOut' }}
              />
              <motion.circle
                r="6"
                fill="#22A06B"
                animate={{ cx: [450, CX], cy: [195, CY] }}
                transition={{ repeat: Infinity, duration: 0.9 / speed, ease: 'easeOut' }}
              />
            </g>
          )}

          {/* ============================================================ */}
          {/* 5 STORAGE NODE CARDS                                         */}
          {/* ============================================================ */}
          {nodeCoords.map((node) => {
            const isPrimary = node.id === activeChunk.primaryNode && step >= 3
            const isReplica = activeChunk.replicas.includes(node.id) && step >= 4
            const isAcked = step === 5 && (isPrimary || isReplica)
            const isInspected = inspectedNode === node.id

            return (
              <g
                key={node.id}
                transform={`translate(${node.x - 58}, ${node.y - 28})`}
                className="cursor-pointer"
                onClick={() => setInspectedNode(node.id === inspectedNode ? null : node.id)}
              >
                {/* Node Outer Container */}
                <rect
                  width="116"
                  height="56"
                  rx="10"
                  fill={isPrimary ? '#CAF0F8' : isReplica ? '#E8F8F0' : '#FFFFFF'}
                  stroke={isPrimary ? '#0077B6' : isReplica ? '#22A06B' : '#CBD5E1'}
                  strokeWidth={isPrimary || isReplica ? '2.5' : '1.5'}
                  filter="url(#cardShadow)"
                />

                {/* Node Title */}
                <text x="10" y="18" fontSize="11" fontFamily="Inter, sans-serif" fontWeight="bold" fill="#03045E">
                  {node.name}
                </text>

                {/* Health Dot */}
                <circle cx="102" cy="14" r="4" fill="#22A06B" />

                {/* Role Pill */}
                <text
                  x="10"
                  y="32"
                  fontSize="8.5"
                  fontFamily="JetBrains Mono, monospace"
                  fontWeight="bold"
                  fill={isPrimary ? '#0077B6' : isReplica ? '#22A06B' : '#64748B'}
                >
                  {isPrimary ? '● Primary Node' : isReplica ? '● Replica Node' : `${node.chunks} chunks stored`}
                </text>

                {/* Mini Storage Progress Bar */}
                <rect x="10" y="40" width="96" height="4" rx="2" fill="#E2E8F0" />
                <rect
                  x="10"
                  y="40"
                  width={node.barW}
                  height="4"
                  rx="2"
                  fill={isPrimary ? '#0077B6' : isReplica ? '#22A06B' : '#94A3B8'}
                />

                {/* Quorum ACK Checkmark (Step 5) */}
                {isAcked && (
                  <g transform="translate(98, -6)">
                    <circle cx="8" cy="8" r="9" fill="#22A06B" />
                    <polyline points="5,8 7.5,11 11.5,5" fill="none" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </g>
                )}
              </g>
            )
          })}

          {/* ============================================================ */}
          {/* STEP 5 QUORUM BANNER                                         */}
          {/* ============================================================ */}
          {step === 5 && (
            <g transform="translate(250, 500)">
              <rect
                width="400"
                height="34"
                rx="17"
                fill="#E8F8F0"
                stroke="#22A06B"
                strokeWidth="1.5"
                filter="url(#cardShadow)"
              />
              <circle cx="20" cy="17" r="9" fill="#22A06B" />
              <polyline points="17,17 19.5,20 23.5,14" fill="none" stroke="#FFFFFF" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
              <text
                x="40"
                y="21"
                fontSize="10.5"
                fontFamily="JetBrains Mono, monospace"
                fontWeight="bold"
                fill="#22A06B"
              >
                QUORUM COMMITTED (W=2/2 ACKs) - fsync OK
              </text>
            </g>
          )}
        </svg>
      </div>

      {/* 3. Bottom Playback Control Strip & Live Story Line */}
      <div className="flex flex-col sm:flex-row items-center justify-between px-5 py-3.5 border-t border-[#E2E8F0] bg-[#F8FAFC] gap-3 font-mono text-xs">
        {/* Story Notice */}
        <div className="flex items-center gap-2 text-[#03045E] text-xs font-semibold">
          <span className="px-2 py-0.5 rounded bg-[#CAF0F8] text-[#0077B6] font-bold text-[10px]">
            EVENT
          </span>
          <span>{stepDescriptions[step - 1]}</span>
        </div>

        {/* Playback Controls */}
        <div className="flex items-center gap-2 shrink-0">
          <button
            onClick={() => setIsPlaying(!isPlaying)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[#0077B6] text-white hover:bg-[#005F92] transition-colors shadow-xs text-xs font-bold cursor-pointer"
          >
            {isPlaying ? <Pause className="w-3.5 h-3.5 fill-current" /> : <Play className="w-3.5 h-3.5 fill-current" />}
            <span>{isPlaying ? 'Pause' : 'Play'}</span>
          </button>

          <button
            onClick={() => setStep((prev) => (prev >= 5 ? 1 : prev + 1))}
            className="px-3 py-1.5 rounded-lg border border-[#CBD5E1] bg-white text-[#03045E] hover:bg-[#F1F5F9] font-bold transition-colors cursor-pointer"
          >
            Step →
          </button>

          <button
            onClick={() => {
              setIsPlaying(false)
              setStep(1)
            }}
            className="p-1.5 rounded-lg border border-[#CBD5E1] bg-white text-[#64748B] hover:text-[#03045E] hover:bg-[#F1F5F9] transition-colors cursor-pointer"
            title="Reset"
          >
            <RotateCcw className="w-3.5 h-3.5" />
          </button>

          <div className="flex items-center gap-1.5 pl-2 text-[10px] text-[#64748B]">
            <Gauge className="w-3.5 h-3.5 text-[#0077B6]" />
            <span className="font-bold text-[#0077B6]">{speed}x</span>
            <input
              type="range"
              min="0.5"
              max="2"
              step="0.5"
              value={speed}
              onChange={(e) => setSpeed(parseFloat(e.target.value))}
              className="w-14 h-1.5 bg-[#CBD5E1] rounded-lg appearance-none cursor-pointer accent-[#0077B6]"
            />
          </div>
        </div>
      </div>
    </div>
  )
}
