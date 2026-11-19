'use client';

import React from 'react';

export function LandingTerminalPreview() {
  return (
    <div className="relative w-full max-w-[540px] flex items-center justify-center select-none py-6">
      {/* Background soft ambient glow */}
      <div
        className="absolute inset-0 bg-brand-glow blur-[80px] rounded-full opacity-25 pointer-events-none"
        aria-hidden="true"
      />

      {/* Layer 1: Left Ghost Window */}
      <div
        className="absolute -left-6 top-8 w-full h-[320px] rounded-2xl border border-border bg-surface-2/40 opacity-25 scale-[0.92] -rotate-1 pointer-events-none hidden sm:block overflow-hidden"
        aria-hidden="true"
      >
        <div className="h-9 border-b border-border/40 px-4 flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
        </div>
        <div className="p-5 space-y-3 opacity-30">
          <div className="h-3 w-1/3 rounded bg-border" />
          <div className="h-3 w-2/3 rounded bg-border" />
          <div className="h-3 w-1/2 rounded bg-border" />
        </div>
      </div>

      {/* Layer 2: Right Ghost Window */}
      <div
        className="absolute -right-6 top-8 w-full h-[320px] rounded-2xl border border-border bg-surface-2/40 opacity-25 scale-[0.92] rotate-1 pointer-events-none hidden sm:block overflow-hidden"
        aria-hidden="true"
      >
        <div className="h-9 border-b border-border/40 px-4 flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
          <div className="w-2.5 h-2.5 rounded-full bg-border/60" />
        </div>
        <div className="p-5 space-y-3 opacity-30">
          <div className="h-3 w-2/5 rounded bg-border" />
          <div className="h-3 w-3/5 rounded bg-border" />
          <div className="h-3 w-1/3 rounded bg-border" />
        </div>
      </div>

      {/* Layer 3: Main Front Terminal Window */}
      <div className="relative z-10 w-full rounded-2xl border border-border bg-surface/95 backdrop-blur-md shadow-2xl overflow-hidden hover:border-brand-border transition-colors duration-300">
        {/* Terminal Window Header with 3 Dots */}
        <div className="h-10 border-b border-border px-4 flex items-center justify-between bg-surface-2/50">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-[#3a3a3a] border border-[#4a4a4a]" />
            <span className="w-2.5 h-2.5 rounded-full bg-[#3a3a3a] border border-[#4a4a4a]" />
            <span className="w-2.5 h-2.5 rounded-full bg-[#3a3a3a] border border-[#4a4a4a]" />
          </div>
          <span className="text-[11px] font-mono text-text-subtle select-none">
            cweave — bash
          </span>
          <div className="w-10" />
        </div>

        {/* Terminal Content Body */}
        <div className="p-5 sm:p-6 font-mono text-[12.5px] sm:text-[13px] leading-[1.65] space-y-4 text-left overflow-x-auto">
          {/* Command 1 */}
          <div className="space-y-1">
            <div className="text-heading font-medium">
              <span className="text-muted select-none">$ </span>cweave cluster status
            </div>
            <div className="text-success text-[12px] flex items-center gap-2">
              <span className="text-success font-semibold">✓</span>
              <span>Available nodes: node-1 (leader), node-2, node-3, node-4, node-5</span>
            </div>
          </div>

          {/* Command 2 */}
          <div className="space-y-1">
            <div className="text-heading font-medium">
              <span className="text-muted select-none">$ </span>cweave put dataset.parquet s3://warehouse/
            </div>
            <div className="text-success text-[12px] flex items-center gap-2">
              <span className="text-success font-semibold">✓</span>
              <span>FastCDC chunked: 4 chunks (SHA-256 CAS dedup)</span>
            </div>
            <div className="text-success text-[12px] flex items-center gap-2">
              <span className="text-success font-semibold">✓</span>
              <span>Quorum committed (W=2, N=3) across 3 nodes</span>
            </div>
          </div>

          {/* Command 3 */}
          <div className="space-y-1">
            <div className="text-heading font-medium">
              <span className="text-muted select-none">$ </span>cweave cluster simulate-failure --node 3
            </div>
            <div className="text-brand text-[12px] flex items-center gap-2">
              <span className="text-brand font-semibold"></span>
              <span>Node-3 offline (failure detected in 300ms)</span>
            </div>
            <div className="text-success text-[12px] flex items-center gap-2">
              <span className="text-success font-semibold">✓</span>
              <span>Self-healing active: replicas rebuilt on node-5</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
