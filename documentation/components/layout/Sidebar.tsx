'use client'

import React from 'react'
import {
  Play,
  Layers,
  Server,
  Activity,
  ShieldCheck,
  Cpu,
  BookOpen,
  Terminal,
  FileCode,
  Zap,
  HardDrive,
  GitBranch,
} from 'lucide-react'

interface SidebarProps {
  currentSection: string
  setCurrentSection: (section: string) => void
  activeTab: string
  setActiveTab: (tab: string) => void
}

interface NavGroup {
  title: string
  items: {
    id: string
    label: string
    tab: string
    icon: any
    badge?: string
  }[]
}

const navGroups: NavGroup[] = [
  {
    title: 'GET STARTED',
    items: [
      { id: 'intro', label: 'Introduction', tab: 'playground', icon: BookOpen },
      { id: 'quickstart', label: 'Quick Start', tab: 'quickstart', icon: Terminal, badge: '5 min' },
      { id: 'installation', label: 'Installation', tab: 'quickstart', icon: HardDrive },
      { id: 'aws-cli', label: 'AWS CLI Guide', tab: 'quickstart', icon: Terminal },
    ],
  },
  {
    title: 'PLAYGROUND',
    items: [
      { id: 'pg-overview', label: 'Overview', tab: 'playground', icon: Play },
      { id: 'pg-upload', label: 'Upload Object', tab: 'playground', icon: Layers },
      { id: 'pg-chunking', label: 'Chunking', tab: 'playground', icon: Zap },
      { id: 'pg-distribution', label: 'Data Distribution', tab: 'playground', icon: Server },
      { id: 'pg-replication', label: 'Replication', tab: 'playground', icon: Layers },
      { id: 'pg-failure', label: 'Failure Demo', tab: 'playground', icon: Activity, badge: 'Interactive' },
      { id: 'pg-repair', label: 'Repair', tab: 'playground', icon: Zap },
      { id: 'pg-metrics', label: 'Metrics', tab: 'playground', icon: ShieldCheck },
    ],
  },
  {
    title: 'CONCEPTS',
    items: [
      { id: 'concept-fastcdc', label: 'FastCDC', tab: 'concepts', icon: Zap },
      { id: 'concept-dedup', label: 'Deduplication', tab: 'concepts', icon: HardDrive },
      { id: 'concept-ring', label: 'Consistent Hashing', tab: 'concepts', icon: Server },
      { id: 'concept-quorum', label: 'Quorum', tab: 'concepts', icon: ShieldCheck },
      { id: 'concept-replication', label: 'Replication', tab: 'concepts', icon: Layers },
      { id: 'concept-erasure', label: 'Erasure Coding', tab: 'concepts', icon: Cpu },
      { id: 'concept-vclock', label: 'Vector Clocks', tab: 'concepts', icon: GitBranch },
      { id: 'concept-wal', label: 'WAL', tab: 'concepts', icon: HardDrive },
    ],
  },
  {
    title: 'ARCHITECTURE',
    items: [
      { id: 'arch-overview', label: 'System Overview', tab: 'architecture', icon: Layers },
      { id: 'arch-write', label: 'Write Path', tab: 'architecture', icon: Server },
      { id: 'arch-read', label: 'Read Path', tab: 'architecture', icon: Server },
      { id: 'arch-recovery', label: 'Recovery', tab: 'architecture', icon: Activity },
    ],
  },
]

export function Sidebar({
  currentSection,
  setCurrentSection,
  activeTab,
  setActiveTab,
}: SidebarProps) {
  return (
    <aside className="w-64 shrink-0 border-r border-[#26262B] bg-[#000000] h-[calc(100vh-4rem)] sticky top-16 overflow-y-auto p-4 hidden md:block font-sans">
      <div className="space-y-6">
        {navGroups.map((group) => (
          <div key={group.title} className="space-y-1">
            <h4 className="px-2.5 text-[11px] font-mono font-bold tracking-wider text-[#A6A9AE] uppercase">
              {group.title}
            </h4>
            <div className="space-y-0.5 pt-1">
              {group.items.map((item) => {
                const isCurrent = currentSection === item.id || (activeTab === item.tab && !currentSection && item.id === 'pg-overview')
                const Icon = item.icon
                return (
                  <button
                    key={item.id}
                    onClick={() => {
                      setCurrentSection(item.id)
                      setActiveTab(item.tab)
                    }}
                    className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded text-xs font-mono transition-all cursor-pointer ${isCurrent
                        ? 'bg-[#19191C] text-[#EDE8DF] font-bold border-l-2 border-[#7A1F2B]'
                        : 'text-[#8B8B8F] hover:bg-[#19191C] hover:text-[#EDE8DF]'
                      }`}
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <Icon className={`w-3.5 h-3.5 shrink-0 ${isCurrent ? 'text-[#7A1F2B]' : 'text-[#A6A9AE]'}`} />
                      <span className="truncate">{item.label}</span>
                    </div>
                    {item.badge && (
                      <span className="shrink-0 rounded px-1.5 py-0.5 text-[9px] font-mono font-bold bg-[#19191C] text-[#A6A9AE] border border-[#26262B]">
                        {item.badge}
                      </span>
                    )}
                  </button>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      {/* Cluster Status Card */}
      <div className="mt-8 rounded-xl border border-[#26262B] bg-[#19191C] p-3 space-y-2 font-mono">
        <div className="text-[10px] font-bold text-[#8B8B8F] uppercase">Cluster Status</div>
        <div className="flex items-center gap-1.5 text-xs font-bold text-[#EDE8DF]">
          <span className="h-2 w-2 rounded-full bg-[#7A1F2B]" />
          <span>Healthy</span>
        </div>
        <div className="grid grid-cols-2 gap-1 pt-1 text-[11px] text-[#8B8B8F]">
          <div><b className="text-[#EDE8DF]">5</b> Nodes</div>
          <div><b className="text-[#EDE8DF]">1,024</b> Chunks</div>
        </div>
      </div>
    </aside>
  )
}
