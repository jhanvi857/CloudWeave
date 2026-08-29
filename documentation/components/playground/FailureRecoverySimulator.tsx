'use client'

import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { Server, Activity, AlertOctagon, RefreshCw, CheckCircle, Zap, ShieldAlert, ArrowRight } from 'lucide-react'

interface ClusterNode {
  id: string
  name: string
  status: 'healthy' | 'warning' | 'failed' | 'repairing'
  chunks: number
  storageUsed: string
  ip: string
  role: string
}

interface TimelineEvent {
  time: string
  message: string
  type: 'info' | 'fail' | 'warn' | 'success'
}

export function FailureRecoverySimulator() {
  const [stage, setStage] = useState<'idle' | 'failed' | 'detected' | 'repairing' | 'recovered'>('idle')
  const [timer, setTimer] = useState<number>(3.0)
  const [events, setEvents] = useState<TimelineEvent[]>([
    { time: '12:01:20', message: 'All 5 nodes reporting 500ms heartbeat OK', type: 'info' },
  ])

  // Nodes state
  const [nodes, setNodes] = useState<ClusterNode[]>([
    { id: 'n1', name: 'Node 1', status: 'healthy', chunks: 18, storageUsed: '2.4 GB', ip: '10.0.0.1:9000', role: 'Primary' },
    { id: 'n2', name: 'Node 2', status: 'healthy', chunks: 15, storageUsed: '2.1 GB', ip: '10.0.0.2:9001', role: 'Replica' },
    { id: 'n3', name: 'Node 3', status: 'healthy', chunks: 16, storageUsed: '2.2 GB', ip: '10.0.0.3:9002', role: 'Target Victim' },
    { id: 'n4', name: 'Node 4', status: 'healthy', chunks: 14, storageUsed: '1.9 GB', ip: '10.0.0.4:9003', role: 'Standby' },
    { id: 'n5', name: 'Node 5', status: 'healthy', chunks: 17, storageUsed: '2.3 GB', ip: '10.0.0.5:9004', role: 'Replica' },
  ])

  const handleSimulateFailure = () => {
    setStage('failed')
    setEvents((prev) => [
      { time: '12:01:23', message: 'CRITICAL: SIGKILL signal sent to Node 3 process', type: 'fail' },
      ...prev,
    ])

    // Transition 1: Detection
    setTimeout(() => {
      setStage('detected')
      setNodes((prev) =>
        prev.map((n) =>
          n.id === 'n3'
            ? { ...n, status: 'failed', role: 'DEAD / TIMEOUT' }
            : n.id === 'n1' || n.id === 'n2'
            ? { ...n, status: 'warning' }
            : n
        )
      )
      setEvents((prev) => [
        { time: '12:01:24', message: 'Heartbeat timeout: Node 3 failed 3 consecutive pings (3000ms)', type: 'fail' },
        { time: '12:01:25', message: 'Anti-Entropy detected 16 under-replicated chunks on hash ring', type: 'warn' },
        ...prev,
      ])
    }, 1500)

    // Transition 2: Repair
    setTimeout(() => {
      setStage('repairing')
      setNodes((prev) =>
        prev.map((n) =>
          n.id === 'n4'
            ? { ...n, status: 'repairing', chunks: 22, storageUsed: '3.0 GB', role: 'Repair Target' }
            : n
        )
      )
      setEvents((prev) => [
        { time: '12:01:26', message: 'Self-healing worker spawned: streaming 16 chunks from Node 1 -> Node 4', type: 'warn' },
        ...prev,
      ])
    }, 3200)

    // Transition 3: Recovered
    setTimeout(() => {
      setStage('recovered')
      setNodes((prev) =>
        prev.map((n) =>
          n.id === 'n3'
            ? { ...n, status: 'failed', chunks: 0, storageUsed: '0 GB', role: 'Offline Excluded' }
            : { ...n, status: 'healthy', role: 'Active Quorum' }
        )
      )
      setEvents((prev) => [
        { time: '12:01:28', message: 'Replicas restored: All object manifests at target N=3 quorum', type: 'success' },
        { time: '12:01:29', message: 'Cluster returned to 100% healthy state over 4 active nodes', type: 'success' },
        ...prev,
      ])
    }, 5000)
  }

  const handleReset = () => {
    setStage('idle')
    setNodes([
      { id: 'n1', name: 'Node 1', status: 'healthy', chunks: 18, storageUsed: '2.4 GB', ip: '10.0.0.1:9000', role: 'Primary' },
      { id: 'n2', name: 'Node 2', status: 'healthy', chunks: 15, storageUsed: '2.1 GB', ip: '10.0.0.2:9001', role: 'Replica' },
      { id: 'n3', name: 'Node 3', status: 'healthy', chunks: 16, storageUsed: '2.2 GB', ip: '10.0.0.3:9002', role: 'Target Victim' },
      { id: 'n4', name: 'Node 4', status: 'healthy', chunks: 14, storageUsed: '1.9 GB', ip: '10.0.0.4:9003', role: 'Standby' },
      { id: 'n5', name: 'Node 5', status: 'healthy', chunks: 17, storageUsed: '2.3 GB', ip: '10.0.0.5:9004', role: 'Replica' },
    ])
    setEvents([
      { time: '12:01:20', message: 'All 5 nodes reporting 500ms heartbeat OK', type: 'info' },
    ])
  }

  return (
    <div className="p-4 sm:p-6 bg-white rounded-2xl border border-[#CBD5E1] shadow-xs space-y-6">
      {/* Header with Title & Action */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#E2E8F0] pb-4">
        <div>
          <div className="flex items-center gap-2">
            <Activity className="w-5 h-5 text-[#0077B6]" />
            <h3 className="font-bold text-base text-[#03045E]">Kill a Node. Watch the Cluster Recover.</h3>
          </div>
          <p className="text-xs text-[#64748B] mt-0.5">
            Test CloudWeave&apos;s heartbeat failure detection and background self-healing replica worker.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {stage === 'idle' ? (
            <button
              onClick={handleSimulateFailure}
              className="flex items-center gap-2 rounded-lg bg-[#E5484D] px-4 py-2 text-xs font-bold text-white shadow-xs hover:bg-[#C92A2A] transition-all"
            >
              <AlertOctagon className="w-4 h-4" />
              <span>Simulate Node 3 Failure</span>
            </button>
          ) : (
            <button
              onClick={handleReset}
              className="flex items-center gap-2 rounded-lg border border-[#CBD5E1] bg-white px-4 py-2 text-xs font-semibold text-[#03045E] hover:bg-[#F1F5F9] transition-all"
            >
              <RefreshCw className="w-4 h-4 text-[#0077B6]" />
              <span>Reset Cluster</span>
            </button>
          )}
        </div>
      </div>

      {/* 5-Node Health Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
        {nodes.map((node) => {
          const isDead = node.status === 'failed'
          const isRepairing = node.status === 'repairing'
          const isWarn = node.status === 'warning'

          return (
            <div
              key={node.id}
              className={`p-3.5 rounded-xl border transition-all ${
                isDead
                  ? 'border-[#FECACA] bg-[#FEECEB] animate-pulse'
                  : isRepairing
                  ? 'border-[#FDE68A] bg-[#FFF8E6]'
                  : isWarn
                  ? 'border-[#FDE68A] bg-white'
                  : 'border-[#CBD5E1] bg-white'
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono font-bold text-[#03045E]">{node.name}</span>
                <span
                  className={`h-2.5 w-2.5 rounded-full ${
                    isDead
                      ? 'bg-[#E5484D]'
                      : isRepairing
                      ? 'bg-[#F59E0B] animate-ping'
                      : 'bg-[#22A06B]'
                  }`}
                />
              </div>

              <div className="mt-2 text-[10px] font-mono text-[#64748B] space-y-1">
                <div>IP: <span className="text-[#03045E] font-semibold">{node.ip}</span></div>
                <div>Chunks: <span className="text-[#03045E] font-semibold">{node.chunks}</span></div>
                <div>Disk: <span className="text-[#03045E] font-semibold">{node.storageUsed}</span></div>
              </div>

              <div className="mt-3 pt-2 border-t border-[#E2E8F0] flex items-center justify-between">
                <span
                  className={`text-[9px] font-mono font-bold uppercase px-1.5 py-0.5 rounded ${
                    isDead
                      ? 'bg-[#E5484D] text-white'
                      : isRepairing
                      ? 'bg-[#F59E0B] text-white'
                      : 'bg-[#CAF0F8] text-[#0077B6]'
                  }`}
                >
                  {node.status}
                </span>
                <span className="text-[9px] font-mono text-[#64748B]">{node.role}</span>
              </div>
            </div>
          )
        })}
      </div>

      {/* Real-Time Event Timeline */}
      <div className="rounded-xl border border-[#E2E8F0] bg-[#F8FAFC] p-4 space-y-3">
        <div className="flex items-center justify-between border-b border-[#E2E8F0] pb-2">
          <span className="text-xs font-mono font-bold uppercase text-[#64748B]">
            Cluster Event Timeline
          </span>
          <span className="text-xs font-mono text-[#0077B6] flex items-center gap-1">
            <Activity className="w-3.5 h-3.5" />
            <span>Self-Healing Engine Active</span>
          </span>
        </div>

        <div className="space-y-2 max-h-48 overflow-y-auto font-mono text-xs">
          {events.map((evt, idx) => {
            const typeColor =
              evt.type === 'fail'
                ? 'text-[#E5484D]'
                : evt.type === 'warn'
                ? 'text-[#D97706]'
                : evt.type === 'success'
                ? 'text-[#22A06B]'
                : 'text-[#64748B]'

            return (
              <motion.div
                key={idx}
                initial={{ opacity: 0, x: -5 }}
                animate={{ opacity: 1, x: 0 }}
                className="flex items-start gap-3 py-1 border-b border-[#F1F5F9] last:border-0"
              >
                <span className="text-[10px] text-[#94A3B8] shrink-0">{evt.time}</span>
                <span className={`font-medium ${typeColor}`}>{evt.message}</span>
              </motion.div>
            )
          })}
        </div>
      </div>
    </div>
  )
}
