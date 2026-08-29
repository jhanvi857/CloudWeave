'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { ShieldCheck, Server, Check, X, AlertTriangle, ArrowDown } from 'lucide-react'

export function QuorumVisualizer() {
  const [w, setW] = useState(2)
  const [r, setR] = useState(2)
  const [n] = useState(3)
  const [failedNode, setFailedNode] = useState<number | null>(3) // Node 3 is failed by default to show W=2/3 behavior

  const toggleNodeFail = (nodeIdx: number) => {
    setFailedNode((prev) => (prev === nodeIdx ? null : nodeIdx))
  }

  // Count active acknowledgments
  const acks = [1, 2, 3].map((nodeNum) => ({
    node: `Node ${nodeNum}`,
    nodeNum,
    acknowledged: failedNode !== nodeNum,
  }))

  const ackCount = acks.filter((a) => a.acknowledged).length
  const writeSuccess = ackCount >= w
  const strongConsistency = w + r > n

  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      {/* Header & Controls */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#E2E8F0] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-5 h-5 text-[#0077B6]" />
            <h3 className="font-bold text-base text-[#03045E]">Tunable Quorum Consensus (N / W / R)</h3>
          </div>
          <p className="text-xs text-[#64748B] mt-0.5">
            Dynamo-style quorum ensures durability and availability trade-offs can be tuned per-bucket.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <div className="px-3 py-1.5 rounded-lg bg-[#CAF0F8] border border-[#90E0EF] text-xs font-mono font-bold text-[#0077B6]">
            N = {n} Replicas
          </div>
          <div
            className={`px-3 py-1.5 rounded-lg border text-xs font-mono font-bold ${
              writeSuccess
                ? 'bg-[#E8F8F0] border-[#A8E6CF] text-[#22A06B]'
                : 'bg-[#FEECEB] border-[#FECACA] text-[#E5484D]'
            }`}
          >
            {writeSuccess ? 'WRITE QUORUM ACHIEVED' : 'WRITE QUORUM FAILED'}
          </div>
        </div>
      </div>

      {/* Sliders Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 bg-[#F8FAFC] p-4 rounded-xl border border-[#E2E8F0]">
        <div>
          <div className="flex items-center justify-between mb-1.5 text-xs font-mono">
            <span className="font-semibold text-[#03045E]">Write Quorum (W)</span>
            <span className="font-bold text-[#0077B6] bg-white px-2 py-0.5 rounded border border-[#CBD5E1]">
              W = {w}
            </span>
          </div>
          <input
            type="range"
            min="1"
            max="3"
            value={w}
            onChange={(e) => setW(Number(e.target.value))}
            className="w-full h-2 bg-[#CBD5E1] rounded-lg appearance-none cursor-pointer accent-[#0077B6]"
          />
          <span className="text-[10px] text-[#64748B] block mt-1">
            Number of storage replicas required to synchronously acknowledge write before 201 response.
          </span>
        </div>

        <div>
          <div className="flex items-center justify-between mb-1.5 text-xs font-mono">
            <span className="font-semibold text-[#03045E]">Read Quorum (R)</span>
            <span className="font-bold text-[#0077B6] bg-white px-2 py-0.5 rounded border border-[#CBD5E1]">
              R = {r}
            </span>
          </div>
          <input
            type="range"
            min="1"
            max="3"
            value={r}
            onChange={(e) => setR(Number(e.target.value))}
            className="w-full h-2 bg-[#CBD5E1] rounded-lg appearance-none cursor-pointer accent-[#0077B6]"
          />
          <span className="text-[10px] text-[#64748B] block mt-1">
            Number of replicas queried during GET to compare vector clocks and return freshest version.
          </span>
        </div>
      </div>

      {/* Interactive Coordinator Fan-out Diagram */}
      <div className="flex flex-col items-center py-2 space-y-4">
        {/* Coordinator Node */}
        <div className="flex flex-col items-center">
          <div className="px-5 py-2.5 rounded-xl border border-[#0077B6] bg-[#CAF0F8] text-center shadow-xs">
            <span className="text-[10px] font-mono font-bold text-[#0077B6] uppercase block">Coordinator Node</span>
            <span className="text-xs font-mono font-bold text-[#03045E]">Fan-out write stream to N=3 nodes</span>
          </div>
          <div className="h-6 w-0.5 bg-[#00B4D8]" />
        </div>

        {/* 3 Replica Target Nodes */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 w-full">
          {acks.map((ack) => {
            const isFailed = failedNode === ack.nodeNum
            return (
              <div
                key={ack.node}
                onClick={() => toggleNodeFail(ack.nodeNum)}
                className={`p-3.5 rounded-xl border cursor-pointer transition-all ${
                  isFailed
                    ? 'border-[#FECACA] bg-[#FEECEB]'
                    : 'border-[#A8E6CF] bg-[#E8F8F0]'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-mono font-bold text-[#03045E]">
                    {ack.node}
                  </span>
                  <div
                    className={`flex h-5 w-5 items-center justify-center rounded-full text-white text-xs font-bold ${
                      isFailed ? 'bg-[#E5484D]' : 'bg-[#22A06B]'
                    }`}
                  >
                    {isFailed ? <X className="w-3.5 h-3.5" /> : <Check className="w-3.5 h-3.5" />}
                  </div>
                </div>

                <div className="mt-2 text-[11px] font-mono">
                  {isFailed ? (
                    <span className="text-[#E5484D] font-bold">Unreachable (Click to toggle)</span>
                  ) : (
                    <span className="text-[#22A06B] font-bold">ACK Received (fsync OK)</span>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      </div>

      {/* Quorum Math Evaluation Banner */}
      <div
        className={`p-3.5 rounded-xl border flex items-start gap-3 text-xs font-mono ${
          strongConsistency
            ? 'bg-[#E8F8F0] border-[#A8E6CF] text-[#22A06B]'
            : 'bg-[#FFF8E6] border-[#FDE68A] text-[#D97706]'
        }`}
      >
        <div className="mt-0.5">
          {strongConsistency ? (
            <ShieldCheck className="w-4 h-4 text-[#22A06B]" />
          ) : (
            <AlertTriangle className="w-4 h-4 text-[#D97706]" />
          )}
        </div>
        <div className="flex-1">
          <div className="font-bold">
            {strongConsistency
              ? `Strong Consistency Overlap Guaranteed (W + R = ${w + r} > N = ${n})`
              : `Eventual Consistency / Overlap Gap (W + R = ${w + r} ≤ N = ${n})`}
          </div>
          <p className="text-[11px] mt-0.5 text-[#475569]">
            {strongConsistency
              ? 'At least one replica will overlap in both write and read quorums, guaranteeing reads always observe the latest vector clock.'
              : 'Favors extreme write availability during partitions, but concurrent reads may observe stale data until anti-entropy reconciles.'}
          </p>
        </div>
      </div>
    </div>
  )
}
