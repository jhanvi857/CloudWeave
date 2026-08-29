'use client'

import React from 'react'
import { motion } from 'framer-motion'
import { Server, Layers, ShieldCheck, ArrowRight } from 'lucide-react'

export function ReplicationVisualizer() {
  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-4">
        <div>
          <h3 className="font-bold text-base text-[#03045E]">Parallel N=3 Replication Fan-Out</h3>
          <p className="text-xs text-[#64748B]">
            Primary storage node streams replicas to distinct physical peers with zero head-of-line blocking.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="flex items-center gap-1.5 text-[11px] font-mono text-[#0077B6] bg-[#CAF0F8] px-2.5 py-1 rounded-md font-bold">
            <span className="h-2 w-2 rounded-full bg-[#00B4D8] animate-ping" />
            Bounded 8-Worker Pool
          </span>
        </div>
      </div>

      {/* SVG Multi-Branch Diagram */}
      <div className="relative w-full max-w-2xl mx-auto py-4">
        <div className="flex flex-col md:flex-row items-center justify-between gap-6">
          {/* Source Chunk */}
          <div className="flex flex-col items-center p-4 rounded-xl border-2 border-[#00B4D8] bg-[#CAF0F8] text-center w-40 shrink-0 shadow-xs">
            <Layers className="w-6 h-6 text-[#0077B6] mb-1" />
            <span className="font-mono text-xs font-bold text-[#03045E]">Chunk A</span>
            <span className="text-[10px] font-mono text-[#64748B]">12 MB (SHA-256)</span>
            <span className="mt-1.5 text-[9px] font-mono font-bold uppercase px-1.5 py-0.5 rounded bg-[#0077B6] text-white">
              Primary Source
            </span>
          </div>

          {/* Branching SVG Lines */}
          <div className="hidden md:flex flex-col justify-around h-48 w-24 relative">
            <svg className="w-full h-full" viewBox="0 0 100 200" fill="none">
              {/* Branch 1 to Node 1 */}
              <path
                d="M 0 100 C 50 100, 50 30, 100 30"
                stroke="#00B4D8"
                strokeWidth="3"
                strokeDasharray="4 4"
                className="animate-pulse"
              />
              {/* Branch 2 to Node 2 */}
              <path
                d="M 0 100 L 100 100"
                stroke="#00B4D8"
                strokeWidth="3"
              />
              {/* Branch 3 to Node 3 */}
              <path
                d="M 0 100 C 50 100, 50 170, 100 170"
                stroke="#90E0EF"
                strokeWidth="2"
              />
            </svg>
          </div>

          {/* Destination Nodes */}
          <div className="flex flex-col gap-3 w-full md:w-64">
            {/* Node 1 */}
            <div className="flex items-center justify-between p-3 rounded-xl border border-[#A8E6CF] bg-[#E8F8F0] shadow-2xs">
              <div className="flex items-center gap-2.5">
                <Server className="w-4 h-4 text-[#22A06B]" />
                <div>
                  <span className="text-xs font-mono font-bold text-[#03045E]">Node 1 (10.0.0.1)</span>
                  <p className="text-[10px] font-mono text-[#22A06B]">Replica 1 • Primary Sync</p>
                </div>
              </div>
              <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded bg-white text-[#22A06B] border border-[#A8E6CF]">
                COMMITTED
              </span>
            </div>

            {/* Node 2 */}
            <div className="flex items-center justify-between p-3 rounded-xl border border-[#A8E6CF] bg-[#E8F8F0] shadow-2xs">
              <div className="flex items-center gap-2.5">
                <Server className="w-4 h-4 text-[#22A06B]" />
                <div>
                  <span className="text-xs font-mono font-bold text-[#03045E]">Node 2 (10.0.0.2)</span>
                  <p className="text-[10px] font-mono text-[#22A06B]">Replica 2 • Quorum ACK</p>
                </div>
              </div>
              <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded bg-white text-[#22A06B] border border-[#A8E6CF]">
                COMMITTED
              </span>
            </div>

            {/* Node 3 */}
            <div className="flex items-center justify-between p-3 rounded-xl border border-[#CBD5E1] bg-white shadow-2xs">
              <div className="flex items-center gap-2.5">
                <Server className="w-4 h-4 text-[#64748B]" />
                <div>
                  <span className="text-xs font-mono font-bold text-[#03045E]">Node 3 (10.0.0.3)</span>
                  <p className="text-[10px] font-mono text-[#64748B]">Replica 3 • Async Sync</p>
                </div>
              </div>
              <span className="text-[10px] font-mono font-bold px-2 py-0.5 rounded bg-[#F1F5F9] text-[#64748B]">
                STREAMING
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
