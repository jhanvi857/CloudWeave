'use client'

import React, { useState } from 'react'
import { Activity, ShieldCheck, Layers, Cpu, Sliders } from 'lucide-react'
import { FailureRecoverySimulator } from '../playground/FailureRecoverySimulator'
import { QuorumVisualizer } from '../playground/QuorumVisualizer'
import { DeduplicationPlayground } from '../playground/DeduplicationPlayground'
import { ErasureCodingPlayground } from '../playground/ErasureCodingPlayground'

export function ExperimentsView() {
  const [activeExp, setActiveExp] = useState<'failure' | 'quorum' | 'dedup' | 'erasure'>('failure')

  return (
    <div className="space-y-6">
      {/* Title */}
      <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs">
        <div className="flex items-center gap-2">
          <span className="rounded-md bg-[#CAF0F8] px-2.5 py-1 text-[10px] font-mono font-bold text-[#0077B6] uppercase">
            Interactive Experiments Lab
          </span>
          <span className="text-xs font-mono text-[#64748B]">Real-Time Simulations</span>
        </div>
        <h2 className="text-2xl font-bold tracking-tight text-[#03045E] mt-2">
          Distributed Systems Experiments
        </h2>
        <p className="text-xs sm:text-sm text-[#64748B] mt-1 max-w-2xl">
          Conduct live interactive experiments with node failures, quorum overlap gaps, FastCDC rolling hash shifts, and Reed-Solomon parity reconstruction.
        </p>

        {/* Exp Selector */}
        <div className="flex flex-wrap gap-2 mt-4 pt-3 border-t border-[#E2E8F0] font-mono text-xs">
          <button
            onClick={() => setActiveExp('failure')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold transition-all ${
              activeExp === 'failure'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'bg-[#F8FAFC] text-[#475569] border border-[#CBD5E1] hover:bg-[#CAF0F8]'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            <span>1. Node Failure & Recovery</span>
          </button>

          <button
            onClick={() => setActiveExp('quorum')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold transition-all ${
              activeExp === 'quorum'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'bg-[#F8FAFC] text-[#475569] border border-[#CBD5E1] hover:bg-[#CAF0F8]'
            }`}
          >
            <ShieldCheck className="w-3.5 h-3.5" />
            <span>2. Quorum Tuning Lab</span>
          </button>

          <button
            onClick={() => setActiveExp('dedup')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold transition-all ${
              activeExp === 'dedup'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'bg-[#F8FAFC] text-[#475569] border border-[#CBD5E1] hover:bg-[#CAF0F8]'
            }`}
          >
            <Layers className="w-3.5 h-3.5" />
            <span>3. FastCDC Deduplication</span>
          </button>

          <button
            onClick={() => setActiveExp('erasure')}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold transition-all ${
              activeExp === 'erasure'
                ? 'bg-[#0077B6] text-white shadow-2xs'
                : 'bg-[#F8FAFC] text-[#475569] border border-[#CBD5E1] hover:bg-[#CAF0F8]'
            }`}
          >
            <Cpu className="w-3.5 h-3.5" />
            <span>4. Erasure Coding Lab</span>
          </button>
        </div>
      </div>

      {/* Embedded Lab Component */}
      <div>
        {activeExp === 'failure' && <FailureRecoverySimulator />}
        {activeExp === 'quorum' && <QuorumVisualizer />}
        {activeExp === 'dedup' && <DeduplicationPlayground />}
        {activeExp === 'erasure' && <ErasureCodingPlayground />}
      </div>
    </div>
  )
}
