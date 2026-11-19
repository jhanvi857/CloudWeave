'use client';

import React from 'react';
import Link from 'next/link';
import { Database, Server, ArrowDown, ArrowRight } from 'lucide-react';

export function LandingArchitectureSection() {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
      {/* 1. LEFT COLUMN: Text Info (3 cols) */}
      <div className="lg:col-span-3 space-y-4">
        <div className="eyebrow">
          ARCHITECTURE
        </div>

        <h2 className="text-[36px] font-sans font-semibold text-heading leading-[1.15] tracking-[-0.02em]">
          See How It Works
        </h2>

        <p className="text-[16px] font-sans text-muted leading-[1.7]">
          From S3 requests to self-healing clusters — all in a nutshell.
        </p>

        <div className="pt-2">
          <Link
            href="/docs/architecture"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>Architecture Docs</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      {/* 2. CENTER COLUMN: Custom Architecture Flowchart (6 cols) */}
      <div className="lg:col-span-6 rounded-xl border border-border bg-surface p-6 sm:p-7 flex flex-col justify-center items-center shadow-none hover:border-brand-border transition-colors relative">
        <div className="w-full max-w-sm flex flex-col items-center space-y-2.5 font-mono text-[12px]">
          {/* Step 1: Client Layer */}
          <div className="w-full py-2.5 px-4 rounded-lg bg-surface-2 border border-border text-text font-medium text-center">
            AWS CLI / boto3 / Go SDK
          </div>

          <div className="flex items-center justify-center py-0.5 text-brand">
            <ArrowDown className="w-4 h-4" />
          </div>

          {/* Step 2: S3 API Gateway */}
          <div className="w-full py-2.5 px-4 rounded-lg bg-surface-2 border border-border text-text font-medium text-center">
            S3-Compatible API
          </div>

          <div className="flex items-center justify-center py-0.5 text-brand">
            <ArrowDown className="w-4 h-4" />
          </div>

          {/* Step 3: FastCDC Streaming Chunker */}
          <div className="w-full py-2.5 px-4 rounded-lg bg-surface-2 border border-border text-text font-medium text-center">
            Streaming Chunker (FastCDC + SHA-256)
          </div>

          <div className="flex items-center justify-center py-0.5 text-brand">
            <ArrowDown className="w-4 h-4" />
          </div>

          {/* Step 4: Consistent Hash Ring + Storage Nodes */}
          <div className="w-full flex items-center gap-3">
            <div className="flex-1 py-2.5 px-3 rounded-lg bg-surface-2 border border-border text-text font-medium text-center text-[12px]">
              Consistent Hash Ring (N/W/R Quorum)
            </div>

            {/* Storage Node Containers */}
            <div className="flex flex-col gap-1 p-2 rounded-lg border border-border bg-surface text-[10px] shrink-0">
              <div className="flex items-center gap-1.5 text-brand font-medium">
                <Server className="w-3 h-3" />
                <span>Node 1</span>
              </div>
              <div className="flex items-center gap-1.5 text-brand font-medium">
                <Server className="w-3 h-3" />
                <span>Node 2</span>
              </div>
              <div className="flex items-center gap-1.5 text-brand font-medium">
                <Server className="w-3 h-3" />
                <span>Node 3</span>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-center py-0.5 text-brand">
            <ArrowDown className="w-4 h-4" />
          </div>

          {/* Step 5: Failure Detection & Repair */}
          <div className="w-full py-2.5 px-4 rounded-lg border border-border bg-surface-2 text-muted text-center text-[12px]">
            Failure Detection → Replica Repair
          </div>
        </div>
      </div>

      {/* 3. RIGHT COLUMN: Storage Modes Comparison Panel (3 cols) */}
      <div className="lg:col-span-3 rounded-xl border border-border bg-surface p-6 flex flex-col justify-between shadow-none hover:border-brand-border transition-colors">
        <div className="space-y-4">
          <div className="w-8 h-8 rounded-lg bg-surface-2 border border-border flex items-center justify-center text-brand">
            <Database className="w-4 h-4" />
          </div>

          <h3 className="text-[20px] font-sans font-semibold text-heading tracking-tight">
            Storage Modes
          </h3>

          <div className="space-y-4 font-sans text-[13px] border-t border-border pt-4">
            {/* Mode 1: Replication */}
            <div className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="font-medium text-text">Replication</span>
                <span className="text-brand text-[11px] font-mono px-2 py-0.5 rounded-md bg-brand-soft border border-brand-border">
                  N = 3
                </span>
              </div>
              <p className="text-muted text-[13px] leading-relaxed">
                Faster, 3× storage.
              </p>
            </div>

            {/* Mode 2: Erasure Coding */}
            <div className="space-y-1 border-t border-border pt-3">
              <div className="flex items-center justify-between">
                <span className="font-medium text-text">Erasure Coding</span>
                <span className="text-brand text-[11px] font-mono px-2 py-0.5 rounded-md bg-brand-soft border border-brand-border">
                  K = 4, M = 2
                </span>
              </div>
              <p className="text-muted text-[13px] leading-relaxed">
                ~1.5× storage, higher recovery cost.
              </p>
            </div>
          </div>
        </div>

        <div className="pt-4 border-t border-border mt-4">
          <Link
            href="/docs/architecture/erasure-coding"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>Learn More</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </div>
  );
}
