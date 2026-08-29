'use client'

import React, { useState, useEffect, useRef } from 'react'
import { Terminal, Trash2, ArrowDown } from 'lucide-react'

export interface LogEntry {
  id: string
  time: string
  subsystem: 'S3' | 'FASTCDC' | 'RING' | 'REPLICATION' | 'QUORUM' | 'WAL' | 'MEMBERSHIP'
  level: 'INFO' | 'WARN' | 'DEBUG' | 'SUCCESS'
  message: string
}

const initialLogs: LogEntry[] = [
  { id: '1', time: '12:01:23.104', subsystem: 'S3', level: 'INFO', message: 'PUT /bucket/media/video.mp4 stream opened (size=60MB)' },
  { id: '2', time: '12:01:23.210', subsystem: 'FASTCDC', level: 'INFO', message: 'FastCDC streaming: generated 4 variable content chunks' },
  { id: '3', time: '12:01:23.220', subsystem: 'FASTCDC', level: 'SUCCESS', message: 'SHA-256 digests computed: 8f3a..., 17bd..., a92c..., 7b3f...' },
  { id: '4', time: '12:01:23.245', subsystem: 'RING', level: 'INFO', message: 'Consistent hash ring lookup: Chunk A -> Primary Node 2 (replicas: Node 3, Node 4)' },
  { id: '5', time: '12:01:23.280', subsystem: 'REPLICATION', level: 'INFO', message: 'Parallel transfer fanning out 4 chunks across 8-worker threadpool' },
  { id: '6', time: '12:01:23.310', subsystem: 'QUORUM', level: 'SUCCESS', message: 'Quorum W=2 achieved: Node 2 (ACK), Node 3 (ACK)' },
  { id: '7', time: '12:01:23.330', subsystem: 'WAL', level: 'SUCCESS', message: 'metadata.wal append-only sync fsync() complete. Manifest committed.' },
  { id: '8', time: '12:01:23.345', subsystem: 'S3', level: 'SUCCESS', message: 'HTTP 201 Created returned. ETag: "cw-8f3a2c41-v1"' },
]

interface LiveLogsProps {
  customLogs?: LogEntry[]
}

export function LiveLogs({ customLogs }: LiveLogsProps) {
  const [logs, setLogs] = useState<LogEntry[]>(customLogs || initialLogs)
  const logEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (customLogs) {
      setLogs(customLogs)
    }
  }, [customLogs])

  const clearLogs = () => {
    setLogs([])
  }

  const getBadgeStyle = (subsystem: string) => {
    switch (subsystem) {
      case 'S3':
        return 'bg-[#CAF0F8] text-[#0077B6] border-[#90E0EF]'
      case 'FASTCDC':
        return 'bg-[#E8F8F0] text-[#22A06B] border-[#A8E6CF]'
      case 'RING':
        return 'bg-[#F1F5F9] text-[#03045E] border-[#CBD5E1]'
      case 'QUORUM':
        return 'bg-[#CAF0F8] text-[#00B4D8] border-[#90E0EF]'
      case 'WAL':
        return 'bg-[#FFF8E6] text-[#D97706] border-[#FDE68A]'
      default:
        return 'bg-[#F1F5F9] text-[#64748B] border-[#E2E8F0]'
    }
  }

  return (
    <div className="rounded-xl border border-[#CBD5E1] bg-[#03045E] text-white shadow-md overflow-hidden font-mono text-xs">
      <div className="flex items-center justify-between px-4 py-2.5 bg-[#001D3D] border-b border-white/10">
        <div className="flex items-center gap-2">
          <Terminal className="w-4 h-4 text-[#00B4D8]" />
          <span className="font-bold text-white tracking-tight">CloudWeave Live Node Logs</span>
          <span className="text-[10px] text-[#90E0EF] bg-white/10 px-2 py-0.5 rounded">
            STDOUT / DEV
          </span>
        </div>
        <button
          onClick={clearLogs}
          className="text-white/60 hover:text-white transition-colors"
          title="Clear logs"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="p-4 max-h-56 overflow-y-auto space-y-2 select-text">
        {logs.map((log) => (
          <div key={log.id} className="flex items-start gap-2.5 leading-relaxed text-[11px]">
            <span className="text-[#90E0EF]/60 shrink-0 select-none">{log.time}</span>
            <span
              className={`shrink-0 px-1.5 py-0.2 rounded border text-[9px] font-bold ${getBadgeStyle(
                log.subsystem
              )}`}
            >
              {log.subsystem}
            </span>
            <span
              className={
                log.level === 'SUCCESS'
                  ? 'text-[#48CAE4]'
                  : log.level === 'WARN'
                  ? 'text-[#FFB703]'
                  : 'text-[#CAF0F8]'
              }
            >
              {log.message}
            </span>
          </div>
        ))}
        <div ref={logEndRef} />
      </div>
    </div>
  )
}
