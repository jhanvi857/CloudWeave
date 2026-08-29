'use client'

import React from 'react'
import { motion } from 'framer-motion'
import { FileVideo, Server, ArrowDown, Hash, Layers, CheckCircle } from 'lucide-react'

export interface ChunkInfo {
  id: string
  name: string
  sizeMb: number
  hash: string
  primaryNode: string
  replicas: string[]
  status: 'idle' | 'chunking' | 'hashed' | 'placed' | 'replicated' | 'committed'
}

export const sampleChunks: ChunkInfo[] = [
  {
    id: 'chunk-a',
    name: 'Chunk A',
    sizeMb: 12,
    hash: '8f3a2c419be091c2',
    primaryNode: 'Node 2',
    replicas: ['Node 3', 'Node 4'],
    status: 'committed',
  },
  {
    id: 'chunk-b',
    name: 'Chunk B',
    sizeMb: 18,
    hash: '17bd9e30fac882a1',
    primaryNode: 'Node 4',
    replicas: ['Node 5', 'Node 1'],
    status: 'committed',
  },
  {
    id: 'chunk-c',
    name: 'Chunk C',
    sizeMb: 9,
    hash: 'a92c7104bd1111ef',
    primaryNode: 'Node 1',
    replicas: ['Node 2', 'Node 3'],
    status: 'committed',
  },
  {
    id: 'chunk-d',
    name: 'Chunk D',
    sizeMb: 21,
    hash: '7b3f5481d4a091c8',
    primaryNode: 'Node 3',
    replicas: ['Node 4', 'Node 5'],
    status: 'committed',
  },
]

interface UploadChunkingVisualizerProps {
  step: 'upload' | 'chunking' | 'distribution' | 'replication' | 'quorum' | 'complete'
  selectedChunk: string | null
  onSelectChunk: (id: string) => void
}

export function UploadChunkingVisualizer({
  step,
  selectedChunk,
  onSelectChunk,
}: UploadChunkingVisualizerProps) {
  const isUploaded = step !== 'upload'
  const isChunked = step !== 'upload'
  const isHashed = step !== 'upload' && step !== 'chunking'

  return (
    <div className="flex flex-col items-center justify-center p-6 space-y-6 w-full max-w-4xl mx-auto">
      {/* 1. Client S3 Request */}
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        className="w-full flex items-center justify-between p-4 rounded-xl border border-[#CBD5E1] bg-white shadow-xs"
      >
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-[#CAF0F8] text-[#0077B6]">
            <FileVideo className="w-5 h-5" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold text-sm text-[#03045E]">video.mp4</span>
              <span className="rounded bg-[#F1F5F9] px-2 py-0.5 text-[10px] font-mono text-[#64748B]">
                60.0 MB total
              </span>
            </div>
            <p className="text-xs font-mono text-[#64748B]">PUT /bucket/media/video.mp4</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <span className="inline-block h-2 w-2 rounded-full bg-[#00B4D8] animate-ping" />
          <span className="text-xs font-mono font-semibold text-[#0077B6]">
            {step === 'upload' ? 'Streaming to S3 API...' : 'FastCDC Rolling Stream Active'}
          </span>
        </div>
      </motion.div>

      {/* Downward Data Stream Line */}
      <div className="flex flex-col items-center">
        <div className="h-6 w-0.5 bg-[#90E0EF] relative overflow-hidden">
          <motion.div
            className="absolute top-0 w-full bg-[#00B4D8] h-3"
            animate={{ y: [0, 24] }}
            transition={{ repeat: Infinity, duration: 1.2, ease: 'linear' }}
          />
        </div>
        <div className="flex items-center gap-1.5 rounded-full border border-[#90E0EF] bg-[#CAF0F8] px-3 py-1 text-[11px] font-mono font-semibold text-[#0077B6]">
          <Layers className="w-3 h-3 text-[#00B4D8]" />
          <span>FastCDC Content-Defined Chunker (Min: 8MB, Avg: 15MB, Max: 24MB)</span>
        </div>
        <div className="h-6 w-0.5 bg-[#90E0EF]" />
      </div>

      {/* 2. FastCDC Chunks Splitting Grid */}
      <div className="w-full">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-mono font-bold uppercase text-[#64748B]">
            Variable-Sized Content-Defined Chunks
          </span>
          <span className="text-xs font-mono text-[#0077B6]">
            {isHashed ? '4 Chunks Hashed & Verified' : 'Rolling Gear Hash in progress'}
          </span>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          {sampleChunks.map((chunk, idx) => {
            const isSelected = selectedChunk === chunk.id
            return (
              <motion.div
                key={chunk.id}
                initial={{ opacity: 0, scale: 0.95 }}
                animate={{
                  opacity: 1,
                  scale: isSelected ? 1.02 : 1,
                  y: isChunked ? 0 : -10,
                }}
                transition={{ delay: idx * 0.1, duration: 0.3 }}
                onClick={() => onSelectChunk(chunk.id)}
                className={`p-3.5 rounded-xl border cursor-pointer transition-all ${
                  isSelected
                    ? 'border-[#00B4D8] bg-[#CAF0F8] shadow-md ring-2 ring-[#00B4D8]/30'
                    : 'border-[#CBD5E1] bg-white hover:border-[#90E0EF] hover:bg-[#F8FAFC]'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-mono font-bold text-[#03045E]">
                    {chunk.name}
                  </span>
                  <span className="text-[11px] font-mono font-semibold px-2 py-0.5 rounded bg-white border border-[#E2E8F0] text-[#0077B6]">
                    {chunk.sizeMb} MB
                  </span>
                </div>

                {/* Boundary bar indicator */}
                <div className="mt-2.5 w-full bg-[#F1F5F9] rounded-full h-1.5 overflow-hidden">
                  <div
                    className="bg-[#00B4D8] h-full rounded-full"
                    style={{ width: `${(chunk.sizeMb / 24) * 100}%` }}
                  />
                </div>

                {/* SHA-256 Hash Display */}
                <div className="mt-3 pt-2.5 border-t border-[#E2E8F0] space-y-1">
                  <div className="flex items-center gap-1 text-[10px] font-mono text-[#64748B]">
                    <Hash className="w-3 h-3 text-[#0077B6]" />
                    <span>SHA-256 Digest</span>
                  </div>
                  <div className="text-[11px] font-mono font-bold text-[#03045E] truncate bg-white px-2 py-1 rounded border border-[#E2E8F0]">
                    {chunk.hash}
                  </div>
                </div>

                {/* Primary & Replicas */}
                <div className="mt-2 flex items-center justify-between text-[10px] font-mono text-[#64748B]">
                  <span>Primary: <b className="text-[#03045E]">{chunk.primaryNode}</b></span>
                  <span>N=3</span>
                </div>
              </motion.div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
