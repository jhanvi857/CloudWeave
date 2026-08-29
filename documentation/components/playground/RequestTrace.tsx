'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { Activity, CheckCircle, Clock, Layers, ShieldCheck, HardDrive, Server } from 'lucide-react'

export interface TraceEvent {
  id: string
  offsetMs: number
  label: string
  component: string
  detail: string
  icon: any
}

const traceEvents: TraceEvent[] = [
  { id: '1', offsetMs: 0, label: 'API request received', component: 'S3 Compatibility Layer', detail: 'Received HTTP PUT /bucket/video.mp4 with Content-Length: 62914560', icon: Activity },
  { id: '2', offsetMs: 5, label: 'Metadata created', component: 'Metadata Engine', detail: 'Initialized pending manifest in memory & registered in InFlightRegistry', icon: HardDrive },
  { id: '3', offsetMs: 11, label: 'FastCDC started', component: 'FastCDC Chunker', detail: 'Rolling Gear Hash chunking initialized with target size 16MB', icon: Layers },
  { id: '4', offsetMs: 18, label: 'Chunk generated', component: 'Streaming Pipeline', detail: 'Generated 4 variable-sized chunks [12MB, 18MB, 9MB, 21MB]', icon: Layers },
  { id: '5', offsetMs: 19, label: 'SHA-256 calculated', component: 'CAS Crypto', detail: 'Computed hexadecimal SHA-256 content addresses for chunk verification', icon: HardDrive },
  { id: '6', offsetMs: 21, label: 'Placement selected', component: 'Consistent Hash Ring', detail: 'Mapped hashes against 150 virtual nodes to determine N=3 primary & replicas', icon: Server },
  { id: '7', offsetMs: 25, label: 'Replica written', component: 'DiskStore Transport', detail: 'Parallel worker pool streamed chunk data across persistent keep-alive connections', icon: Server },
  { id: '8', offsetMs: 31, label: 'Quorum achieved', component: 'Quorum Coordinator', detail: 'Received W=2 acknowledgments from Node 2 and Node 3', icon: ShieldCheck },
  { id: '9', offsetMs: 33, label: 'WAL committed', component: 'Write-Ahead Log', detail: 'Appended manifest entry to metadata.wal with synchronous fsync() disk flush', icon: HardDrive },
  { id: '10', offsetMs: 35, label: 'Response returned', component: 'HTTP S3 API', detail: 'Returned HTTP 201 Created with ETag: "cw-8f3a2c41-v1"', icon: CheckCircle },
]

export function RequestTrace() {
  const [selectedEvent, setSelectedEvent] = useState<TraceEvent>(traceEvents[0])

  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Clock className="w-5 h-5 text-[#0077B6]" />
            <h3 className="font-bold text-base text-[#03045E]">End-to-End Request Trace (35ms Total)</h3>
          </div>
          <p className="text-xs text-[#64748B] mt-0.5">
            Click any execution phase to inspect sub-millisecond execution details.
          </p>
        </div>
        <div className="text-xs font-mono font-bold px-3 py-1 rounded-lg bg-[#E8F8F0] text-[#22A06B] border border-[#A8E6CF]">
          Latency: 35 ms
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Left: Vertical Timeline */}
        <div className="lg:col-span-7 space-y-2 font-mono text-xs">
          {traceEvents.map((evt) => {
            const isSelected = selectedEvent.id === evt.id
            const Icon = evt.icon

            return (
              <button
                key={evt.id}
                onClick={() => setSelectedEvent(evt)}
                className={`w-full flex items-center justify-between p-2.5 rounded-lg border text-left transition-all ${
                  isSelected
                    ? 'border-[#00B4D8] bg-[#CAF0F8] text-[#03045E] shadow-2xs font-semibold'
                    : 'border-[#E2E8F0] bg-white text-[#475569] hover:bg-[#F8FAFC]'
                }`}
              >
                <div className="flex items-center gap-3">
                  <span className="w-12 text-[11px] font-bold text-[#0077B6] shrink-0">
                    +{String(evt.offsetMs).padStart(2, '0')} ms
                  </span>
                  <div className="flex items-center gap-2">
                    <Icon className={`w-3.5 h-3.5 ${isSelected ? 'text-[#0077B6]' : 'text-[#64748B]'}`} />
                    <span className="text-xs">{evt.label}</span>
                  </div>
                </div>

                <span className="text-[10px] text-[#64748B] hidden sm:block">
                  {evt.component}
                </span>
              </button>
            )
          })}
        </div>

        {/* Right: Selected Event Inspector Card */}
        <div className="lg:col-span-5 flex flex-col justify-between p-5 rounded-xl border border-[#CBD5E1] bg-[#F8FAFC]">
          <div>
            <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-3">
              <span className="text-[10px] font-mono font-bold uppercase text-[#0077B6]">
                Phase Inspector
              </span>
              <span className="text-xs font-mono font-bold px-2 py-0.5 rounded bg-white border border-[#CBD5E1] text-[#03045E]">
                +{selectedEvent.offsetMs} ms
              </span>
            </div>

            <div className="mt-4 space-y-3 font-mono">
              <div>
                <span className="text-[11px] text-[#64748B] block">Event Name:</span>
                <span className="text-sm font-bold text-[#03045E]">{selectedEvent.label}</span>
              </div>

              <div>
                <span className="text-[11px] text-[#64748B] block">Subsystem Component:</span>
                <span className="text-xs font-semibold text-[#0077B6]">{selectedEvent.component}</span>
              </div>

              <div>
                <span className="text-[11px] text-[#64748B] block">Execution Detail:</span>
                <p className="text-xs text-[#475569] leading-relaxed mt-1 p-3 rounded-lg bg-white border border-[#E2E8F0]">
                  {selectedEvent.detail}
                </p>
              </div>
            </div>
          </div>

          <div className="mt-4 pt-3 border-t border-[#E2E8F0] text-[10px] font-mono text-[#64748B]">
            Zero CGO overhead • Synchronous WAL fsync guaranteed
          </div>
        </div>
      </div>
    </div>
  )
}
