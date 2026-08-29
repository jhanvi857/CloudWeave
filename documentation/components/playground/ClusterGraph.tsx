'use client'

import React from 'react'
import { Server, HardDrive, Cpu, Wifi } from 'lucide-react'

export function ClusterGraph() {
  const clusterNodes = [
    { name: 'Node 1', status: 'Healthy', chunks: 18, disk: '2.4 GB / 10 GB', cpu: '8%', role: 'Leader / Ring 0°' },
    { name: 'Node 2', status: 'Healthy', chunks: 15, disk: '2.1 GB / 10 GB', cpu: '5%', role: 'Replica / Ring 72°' },
    { name: 'Node 3', status: 'Healthy', chunks: 16, disk: '2.2 GB / 10 GB', cpu: '6%', role: 'Replica / Ring 144°' },
    { name: 'Node 4', status: 'Healthy', chunks: 14, disk: '1.9 GB / 10 GB', cpu: '4%', role: 'Standby / Ring 216°' },
    { name: 'Node 5', status: 'Healthy', chunks: 17, disk: '2.3 GB / 10 GB', cpu: '7%', role: 'Replica / Ring 288°' },
  ]

  return (
    <div className="w-full space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-xs font-mono font-bold uppercase text-[#64748B]">
          Active 5-Node Local Mesh Topology
        </span>
        <span className="text-xs font-mono text-[#22A06B] font-semibold flex items-center gap-1">
          <span className="h-2 w-2 rounded-full bg-[#22A06B] animate-pulse" />
          Quorum N=3 Active
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
        {clusterNodes.map((node) => (
          <div
            key={node.name}
            className="p-3.5 rounded-xl border border-[#CBD5E1] bg-white shadow-2xs hover:border-[#00B4D8] transition-all"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-mono font-bold text-[#03045E]">{node.name}</span>
              <span className="flex items-center gap-1 text-[10px] font-mono text-[#22A06B] font-semibold">
                <span className="h-1.5 w-1.5 rounded-full bg-[#22A06B]" />
                {node.status}
              </span>
            </div>

            <div className="mt-3 space-y-1 text-[10px] font-mono text-[#64748B]">
              <div className="flex justify-between">
                <span>Chunks:</span>
                <span className="font-semibold text-[#03045E]">{node.chunks}</span>
              </div>
              <div className="flex justify-between">
                <span>Storage:</span>
                <span className="font-semibold text-[#03045E]">{node.disk}</span>
              </div>
              <div className="flex justify-between">
                <span>CPU load:</span>
                <span className="font-semibold text-[#0077B6]">{node.cpu}</span>
              </div>
            </div>

            <div className="mt-3 pt-2 border-t border-[#E2E8F0] text-[9px] font-mono text-[#94A3B8] truncate">
              {node.role}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
