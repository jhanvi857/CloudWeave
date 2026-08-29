'use client'

import React from 'react'
import { Search, Terminal, Play, Menu, X } from 'lucide-react'

function GithubIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
      <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
  )
}

interface HeaderProps {
  activeTab: string
  setActiveTab: (tab: string) => void
  onOpenSearch: () => void
  isMobileMenuOpen: boolean
  setIsMobileMenuOpen: (open: boolean) => void
}

export function Header({
  activeTab,
  setActiveTab,
  onOpenSearch,
  isMobileMenuOpen,
  setIsMobileMenuOpen,
}: HeaderProps) {
  const navItems = [
    { id: 'playground', label: 'Playground' },
    { id: 'architecture', label: 'Architecture' },
    { id: 'concepts', label: 'Concepts' },
    { id: 'internals', label: 'Internals' },
    { id: 'benchmarks', label: 'Benchmarks' },
    { id: 'decisions', label: 'ADRs' },
    { id: 'reference', label: 'API & CLI' },
  ]

  return (
    <header className="sticky top-0 z-40 w-full border-b border-[#E2E8F0] bg-white/95 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        {/* Left: Brand */}
        <div className="flex items-center gap-6">
          <button
            onClick={() => setActiveTab('playground')}
            className="flex items-center gap-3 text-left group cursor-pointer"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-[#0077B6] font-mono text-sm font-bold text-white shadow-xs group-hover:bg-[#005F92] transition-colors">
              CW
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-base tracking-tight text-[#03045E]">CloudWeave</span>
                <span className="hidden sm:inline-block rounded bg-[#CAF0F8] px-1.5 py-0.5 text-[10px] font-mono font-bold text-[#0077B6]">
                  v1.2
                </span>
              </div>
              <p className="hidden md:block text-[11px] font-mono text-[#64748B]">
                Distributed Object Storage
              </p>
            </div>
          </button>
        </div>

        {/* Center: Nav links */}
        <nav className="hidden lg:flex items-center gap-1">
          {navItems.map((item) => {
            const isActive = activeTab === item.id
            return (
              <button
                key={item.id}
                onClick={() => setActiveTab(item.id)}
                className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-all cursor-pointer ${isActive
                    ? 'bg-[#CAF0F8] text-[#0077B6] font-bold shadow-2xs'
                    : 'text-[#475569] hover:bg-[#F8FAFC] hover:text-[#0077B6]'
                  }`}
              >
                {item.label}
              </button>
            )
          })}
        </nav>

        {/* Right: Actions */}
        <div className="flex items-center gap-2.5">
          {/* Quick Search */}
          <button
            onClick={onOpenSearch}
            className="flex items-center gap-2 rounded-lg border border-[#CBD5E1] bg-white px-3 py-1.5 text-xs text-[#64748B] hover:border-[#0077B6] hover:bg-white hover:text-[#0077B6] transition-all shadow-2xs cursor-pointer"
          >
            <Search className="w-3.5 h-3.5 text-[#0077B6]" />
            <span className="hidden sm:inline">Search docs & simulations...</span>
            <span className="sm:hidden">Search</span>
            <kbd className="hidden sm:inline-block rounded border border-[#CBD5E1] bg-[#F8FAFC] px-1.5 py-0.5 text-[10px] font-mono text-[#64748B]">
              ⌘K
            </kbd>
          </button>

          {/* GitHub link */}
          <a
            href="https://github.com/jhanvi857/CloudWeave"
            target="_blank"
            rel="noopener noreferrer"
            className="hidden sm:flex h-8 w-8 items-center justify-center rounded-lg border border-[#E2E8F0] text-[#475569] hover:bg-[#F8FAFC] hover:text-[#0077B6] transition-colors"
            title="GitHub Repository"
          >
            <GithubIcon className="w-4 h-4" />
          </a>

          {/* Quickstart CTA */}
          <button
            onClick={() => setActiveTab('quickstart')}
            className="hidden sm:flex items-center gap-1.5 rounded-lg bg-[#0077B6] px-3 py-1.5 text-xs font-bold text-white hover:bg-[#005F92] shadow-xs transition-colors cursor-pointer"
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Quick Start</span>
          </button>

          {/* Mobile menu toggle */}
          <button
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
            className="lg:hidden flex h-8 w-8 items-center justify-center rounded-lg border border-[#E2E8F0] text-[#475569] cursor-pointer"
          >
            {isMobileMenuOpen ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
          </button>
        </div>
      </div>

      {/* Mobile nav dropdown */}
      {isMobileMenuOpen && (
        <div className="lg:hidden border-t border-[#E2E8F0] bg-white px-4 py-3 space-y-1">
          {navItems.map((item) => (
            <button
              key={item.id}
              onClick={() => {
                setActiveTab(item.id)
                setIsMobileMenuOpen(false)
              }}
              className={`w-full text-left rounded-md px-3 py-2 text-xs font-semibold ${activeTab === item.id
                  ? 'bg-[#CAF0F8] text-[#0077B6] font-bold'
                  : 'text-[#475569] hover:bg-[#F8FAFC] hover:text-[#0077B6]'
                }`}
            >
              {item.label}
            </button>
          ))}
          <div className="pt-2 border-t border-[#E2E8F0] flex gap-2">
            <button
              onClick={() => {
                setActiveTab('quickstart')
                setIsMobileMenuOpen(false)
              }}
              className="w-full flex items-center justify-center gap-1.5 rounded-md bg-[#0077B6] py-2 text-xs font-bold text-white"
            >
              <Terminal className="w-3.5 h-3.5" /> Quick Start
            </button>
          </div>
        </div>
      )}
    </header>
  )
}
