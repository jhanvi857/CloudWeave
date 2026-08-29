'use client'

import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { AlertOctagon, RefreshCw, Activity, Check } from 'lucide-react'

interface MeshNode {
  id: number
  name: string
  x: number
  y: number
  status: 'healthy' | 'timeout' | 'failed' | 'repairing'
  chunks: string[]
}

export function FailureCanvas() {
  const [phase, setPhase] = useState<'idle' | 'lost' | 'failed' | 'repairing' | 'recovered'>('idle')
  const [statusMessage, setStatusMessage] = useState<string>('All 5 cluster nodes reporting 500ms heartbeat OK')

  const initialNodes: MeshNode[] = [
    { id: 1, name: 'Node 1', x: 250, y: 100, status: 'healthy', chunks: ['A', 'B', 'C'] },
    { id: 2, name: 'Node 2', x: 100, y: 220, status: 'healthy', chunks: ['A', 'B'] },
    { id: 3, name: 'Node 3', x: 250, y: 340, status: 'healthy', chunks: ['C', 'D'] }, // Victim with Chunk C
    { id: 4, name: 'Node 4', x: 500, y: 220, status: 'healthy', chunks: ['B', 'D'] }, // Repair target
    { id: 5, name: 'Node 5', x: 400, y: 340, status: 'healthy', chunks: ['A', 'D'] },
  ]

  const [nodesState, setNodesState] = useState<MeshNode[]>(initialNodes)

  const handleKillNode = () => {
    setPhase('lost')
    setStatusMessage('CRITICAL: Node 3 heartbeat lost · 3.0s failure detection timeout...')
    setNodesState((prev) =>
      prev.map((n) => (n.id === 3 ? { ...n, status: 'timeout' } : n))
    )

    // Phase 2: Failed
    setTimeout(() => {
      setPhase('failed')
      setStatusMessage('Node 3 marked OFFLINE · Under-replicated chunks detected (N=2 < 3)')
      setNodesState((prev) =>
        prev.map((n) => (n.id === 3 ? { ...n, status: 'failed', chunks: [] } : n))
      )
    }, 1500)

    // Phase 3: Repairing
    setTimeout(() => {
      setPhase('repairing')
      setStatusMessage('Replica repair worker copying Chunk C from Node 1 → Node 4...')
      setNodesState((prev) =>
        prev.map((n) => (n.id === 4 ? { ...n, status: 'repairing' } : n))
      )
    }, 3200)

    // Phase 4: Recovered
    setTimeout(() => {
      setPhase('recovered')
      setStatusMessage('Cluster Healthy: Chunk C restored on Node 4 - Replication back to N=3')
      setNodesState((prev) =>
        prev.map((n) =>
          n.id === 3
            ? { ...n, status: 'failed', chunks: [] }
            : n.id === 4
            ? { ...n, status: 'healthy', chunks: ['B', 'D', 'C (Repaired)'] }
            : { ...n, status: 'healthy' }
        )
      )
    }, 5400)
  }

  const handleReset = () => {
    setPhase('idle')
    setNodesState(initialNodes)
    setStatusMessage('All 5 cluster nodes reporting 500ms heartbeat OK')
  }

  const coordX = 300
  const coordY = 220

  return (
    <div className="flex flex-col items-center justify-between w-full min-h-[620px] bg-white rounded-2xl border border-[#E2E8F0] shadow-sm p-4 sm:p-8 select-none relative overflow-hidden font-sans">
      {/* Top Header Controls */}
      <div className="w-full flex items-center justify-between border-b border-[#E2E8F0] pb-4 font-mono text-xs">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-[#0077B6]" />
          <span className="font-bold text-[#03045E]">KILL A NODE. WATCH THE CLUSTER RECOVER.</span>
        </div>

        <div>
          {phase === 'idle' ? (
            <button
              onClick={handleKillNode}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg bg-[#E5484D] text-white font-semibold hover:bg-[#D0383D] shadow-sm transition-colors text-xs cursor-pointer"
            >
              <AlertOctagon className="w-4 h-4" />
              <span>Simulate Node Failure</span>
            </button>
          ) : (
            <button
              onClick={handleReset}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg border border-[#CBD5E1] bg-white text-[#0077B6] font-semibold hover:bg-[#F1F5F9] transition-colors text-xs shadow-2xs cursor-pointer"
            >
              <RefreshCw className="w-4 h-4 text-[#0077B6]" />
              <span>Reset Cluster</span>
            </button>
          )}
        </div>
      </div>

      {/* Center SVG Mesh Topology */}
      <div className="w-full max-w-2xl h-[420px] relative">
        <svg viewBox="0 0 600 440" className="w-full h-full">
          {/* Coordinator Center */}
          <circle cx={coordX} cy={coordY} r="24" fill="#CAF0F8" stroke="#0077B6" strokeWidth="1.5" />
          <text
            x={coordX}
            y={coordY + 4}
            textAnchor="middle"
            fontSize="9"
            fontFamily="Inter, sans-serif"
            fontWeight="bold"
            fill="#03045E"
          >
            COORD
          </text>

          {/* Lines from Coordinator to Nodes */}
          {nodesState.map((node) => {
            const isDead = node.status === 'failed' || node.status === 'timeout'
            const isRepair = phase === 'repairing' && (node.id === 1 || node.id === 4)

            return (
              <line
                key={node.id}
                x1={coordX}
                y1={coordY}
                x2={node.x}
                y2={node.y}
                stroke={isDead ? '#E5484D' : isRepair ? '#F59E0B' : '#90E0EF'}
                strokeWidth={isRepair ? '2' : '1.5'}
                strokeDasharray={isDead ? '4 4' : 'none'}
              />
            )
          })}

          {/* Physical Repair stream path between Node 1 and Node 4 */}
          {phase === 'repairing' && (
            <g>
              <line
                x1="250"
                y1="100"
                x2="500"
                y2="220"
                stroke="#F59E0B"
                strokeWidth="2.5"
                strokeDasharray="4 4"
              />
              <motion.circle
                r="6"
                fill="#F59E0B"
                animate={{
                  cx: [250, 500],
                  cy: [100, 220],
                }}
                transition={{ repeat: Infinity, duration: 1.0, ease: 'easeInOut' }}
              />
            </g>
          )}

          {/* 5 Nodes */}
          {nodesState.map((node) => {
            const isDead = node.status === 'failed'
            const isTimeout = node.status === 'timeout'
            const isRepairing = node.status === 'repairing'
            const isRecoveredTarget = phase === 'recovered' && node.id === 4

            return (
              <g key={node.id}>
                {/* Node Circle */}
                <circle
                  cx={node.x}
                  cy={node.y}
                  r="28"
                  fill={isDead ? '#FFF1F2' : isTimeout ? '#FEF3C7' : isRepairing ? '#CAF0F8' : isRecoveredTarget ? '#E8F8F0' : '#FFFFFF'}
                  stroke={isDead ? '#E5484D' : isTimeout ? '#F59E0B' : isRepairing ? '#0077B6' : isRecoveredTarget ? '#22A06B' : '#CBD5E1'}
                  strokeWidth={isRecoveredTarget || isRepairing || isDead ? '2.5' : '1.5'}
                />

                {/* Node Label */}
                <text
                  x={node.x}
                  y={node.y - 2}
                  textAnchor="middle"
                  fontSize="10"
                  fontFamily="Inter, sans-serif"
                  fontWeight="bold"
                  fill={isDead ? '#E5484D' : isTimeout ? '#F59E0B' : isRepairing ? '#0077B6' : isRecoveredTarget ? '#22A06B' : '#03045E'}
                >
                  N{node.id}
                </text>

                {/* Stored Chunks count pill inside node */}
                <text
                  x={node.x}
                  y={node.y + 10}
                  textAnchor="middle"
                  fontSize="8"
                  fontFamily="Inter, sans-serif"
                  fontWeight="medium"
                  fill={isDead ? '#E5484D' : '#64748B'}
                >
                  {isDead ? 'OFFLINE' : `${node.chunks.length} chunks`}
                </text>

                {/* Status Dot */}
                <circle
                  cx={node.x + 20}
                  cy={node.y - 20}
                  r="4.5"
                  fill={isDead ? '#E5484D' : isTimeout ? '#F59E0B' : '#22A06B'}
                />
              </g>
            )
          })}
        </svg>

        {/* Floating Chunk C In Flight during repair */}
        {phase === 'repairing' && (
          <motion.div
            initial={{ left: 250, top: 100, opacity: 0 }}
            animate={{ left: 500, top: 220, opacity: 1 }}
            transition={{ repeat: Infinity, duration: 1.2, ease: 'easeInOut' }}
            className="absolute z-30 -translate-x-1/2 -translate-y-1/2 px-2.5 py-1 rounded-lg bg-[#CAF0F8] border border-[#0077B6]/50 text-[9px] font-mono font-bold text-[#0077B6] shadow-sm"
          >
            Chunk C (Replica)
          </motion.div>
        )}
      </div>

      {/* Bottom Live Single-Line Status Notice */}
      <div className="w-full max-w-xl text-center py-2.5 px-4 rounded-full border border-[#CBD5E1] bg-[#F8FAFC] font-mono text-xs text-[#03045E] shadow-2xs">
        <span
          className={`inline-block h-2 w-2 rounded-full mr-2 ${
            phase === 'failed' || phase === 'lost'
              ? 'bg-[#E5484D]'
              : phase === 'repairing'
              ? 'bg-[#F59E0B] animate-ping'
              : 'bg-[#22A06B]'
          }`}
        />
        <span className="font-medium">{statusMessage}</span>
      </div>
    </div>
  )
}
