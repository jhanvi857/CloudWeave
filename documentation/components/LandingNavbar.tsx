'use client';

import React, { useEffect, useState } from 'react';
import Link from 'next/link';
import { Search } from 'lucide-react';
import { useSearchContext } from 'fumadocs-ui/contexts/search';
import { CloudWeaveLogo } from './CloudWeaveLogo';

function GithubIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      width="16"
      height="16"
      stroke="currentColor"
      strokeWidth="2"
      fill="none"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
      <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
  );
}

export function LandingNavbar() {
  const { setOpenSearch } = useSearchContext();
  const [isMac, setIsMac] = useState(false);

  useEffect(() => {
    if (typeof window !== 'undefined') {
      setIsMac(/(Mac|iPhone|iPod|iPad)/i.test(navigator.userAgent));
    }
  }, []);

  return (
    <header className="w-full border-b border-border bg-bg/95 backdrop-blur-md sticky top-0 z-50 h-16 flex items-center">
      <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10 flex items-center justify-between gap-4">
        {/* Left: Same Logo component as docs */}
        <Link href="/" className="flex items-center group cursor-pointer shrink-0">
          <CloudWeaveLogo />
        </Link>

        {/* Center: Search bar */}
        <div className="flex-1 max-w-md mx-4">
          <button
            type="button"
            onClick={() => setOpenSearch(true)}
            className="w-full flex items-center gap-2.5 px-3 py-1.5 rounded-lg bg-surface hover:bg-surface-2 border border-border hover:border-brand-border text-[13px] font-sans text-muted hover:text-text transition-colors cursor-pointer text-left shadow-none group"
          >
            <Search className="w-4 h-4 text-subtle group-hover:text-text transition-colors shrink-0" />
            <span className="flex-1 truncate">Search documentation...</span>
            <kbd className="hidden sm:inline-flex items-center gap-0.5 text-[10px] font-mono px-1.5 py-0.5 rounded bg-surface-2 border border-border text-muted">
              {isMac ? '⌘K' : 'Ctrl K'}
            </kbd>
          </button>
        </div>

        {/* Right: GitHub Links */}
        <div className="flex items-center gap-3 shrink-0">
          <a
            href="https://github.com/jhanvi857/cloudweave"
            target="_blank"
            rel="noreferrer"
            className="hidden md:flex items-center gap-1.5 text-[13px] font-sans text-muted hover:text-text transition-colors"
          >
            <GithubIcon className="w-4 h-4" />
            <span>GitHub</span>
          </a>

          <a
            href="https://github.com/jhanvi857/cloudweave"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-surface hover:bg-surface-2 border border-border hover:border-brand-border text-[12px] font-sans text-text transition-colors shadow-none"
          >
            <span className="text-brand">★</span>
            <span className="font-medium">Star on GitHub</span>
            {/* <span className="text-[11px] font-mono text-subtle border-l border-border pl-1.5">
              1.2k
            </span> */}
          </a>
        </div>
      </div>
    </header>
  );
}
