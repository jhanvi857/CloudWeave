'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Cpu, Server, Check, X, RefreshCw, Layers, ShieldCheck, Zap } from 'lucide-react'

interface Shard {
  id: string
  name: string
  type: 'data' | 'parity'
  node: string
  alive: boolean
}

export function ErasureCodingPlayground() {
  const [mode, setMode] = useState<'erasure' | 'replication'>('erasure')
  const [shards, setShards] = useState<Shard[]>([
    { id: 'd1', name: 'D1 (Data)', type: 'data', node: 'Node 1', alive: true },
    { id: 'd2', name: 'D2 (Data)', type: 'data', node: 'Node 2', alive: true },
    { id: 'd3', name: 'D3 (Data)', type: 'data', node: 'Node 3', alive: true },
    { id: 'd4', name: 'D4 (Data)', type: 'data', node: 'Node 4', alive: true },
    { id: 'p1', name: 'P1 (Parity)', type: 'parity', node: 'Node 5', alive: true },
    { id: 'p2', name: 'P2 (Parity)', type: 'parity', node: 'Node 6', alive: true },
  ])

  const toggleShard = (id: string) => {
    setShards((prev) =>
      prev.map((s) => (s.id === id ? { ...s, alive: !s.alive } : s))
    )
  }

  const aliveCount = shards.filter((s) => s.alive).length
  const failedCount = 6 - aliveCount
  const canReconstruct = aliveCount >= 4

  const handleReset = () => {
    setShards((prev) => prev.map((s) => ({ ...s, alive: true })))
  }

  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      {/* Header with Mode Toggle */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#E2E8F0] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Cpu className="w-5 h-5 text-[#0077B6]" />
            <h3 className="font-bold text-base text-[#03045E]">
              Reed-Solomon Erasure Coding (K=4, M=2)
            </h3>
          </div>
          <p className="text-xs text-[#64748B] mt-0.5">
            Tolerates up to 2 concurrent node failures with only 1.5x storage overhead.
          </p>
        </div>

        {/* Toggle Mode */}
        <div className="flex items-center gap-2 bg-[#F8FAFC] p-1 rounded-lg border border-[#E2E8F0]">
          <button
            onClick={() => setMode('erasure')}
            className={`px-3 py-1.5 rounded-md text-xs font-mono font-bold transition-all ${
              mode === 'erasure'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'text-[#64748B] hover:text-[#03045E]'
            }`}
          >
            Erasure Coding (1.5x)
          </button>
          <button
            onClick={() => setMode('replication')}
            className={`px-3 py-1.5 rounded-md text-xs font-mono font-bold transition-all ${
              mode === 'replication'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'text-[#64748B] hover:text-[#03045E]'
            }`}
          >
            Replication (3.0x)
          </button>
        </div>
      </div>

      {mode === 'erasure' ? (
        <>
          {/* Shards Status Grid */}
          <div className="space-y-3">
            <div className="flex items-center justify-between font-mono text-xs">
              <span className="text-[#64748B]">Click any node/shard to toggle failure (K=4, M=2)</span>
              <button
                onClick={handleReset}
                className="flex items-center gap-1 text-[#0077B6] hover:underline"
              >
                <RefreshCw className="w-3 h-3" />
                <span>Restore All Shards</span>
              </button>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
              {shards.map((shard) => {
                const isParity = shard.type === 'parity'

                return (
                  <div
                    key={shard.id}
                    onClick={() => toggleShard(shard.id)}
                    className={`p-3 rounded-xl border cursor-pointer font-mono transition-all ${
                      !shard.alive
                        ? 'border-[#FECACA] bg-[#FEECEB]'
                        : isParity
                        ? 'border-[#90E0EF] bg-[#CAF0F8]'
                        : 'border-[#A8E6CF] bg-[#E8F8F0]'
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-bold text-[#03045E]">{shard.name}</span>
                      <div
                        className={`flex h-4 w-4 items-center justify-center rounded-full text-white text-[10px] ${
                          shard.alive ? 'bg-[#22A06B]' : 'bg-[#E5484D]'
                        }`}
                      >
                        {shard.alive ? <Check className="w-3 h-3" /> : <X className="w-3 h-3" />}
                      </div>
                    </div>

                    <div className="mt-2 text-[10px] text-[#64748B]">
                      <div>{shard.node}</div>
                      <div className="font-bold text-[#03045E] mt-0.5">
                        {shard.alive ? 'Online (Shard OK)' : 'FAILED (Lost)'}
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          {/* Mathematical Reconstruction Status */}
          <div
            className={`p-4 rounded-xl border flex items-center justify-between font-mono text-xs ${
              canReconstruct
                ? 'bg-[#E8F8F0] border-[#A8E6CF] text-[#22A06B]'
                : 'bg-[#FEECEB] border-[#FECACA] text-[#E5484D]'
            }`}
          >
            <div className="flex items-center gap-3">
              {canReconstruct ? (
                <ShieldCheck className="w-5 h-5 text-[#22A06B] shrink-0" />
              ) : (
                <X className="w-5 h-5 text-[#E5484D] shrink-0" />
              )}
              <div>
                <span className="font-bold block text-sm">
                  {canReconstruct
                    ? `Object Fully Reconstructible (${aliveCount}/4 required shards available)`
                    : `Permanent Data Loss (${failedCount} failed > M=2 parity limit)`}
                </span>
                <span className="text-[11px] text-[#475569]">
                  {canReconstruct
                    ? 'Cauchy / Vandermonde Galois Field GF(2^8) matrix multiplication reconstructs all lost data.'
                    : 'Too many simultaneous shard failures to mathematically invert the generator matrix.'}
                </span>
              </div>
            </div>

            <span className="text-[10px] font-bold px-2 py-1 rounded bg-white border border-current">
              {failedCount} Node Failures
            </span>
          </div>
        </>
      ) : (
        /* Side-by-Side Replication Mode */
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs">
          <div className="p-4 rounded-xl border border-[#CBD5E1] bg-[#F8FAFC] space-y-3">
            <span className="text-xs font-bold text-[#03045E] block uppercase">
              3-Way Full Replication (N=3)
            </span>
            <div className="space-y-1 text-[#64748B] text-[11px]">
              <div>• Storage Footprint: <b className="text-[#03045E]">3.0x</b> (300% storage overhead)</div>
              <div>• CPU Overhead: <b className="text-[#22A06B]">Minimal</b> (raw byte copying)</div>
              <div>• Read Throughput: <b className="text-[#22A06B]">Highest</b> (stream from any replica)</div>
              <div>• Max Node Failures: <b className="text-[#0077B6]">2 of 3 nodes</b></div>
            </div>
          </div>

          <div className="p-4 rounded-xl border border-[#00B4D8] bg-[#CAF0F8] space-y-3">
            <span className="text-xs font-bold text-[#0077B6] block uppercase">
              Reed-Solomon Erasure Coding (K=4, M=2)
            </span>
            <div className="space-y-1 text-[#03045E] text-[11px]">
              <div>• Storage Footprint: <b className="text-[#22A06B]">1.5x</b> (50% storage overhead)</div>
              <div>• CPU Overhead: <b className="text-[#D97706]">Moderate</b> (Galois matrix math)</div>
              <div>• Read Reconstruction: <b className="text-[#64748B]">On failure only</b></div>
              <div>• Max Node Failures: <b className="text-[#0077B6]">2 of 6 nodes</b></div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
