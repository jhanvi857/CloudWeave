'use client'

import React from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { ArrowRight } from 'lucide-react'

function VaultDialGlyph(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 28 28" width="24" height="24" fill="none" stroke="currentColor" strokeWidth="1.5" {...props}>
      {/* Outer Dial Ring */}
      <circle cx="14" cy="14" r="11.5" stroke="#7A1F2B" strokeWidth="1.5" />
      <circle cx="14" cy="14" r="8.5" stroke="#26262B" strokeWidth="1" strokeDasharray="2 3" />
      {/* Central Spindle Hub */}
      <circle cx="14" cy="14" r="3" fill="#7A1F2B" stroke="#000000" strokeWidth="1" />
      {/* Locking Tumbler Notches */}
      <line x1="14" y1="2.5" x2="14" y2="6.5" stroke="#7A1F2B" strokeWidth="2" strokeLinecap="round" />
      <line x1="25.5" y1="14" x2="21.5" y2="14" stroke="#A6A9AE" strokeWidth="1.5" />
      <line x1="14" y1="25.5" x2="14" y2="21.5" stroke="#A6A9AE" strokeWidth="1.5" />
      <line x1="2.5" y1="14" x2="6.5" y2="14" stroke="#A6A9AE" strokeWidth="1.5" />
    </svg>
  )
}

function GithubIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
      <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
  )
}

export function Navbar() {
  const pathname = usePathname()

  const navItems = [
    { href: '/playground', label: 'Playground' },
    { href: '/how-it-works', label: 'How It Works' },
    { href: '/concepts', label: 'Concepts' },
    { href: '/benchmarks', label: 'Benchmarks' },
    { href: '/docs', label: 'Docs' },
  ]

  return (
    <header className="sticky top-0 z-50 w-full border-b border-[#26262B] bg-[#000000]/90 backdrop-blur-md transition-colors">
      <div className="w-full max-w-[1380px] mx-auto px-6 h-16 flex items-center justify-between">
        {/* Brand */}
        <div className="flex items-center gap-8">
          <Link
            href="/"
            className="flex items-center gap-3 group transition-opacity hover:opacity-95"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#19191C] border border-[#26262B] group-hover:border-[#7A1F2B]/60 transition-colors shadow-xs">
              <VaultDialGlyph className="w-5 h-5 text-[#7A1F2B]" />
            </div>
            <div className="flex flex-col">
              <span className="text-[17px] font-serif font-bold tracking-tight text-[#EDE8DF]">
                CloudWeave
              </span>
              <span className="text-[9px] font-mono tracking-widest text-[#8B8B8F] uppercase -mt-0.5">
                Object Storage
              </span>
            </div>
          </Link>

          {/* Navigation Links */}
          <nav className="hidden lg:flex items-center gap-7 text-sm font-medium">
            {navItems.map((item) => {
              const isActive = pathname === item.href || (item.href !== '/' && pathname.startsWith(item.href))
              return (
                <Link
                  key={item.label}
                  href={item.href}
                  className={`relative py-1 text-[13px] font-mono tracking-wide transition-colors hover:text-[#EDE8DF] ${isActive ? 'text-[#EDE8DF] font-bold' : 'text-[#8B8B8F]'
                    }`}
                >
                  {item.label}
                  {isActive && (
                    <span className="absolute bottom-0 left-0 w-full h-[2px] bg-[#7A1F2B] rounded-full shadow-[0_0_8px_rgba(122,31,43,0.8)]" />
                  )}
                </Link>
              )
            })}
          </nav>
        </div>

        {/* Right Actions */}
        <div className="flex items-center gap-4">
          <a
            href="https://github.com/jhanvi857/CloudWeave"
            target="_blank"
            rel="noreferrer"
            className="hidden sm:flex items-center gap-1.5 text-xs font-mono text-[#8B8B8F] hover:text-[#EDE8DF] transition-colors px-2 py-1"
          >
            <GithubIcon className="w-4 h-4 text-[#8B8B8F]" />
            <span>GitHub</span>
          </a>

          <Link
            href="/docs"
            className="inline-flex justify-center items-center h-9 px-4 text-xs font-mono font-semibold bg-[#7A1F2B] text-[#EDE8DF] hover:bg-[#912735] transition-all rounded-md shadow-xs gap-1.5 cursor-pointer active:scale-98"
          >
            <span>Quickstart</span>
            <ArrowRight className="w-3.5 h-3.5 stroke-[2.5]" />
          </Link>
        </div>
      </div>
    </header>
  )
}


