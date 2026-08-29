'use client'

import React, { useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { VaultRing3D } from '../canvas/VaultRing3D'
import { FailureRecovery3D } from '../canvas/FailureRecovery3D'
import { DedupCanvas } from '../canvas/DedupCanvas'
import { ErasureCoding3D } from '../canvas/ErasureCoding3D'
import {
  Play,
  Activity,
  Layers,
  Cpu,
  ShieldCheck,
  Clock,
} from 'lucide-react'

interface PlaygroundShellProps {
  activeExperiment: 'pipeline' | 'failure' | 'dedup' | 'erasure'
  setActiveExperiment: (exp: 'pipeline' | 'failure' | 'dedup' | 'erasure') => void
}

export function PlaygroundShell({
  activeExperiment,
  setActiveExperiment,
}: PlaygroundShellProps) {
  const experiments = [
    { id: 'pipeline', label: '1. Consistent Hash Ring', icon: Play },
    { id: 'failure', label: '2. Failure Recovery (kill -9)', icon: Activity },
    { id: 'dedup', label: '3. FastCDC Deduplication', icon: Layers },
    { id: 'erasure', label: '4. Reed-Solomon Erasure', icon: Cpu },
  ]

  const liveTelemetry = [
    { label: 'Cluster State', val: 'HEALTHY (Quorum 100%)', highlight: true },
    { label: 'Virtual Nodes', val: '150 per host (900 total)' },
    { label: 'Consistency', val: 'Dynamo N=3, W=2, R=2' },
    { label: 'FastCDC Window', val: '16 KB - 256 KB (Gear Hash)' },
    { label: 'CAS Hash Algo', val: 'SHA-256 (64-byte hex digest)' },
    { label: 'Durability', val: 'Sync WAL + Fsync flush' },
  ]

  const liveEventLogs = [
    { time: '14:22:01', text: 'SigV4 validated: PUT /dataset.parquet (1.8 GB)' },
    { time: '14:22:02', text: 'FastCDC chunking: 112 variable shards created' },
    { time: '14:22:03', text: 'SHA-256 CAS: 18 chunks deduplicated (0 disk I/O)' },
    { time: '14:22:04', text: 'Consistent ring: 94 chunks routed to primary nodes' },
    { time: '14:22:05', text: 'Quorum fan-out: W=2/2 ACKs confirmed via mTLS' },
    { time: '14:22:06', text: 'Manifest committed to WAL · 200 OK returned' },
  ]

  return (
    <div className="w-full space-y-8 font-sans">
      {/* 1. Top Header */}
      <div className="text-center space-y-3 max-w-3xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>Interactive Storage Simulator</span>
        </div>
        <h1 className="text-3xl sm:text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          Inspect Distributed Mechanics in Real Time
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          Switch between interactive experiments to test consistent hashing, node death self-healing, content-defined deduplication, and Reed-Solomon polynomial reconstruction.
        </p>
      </div>

      {/* 2. Sub-Tab Switcher */}
      <div className="flex items-center justify-center">
        <div className="flex flex-wrap items-center gap-1.5 p-1 rounded-lg bg-[#19191C] border border-[#26262B] font-mono text-xs">
          {experiments.map((exp) => {
            const isSelected = activeExperiment === exp.id
            const Icon = exp.icon

            return (
              <button
                key={exp.id}
                onClick={() => setActiveExperiment(exp.id as any)}
                className={`flex items-center gap-2 px-3.5 py-2 rounded text-xs transition-all cursor-pointer ${isSelected
                    ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold shadow-xs'
                    : 'text-[#8B8B8F] hover:text-[#EDE8DF] hover:bg-[#222226]'
                  }`}
              >
                <Icon className="w-3.5 h-3.5" />
                <span>{exp.label}</span>
              </button>
            )
          })}
        </div>
      </div>

      {/* 3. Main Split View: Left (3D Canvas - 70%) & Right (Mono Telemetry HUD - 30%) */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* 3D Canvas Box */}
        <div className="lg:col-span-8 w-full">
          <AnimatePresence mode="wait">
            <motion.div
              key={activeExperiment}
              initial={{ opacity: 0, y: 6 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -6 }}
              transition={{ duration: 0.2 }}
              className="w-full"
            >
              {activeExperiment === 'pipeline' && <VaultRing3D />}
              {activeExperiment === 'failure' && <FailureRecovery3D />}
              {activeExperiment === 'dedup' && <DedupCanvas />}
              {activeExperiment === 'erasure' && <ErasureCoding3D />}
            </motion.div>
          </AnimatePresence>
        </div>

        {/* Right Mono Telemetry HUD */}
        <div className="lg:col-span-4 w-full space-y-4 font-mono text-xs">
          {/* Active Status Panel */}
          <div className="p-4 rounded-xl border border-[#26262B] bg-[#19191C] space-y-3">
            <div className="flex items-center justify-between border-b border-[#26262B] pb-2">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-[#A6A9AE]" />
                <span className="font-bold text-[#EDE8DF]">CLUSTER TELEMETRY</span>
              </div>
              <span className="text-[10px] text-[#EDE8DF] font-bold px-1.5 py-0.5 rounded bg-[#000000] border border-[#26262B]">
                6 NODES UP
              </span>
            </div>

            <div className="space-y-2 text-[11px]">
              {liveTelemetry.map((item, i) => (
                <div key={i} className="flex items-center justify-between py-1 border-b border-[#26262B]/50 last:border-0">
                  <span className="text-[#8B8B8F]">{item.label}</span>
                  <span className={item.highlight ? 'text-[#EDE8DF] font-bold' : 'text-[#EDE8DF]'}>
                    {item.val}
                  </span>
                </div>
              ))}
            </div>
          </div>

          {/* Live Node Event Journal */}
          <div className="p-4 rounded-xl border border-[#26262B] bg-[#19191C] space-y-3">
            <div className="flex items-center justify-between border-b border-[#26262B] pb-2">
              <div className="flex items-center gap-2">
                <Clock className="w-4 h-4 text-[#8B8B8F]" />
                <span className="font-bold text-[#EDE8DF]">EVENT JOURNAL</span>
              </div>
              <span className="text-[10px] text-[#8B8B8F]">WAL Log</span>
            </div>

            <div className="space-y-2 text-[10px]">
              {liveEventLogs.map((log, i) => (
                <div key={i} className="flex items-start gap-2 text-[#8B8B8F]">
                  <span className="text-[#A6A9AE] shrink-0 font-bold">{log.time}</span>
                  <span className="text-[#EDE8DF]/90 leading-tight">{log.text}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
