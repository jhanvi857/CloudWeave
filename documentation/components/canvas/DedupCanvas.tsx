'use client'

import React, { useState } from 'react'
import { Layers, RefreshCw, ArrowRight } from 'lucide-react'

export function DedupCanvas() {
  const [hasUploadedV2, setHasUploadedV2] = useState<boolean>(false)

  const chunksV1 = [
    { name: 'Chunk A', hash: '8f3a...91c2', size: '16 KB', status: 'stored' },
    { name: 'Chunk B', hash: '17bd...82a1', size: '64 KB', status: 'stored' },
    { name: 'Chunk C', hash: 'a92c...11ef', size: '32 KB', status: 'stored' },
    { name: 'Chunk D', hash: '7b3f...91c8', size: '48 KB', status: 'stored' },
  ]

  const chunksV2 = [
    { name: 'Chunk A', hash: '8f3a...91c2', size: '16 KB', status: 'reused' },
    { name: 'Chunk B\'', hash: '3e4f...1290', size: '64 KB', status: 'new' }, // Only 1 byte changed in B
    { name: 'Chunk C', hash: 'a92c...11ef', size: '32 KB', status: 'reused' },
    { name: 'Chunk D', hash: '7b3f...91c8', size: '48 KB', status: 'reused' },
  ]

  return (
    <div className="w-full flex flex-col bg-[#19191C] border border-[#26262B] rounded-xl overflow-hidden font-sans select-none">
      {/* Header */}
      <div className="p-4 border-b border-[#26262B] flex flex-wrap items-center justify-between gap-3 font-mono text-xs bg-[#000000]">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-[#A6A9AE]" />
          <span className="font-bold text-[#EDE8DF]">FASTCDC CONTENT-DEFINED DEDUPLICATION</span>
        </div>

        <div>
          {!hasUploadedV2 ? (
            <button
              onClick={() => setHasUploadedV2(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-semibold hover:bg-[#912735] transition-colors text-xs cursor-pointer shadow-xs"
            >
              <span>Upload Modified File (1-Byte Edit)</span>
              <ArrowRight className="w-3.5 h-3.5 stroke-[2.5]" />
            </button>
          ) : (
            <button
              onClick={() => setHasUploadedV2(false)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded border border-[#26262B] text-[#8B8B8F] hover:text-[#EDE8DF] hover:border-[#34343A] transition-colors text-xs cursor-pointer"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              <span>Reset Experiment</span>
            </button>
          )}
        </div>
      </div>

      {/* Comparison Grid */}
      <div className="p-6 space-y-6 font-mono text-xs bg-[#000000]">
        {/* Upload 1 */}
        <div className="space-y-2">
          <div className="flex items-center justify-between text-[#8B8B8F] text-[11px]">
            <span className="font-bold text-[#EDE8DF]">Original Object: backup_v1.tar (160 KB)</span>
            <span>4 FastCDC Chunks</span>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {chunksV1.map((c) => (
              <div key={c.name} className="p-3 rounded-lg border border-[#26262B] bg-[#19191C] text-center space-y-1">
                <div className="font-bold text-[#EDE8DF]">{c.name}</div>
                <div className="text-[10px] text-[#8B8B8F]">{c.hash}</div>
                <div className="text-[10px] text-[#EDE8DF] font-semibold pt-1 border-t border-[#26262B]">
                  Stored to Disk ({c.size})
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Upload 2 */}
        {hasUploadedV2 && (
          <div className="space-y-2 pt-2 border-t border-[#26262B]">
            <div className="flex items-center justify-between text-[#8B8B8F] text-[11px]">
              <span className="font-bold text-[#EDE8DF]">
                Modified Object: backup_v2.tar (160 KB) · 1 Byte Changed in Chunk B
              </span>
              <span className="text-[#EDE8DF] font-semibold">75% Deduplication Savings</span>
            </div>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {chunksV2.map((c) => (
                <div
                  key={c.name}
                  className={`p-3 rounded-lg border text-center space-y-1 ${c.status === 'reused'
                      ? 'border-[#26262B] bg-[#19191C]'
                      : 'border-[#7A1F2B] bg-[#19191C]'
                    }`}
                >
                  <div className="font-bold text-[#EDE8DF]">{c.name}</div>
                  <div className="text-[10px] text-[#8B8B8F]">{c.hash}</div>
                  <div
                    className={`text-[10px] font-semibold pt-1 border-t border-[#26262B] ${c.status === 'reused' ? 'text-[#EDE8DF]' : 'text-[#7A1F2B]'
                      }`}
                  >
                    {c.status === 'reused' ? 'Reused (0 Disk I/O)' : 'New Chunk Written (64 KB)'}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Bottom Proof Strip */}
      <div className="p-4 border-t border-[#26262B] bg-[#19191C] font-mono text-xs">
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Logical Transferred:</span>
            <span className="text-[#EDE8DF] font-semibold">{hasUploadedV2 ? '320 KB (2 Objects)' : '160 KB (1 Object)'}</span>
          </div>
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Physical Disk Used:</span>
            <span className="text-[#EDE8DF] font-bold">{hasUploadedV2 ? '224 KB (vs 320 KB)' : '160 KB'}</span>
          </div>
          <div className="p-2.5 rounded bg-[#000000] border border-[#26262B]">
            <span className="text-[10px] text-[#8B8B8F] block">Effective Speedup:</span>
            <span className="text-[#EDE8DF] font-bold">{hasUploadedV2 ? '485.2 MB/s (CAS Hit)' : '151.6 MB/s'}</span>
          </div>
        </div>
      </div>
    </div>
  )
}
