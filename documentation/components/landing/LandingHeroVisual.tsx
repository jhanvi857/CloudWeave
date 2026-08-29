'use client'

import React from 'react'
import { FileVideo, Check } from 'lucide-react'

export function LandingHeroVisual() {
  return (
    <div className="w-full max-w-lg lg:max-w-none">
      <div className="rounded-2xl border border-[#E2E8F0] bg-white overflow-hidden shadow-xl relative select-none">
        {/* Terminal Header with 3 macOS dots */}
        <div className="flex items-center px-4 py-3 border-b border-[#E2E8F0] bg-[#F8FAFC] relative">
          <div className="flex space-x-2 absolute left-4">
            <div className="w-3 h-3 rounded-full bg-[#E5484D]" />
            <div className="w-3 h-3 rounded-full bg-[#F59E0B]" />
            <div className="w-3 h-3 rounded-full bg-[#22A06B]" />
          </div>
          <div className="w-full text-center">
            <span className="text-xs font-mono text-[#64748B]">cloudweave_pipeline - visualizer</span>
          </div>
        </div>

        {/* Visual Architecture Flow Body */}
        <div className="p-6 font-mono text-xs flex flex-col items-center space-y-3.5">
          {/* 1. Client Object */}
          <div className="flex items-center gap-3 px-4 py-2.5 rounded-xl border border-[#CBD5E1] bg-white shadow-xs">
            <FileVideo className="w-4 h-4 text-[#0077B6]" />
            <div>
              <span className="font-bold text-[#03045E] block text-xs">video.mp4</span>
              <span className="text-[10px] text-[#64748B]">2.4 GB Byte Stream</span>
            </div>
          </div>

          {/* Downward connecting line */}
          <div className="h-4 w-px bg-[#CBD5E1]" />

          {/* 2. FastCDC Chamber */}
          <div className="px-4 py-2 rounded-xl border border-[#0077B6]/30 bg-[#CAF0F8] text-center shadow-xs">
            <span className="font-bold text-[#0077B6] text-[11px] block">FASTCDC + SHA-256</span>
            <span className="text-[9px] text-[#0077B6]/80 font-sans">Content-Defined CAS Chunking</span>
          </div>

          {/* Downward branching line */}
          <div className="h-4 w-px bg-[#CBD5E1]" />

          {/* 3. 4 Chunks Row */}
          <div className="grid grid-cols-4 gap-2 w-full">
            {[
              { name: 'Chunk A', size: '16 MB', hash: '8f3a...' },
              { name: 'Chunk B', size: '16 MB', hash: '17bd...' },
              { name: 'Chunk C', size: '16 MB', hash: 'a92c...' },
              { name: 'Chunk N', size: '16 MB', hash: '7b3f...' },
            ].map((chunk) => (
              <div
                key={chunk.name}
                className="p-2 rounded-lg border border-[#CBD5E1] bg-[#CAF0F8]/40 text-center shadow-2xs"
              >
                <span className="font-bold text-[#03045E] block text-[10px]">{chunk.name}</span>
                <span className="text-[9px] text-[#64748B] block">{chunk.size}</span>
                <span className="text-[8px] text-[#0077B6] font-mono mt-0.5 block">{chunk.hash}</span>
              </div>
            ))}
          </div>

          {/* Downward connecting lines to nodes */}
          <div className="h-4 w-px bg-[#CBD5E1]" />

          {/* 4. 5 Cluster Storage Nodes */}
          <div className="flex items-center justify-between gap-2 w-full pt-0.5">
            {['Node 1', 'Node 2', 'Node 3', 'Node 4', 'Node 5'].map((nodeName, idx) => {
              const isPrimary = idx === 1
              const isReplica = idx === 2 || idx === 3

              return (
                <div
                  key={nodeName}
                  className={`flex-1 py-2 px-1 rounded-lg border text-center transition-all ${
                    isPrimary
                      ? 'border-[#0077B6] bg-[#CAF0F8] text-[#0077B6] shadow-xs font-bold'
                      : isReplica
                      ? 'border-[#22A06B]/40 bg-[#E8F8F0] text-[#22A06B] font-semibold'
                      : 'border-[#E2E8F0] bg-[#F8FAFC] text-[#94A3B8]'
                  }`}
                >
                  <span className="font-bold text-[10px] block">N{idx + 1}</span>
                  <span
                    className={`text-[8px] block font-sans ${
                      isPrimary ? 'text-[#0077B6]' : isReplica ? 'text-[#22A06B]' : 'text-[#94A3B8]'
                    }`}
                  >
                    {isPrimary ? 'Primary' : isReplica ? 'Replica' : 'Peer'}
                  </span>
                </div>
              )
            })}
          </div>

          {/* Footer Tag */}
          <div className="w-full flex items-center justify-between border-t border-[#E2E8F0] pt-3 mt-3 text-[10px] text-[#64748B] font-medium">
            <span>N=3 Mesh Quorum</span>
            <span className="text-[#22A06B] flex items-center gap-1 font-bold">
              <Check className="w-3.5 h-3.5" />
              <span>W=2 Acknowledged (fsync)</span>
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
