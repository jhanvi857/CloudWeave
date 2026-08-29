'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Layers, HardDrive, Sparkles, CheckCircle2, ArrowRight, RefreshCw, FileText } from 'lucide-react'

export function DeduplicationPlayground() {
  const [uploadCount, setUploadCount] = useState<number>(1)
  const [hasEdit, setHasEdit] = useState<boolean>(false)

  const handleSecondUpload = (edit: boolean) => {
    setHasEdit(edit)
    setUploadCount(2)
  }

  const handleReset = () => {
    setUploadCount(1)
    setHasEdit(false)
  }

  // Chunks calculation
  // Upload 1: 4 chunks (A, B, C, D) -> all stored
  // Upload 2 identical: 4 chunks reused, 0 new
  // Upload 2 with 1-byte edit in B: Chunks A, C, D reused, only chunk B' newly stored!
  const reusedChunks = uploadCount === 1 ? 0 : hasEdit ? 3 : 4
  const newChunks = uploadCount === 1 ? 4 : hasEdit ? 1 : 0
  const logicalMb = uploadCount === 1 ? 60 : 120
  const physicalMb = uploadCount === 1 ? 60 : hasEdit ? 78 : 60
  const savingsPct = Math.round(((logicalMb - physicalMb) / logicalMb) * 100)

  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#E2E8F0] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Layers className="w-5 h-5 text-[#0077B6]" />
            <h3 className="font-bold text-base text-[#03045E]">FastCDC & SHA-256 Deduplication Lab</h3>
          </div>
          <p className="text-xs text-[#64748B] mt-0.5">
            Content-addressable storage (CAS) automatically eliminates redundant chunk storage across uploads.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {uploadCount === 1 ? (
            <div className="flex gap-2">
              <button
                onClick={() => handleSecondUpload(false)}
                className="flex items-center gap-1.5 rounded-lg bg-[#0077B6] px-3.5 py-1.5 text-xs font-bold text-white shadow-xs hover:bg-[#03045E] transition-all"
              >
                <span>Upload Duplicate File</span>
              </button>
              <button
                onClick={() => handleSecondUpload(true)}
                className="flex items-center gap-1.5 rounded-lg border border-[#00B4D8] bg-[#CAF0F8] px-3.5 py-1.5 text-xs font-bold text-[#0077B6] hover:bg-[#90E0EF] transition-all"
              >
                <span>Upload Modified (1-Byte Edit)</span>
              </button>
            </div>
          ) : (
            <button
              onClick={handleReset}
              className="flex items-center gap-1.5 rounded-lg border border-[#CBD5E1] bg-white px-3 py-1.5 text-xs font-semibold text-[#03045E] hover:bg-[#F1F5F9] transition-all"
            >
              <RefreshCw className="w-3.5 h-3.5 text-[#0077B6]" />
              <span>Reset Deduplication Lab</span>
            </button>
          )}
        </div>
      </div>

      {/* Storage Savings Meter */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 bg-[#F8FAFC] p-4 rounded-xl border border-[#E2E8F0] font-mono text-xs">
        <div>
          <span className="text-[#64748B] text-[11px] block">Total Logical Ingested</span>
          <span className="text-lg font-bold text-[#03045E]">{logicalMb} MB</span>
          <span className="text-[10px] text-[#64748B] block mt-0.5">2 files uploaded by client</span>
        </div>

        <div>
          <span className="text-[#64748B] text-[11px] block">Actual Physical Disk Used</span>
          <span className="text-lg font-bold text-[#0077B6]">{physicalMb} MB</span>
          <span className="text-[10px] text-[#22A06B] block mt-0.5 font-bold">
            {savingsPct > 0 ? `${savingsPct}% physical disk savings` : 'Initial Baseline'}
          </span>
        </div>

        <div>
          <span className="text-[#64748B] text-[11px] block">Deduplication Ratio</span>
          <span className="text-lg font-bold text-[#22A06B]">
            {uploadCount === 1 ? '1.0x' : hasEdit ? '1.54x' : '2.0x'}
          </span>
          <span className="text-[10px] text-[#64748B] block mt-0.5">
            {reusedChunks} chunks reused, {newChunks} new written
          </span>
        </div>
      </div>

      {/* Visual Bars Comparison */}
      <div className="space-y-3 font-mono text-xs">
        <div>
          <div className="flex justify-between text-[11px] mb-1 text-[#64748B]">
            <span>Logical Data Size (Client-side)</span>
            <span className="font-bold text-[#03045E]">{logicalMb} MB</span>
          </div>
          <div className="w-full bg-[#E2E8F0] h-3 rounded-full overflow-hidden">
            <div className="bg-[#64748B] h-full rounded-full" style={{ width: '100%' }} />
          </div>
        </div>

        <div>
          <div className="flex justify-between text-[11px] mb-1 text-[#64748B]">
            <span>Physical Storage Footprint (Cluster DiskStore)</span>
            <span className="font-bold text-[#0077B6]">{physicalMb} MB</span>
          </div>
          <div className="w-full bg-[#E2E8F0] h-3 rounded-full overflow-hidden">
            <motion.div
              className="bg-[#00B4D8] h-full rounded-full"
              initial={{ width: '100%' }}
              animate={{ width: `${(physicalMb / logicalMb) * 100}%` }}
              transition={{ duration: 0.5 }}
            />
          </div>
        </div>
      </div>

      {/* Chunk Breakdown Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-3 pt-2">
        {['Chunk A (12MB)', 'Chunk B (18MB)', 'Chunk C (9MB)', 'Chunk D (21MB)'].map((name, idx) => {
          const isModifiedChunk = hasEdit && idx === 1
          const isReused = uploadCount === 2 && !isModifiedChunk

          return (
            <div
              key={name}
              className={`p-3 rounded-xl border text-xs font-mono transition-all ${
                isReused
                  ? 'border-[#A8E6CF] bg-[#E8F8F0]'
                  : isModifiedChunk
                  ? 'border-[#FDE68A] bg-[#FFF8E6]'
                  : 'border-[#CBD5E1] bg-white'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-bold text-[#03045E]">{name}</span>
                {isReused && <CheckCircle2 className="w-3.5 h-3.5 text-[#22A06B]" />}
              </div>
              <div className="mt-2 text-[10px] text-[#64748B]">
                {uploadCount === 1 ? (
                  <span>Initial Upload (CAS written)</span>
                ) : isReused ? (
                  <span className="text-[#22A06B] font-bold">100% Hash Match • Zero I/O</span>
                ) : (
                  <span className="text-[#D97706] font-bold">Boundary Changed • Stored</span>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
