import React from 'react'
import Link from 'next/link'

export function Footer() {
  return (
    <footer className="w-full border-t border-[#26262B] bg-[#000000] py-12 px-6 font-sans">
      <div className="max-w-[1380px] mx-auto flex flex-col md:flex-row items-center justify-between gap-6">
        <div className="flex flex-col items-center md:items-start space-y-1 text-center md:text-left">
          <div className="flex items-center gap-2">
            <span className="font-serif font-bold text-sm tracking-tight text-[#EDE8DF]">
              CloudWeave
            </span>
            <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-[#19191C] border border-[#26262B] text-[#A6A9AE]">
              v1.0.0 (Go 1.24)
            </span>
          </div>
          <p className="text-xs font-mono text-[#8B8B8F]">
            Deterministic distributed object storage. FastCDC · SHA-256 CAS · Dynamo Quorum · Reed-Solomon.
          </p>
        </div>

        <div className="flex items-center gap-6 text-xs font-mono text-[#8B8B8F]">
          <Link href="/playground" className="hover:text-[#EDE8DF] transition-colors">
            Playground
          </Link>
          <Link href="/how-it-works" className="hover:text-[#EDE8DF] transition-colors">
            How It Works
          </Link>
          <Link href="/concepts" className="hover:text-[#EDE8DF] transition-colors">
            Concepts
          </Link>
          <Link href="/benchmarks" className="hover:text-[#EDE8DF] transition-colors">
            Benchmarks
          </Link>
          <Link href="/docs" className="hover:text-[#EDE8DF] transition-colors">
            Docs
          </Link>
          <a
            href="https://github.com/jhanvi857/CloudWeave"
            target="_blank"
            rel="noreferrer"
            className="text-[#A6A9AE] hover:text-[#EDE8DF] transition-colors"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  )
}

