'use client'

import React, { useState } from 'react'
import { Cpu, Check, X, RefreshCw } from 'lucide-react'

interface ECShard {
  id: string
  name: string
  node: string
  type: 'data' | 'parity'
  alive: boolean
}

export function ErasureCanvas() {
  const [shards, setShards] = useState<ECShard[]>([
    { id: 'd1', name: 'D1 (Data)', node: 'Node 1', type: 'data', alive: true },
    { id: 'd2', name: 'D2 (Data)', node: 'Node 2', type: 'data', alive: true },
    { id: 'd3', name: 'D3 (Data)', node: 'Node 3', type: 'data', alive: true },
    { id: 'd4', name: 'D4 (Data)', node: 'Node 4', type: 'data', alive: true },
    { id: 'p1', name: 'P1 (Parity)', node: 'Node 5', type: 'parity', alive: true },
    { id: 'p2', name: 'P2 (Parity)', node: 'Node 6', type: 'parity', alive: true },
  ])

  const toggleShard = (id: string) => {
    setShards((prev) =>
      prev.map((s) => (s.id === id ? { ...s, alive: !s.alive } : s))
    )
  }

  const aliveCount = shards.filter((s) => s.alive).length
  const canReconstruct = aliveCount >= 4

  const handleReset = () => {
    setShards((prev) => prev.map((s) => ({ ...s, alive: true })))
  }

  return (
    <div className="flex flex-col items-center justify-between w-full min-h-[560px] bg-white rounded-2xl border border-[#E2E8F0] shadow-sm p-6 select-none relative space-y-6 font-sans">
      {/* Top Header */}
      <div className="w-full flex items-center justify-between border-b border-[#E2E8F0] pb-3 font-mono text-xs">
        <div className="flex items-center gap-2">
          <Cpu className="w-4 h-4 text-[#0077B6]" />
          <span className="font-bold text-[#03045E]">REED-SOLOMON ERASURE CODING (K=4, M=2)</span>
        </div>

        <button
          onClick={handleReset}
          className="flex items-center gap-1.5 text-xs text-[#0077B6] hover:underline transition-colors font-bold cursor-pointer"
        >
          <RefreshCw className="w-3.5 h-3.5" />
          <span>Restore Shards</span>
        </button>
      </div>

      {/* 6 Shards Grid */}
      <div className="w-full max-w-2xl font-mono text-xs space-y-2">
        <span className="text-[11px] text-[#64748B] uppercase font-bold block text-center mb-2">
          Click any shard to simulate node failure
        </span>

        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          {shards.map((shard) => {
            return (
              <div
                key={shard.id}
                onClick={() => toggleShard(shard.id)}
                className={`p-3.5 rounded-xl border cursor-pointer text-center transition-all ${
                  !shard.alive
                    ? 'border-[#E5484D]/40 bg-[#FFF1F2]'
                    : 'border-[#CBD5E1] bg-[#F8FAFC] hover:border-[#0077B6]/50 shadow-2xs'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-bold text-[#03045E]">{shard.name.split(' ')[0]}</span>
                  <div
                    className={`h-4 w-4 rounded-full flex items-center justify-center text-[10px] ${
                      shard.alive ? 'bg-[#22A06B] text-white font-bold' : 'bg-[#E5484D] text-white font-bold'
                    }`}
                  >
                    {shard.alive ? <Check className="w-2.5 h-2.5" /> : <X className="w-2.5 h-2.5" />}
                  </div>
                </div>

                <div className="mt-2 text-[10px] text-[#64748B]">
                  <div>{shard.node}</div>
                  <div className={`font-bold mt-0.5 ${shard.alive ? 'text-[#03045E]' : 'text-[#E5484D]'}`}>
                    {shard.alive ? 'Online' : 'LOST'}
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Math Reconstruction Status */}
      <div
        className={`w-full max-w-xl text-center py-3 px-6 rounded-full border font-mono text-xs font-bold shadow-2xs ${
          canReconstruct
            ? 'border-[#22A06B]/40 bg-[#E8F8F0] text-[#22A06B]'
            : 'border-[#E5484D]/40 bg-[#FFF1F2] text-[#E5484D]'
        }`}
      >
        {canReconstruct
          ? `Object reconstructible (${aliveCount}/4 required shards available - 1.5x storage overhead)`
          : `Data loss: ${6 - aliveCount} node failures exceeds M=2 parity limit`}
      </div>
    </div>
  )
}
