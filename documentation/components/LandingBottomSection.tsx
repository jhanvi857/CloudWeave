'use client';

import React from 'react';
import Link from 'next/link';
import { Terminal, Box, BarChart3, ArrowRight } from 'lucide-react';

export function LandingBottomSection() {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
      {/* Panel 1: Use It Like S3 */}
      <div className="rounded-xl border border-border bg-surface p-6 flex flex-col justify-between space-y-6 hover:border-brand-border transition-colors shadow-none">
        <div className="space-y-4">
          <div className="w-8 h-8 rounded-lg bg-surface-2 border border-border flex items-center justify-center text-brand">
            <Terminal className="w-4 h-4" />
          </div>

          <h3 className="text-[20px] font-sans font-semibold text-heading">
            Use It Like S3
          </h3>

          <p className="text-[14px] font-sans text-muted leading-relaxed">
            Standard S3 API compatibility means you can point AWS CLI, boto3, or MinIO SDKs directly to your local CloudWeave port.
          </p>

          <div className="p-3 rounded-lg bg-surface-2 border border-border font-mono text-[12px] text-muted overflow-x-auto leading-[1.6]">
            <code>
              <span className="text-brand">aws</span> --endpoint-url http://localhost:9000 \<br />
              &nbsp;&nbsp;s3 mb s3://demo
            </code>
          </div>
        </div>

        <div className="pt-2">
          <Link
            href="/docs/reference/s3-api"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>See Examples</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      {/* Panel 2: Docker & Deployment */}
      <div className="rounded-xl border border-border bg-surface p-6 flex flex-col justify-between space-y-6 hover:border-brand-border transition-colors shadow-none">
        <div className="space-y-4">
          <div className="w-8 h-8 rounded-lg bg-surface-2 border border-border flex items-center justify-center text-brand">
            <Box className="w-4 h-4" />
          </div>

          <h3 className="text-[20px] font-sans font-semibold text-heading">
            Docker & Deployment
          </h3>

          <p className="text-[14px] font-sans text-muted leading-relaxed">
            Drop CloudWeave into your local dev environments or CI test pipelines seamlessly using simple Docker Compose configurations.
          </p>

          <div className="p-3 rounded-lg bg-surface-2 border border-border font-mono text-[12px] text-muted overflow-x-auto leading-[1.6]">
            <code>
              <span className="text-subtle">services:</span><br />
              &nbsp;&nbsp;<span className="text-text">cloudweave:</span><br />
              &nbsp;&nbsp;&nbsp;&nbsp;image: ghcr.io/jhanvi857/cloudweave:latest<br />
              &nbsp;&nbsp;&nbsp;&nbsp;ports: [&quot;9000:9000&quot;]
            </code>
          </div>
        </div>

        <div className="pt-2">
          <Link
            href="/docs/operations/docker-deployment"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>View Compose File</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      {/* Panel 3: Benchmarks */}
      <div className="rounded-xl border border-border bg-surface p-6 flex flex-col justify-between space-y-6 hover:border-brand-border transition-colors shadow-none">
        <div className="space-y-4">
          <div className="w-8 h-8 rounded-lg bg-surface-2 border border-border flex items-center justify-center text-brand">
            <BarChart3 className="w-4 h-4" />
          </div>

          <h3 className="text-[20px] font-sans font-semibold text-heading">
            Benchmarks
          </h3>

          <p className="text-[14px] font-sans text-muted leading-relaxed">
            Locally measured numbers across real runs, not theoretical marketing numbers:
          </p>

          <div className="rounded-lg border border-border bg-surface-2 p-3 font-mono text-[12px] space-y-1.5">
            <div className="flex justify-between border-b border-border pb-1 text-subtle text-[11px]">
              <span>WORKLOAD</span>
              <span>THROUGHPUT</span>
            </div>
            <div className="flex justify-between text-text">
              <span>Upload:</span>
              <span className="text-success font-medium">~152 MB/s</span>
            </div>
            <div className="flex justify-between text-text">
              <span>Download:</span>
              <span className="text-success font-medium">~121 MB/s</span>
            </div>
            <div className="flex justify-between text-text">
              <span>Concurrent:</span>
              <span className="text-success font-medium">~333 req/s</span>
            </div>
            <div className="flex justify-between text-text">
              <span>Deduplicated upload:</span>
              <span className="text-brand font-medium">~485 MB/s</span>
            </div>
          </div>

          <p className="text-[11px] font-mono text-subtle leading-tight">
            *Locally measured benchmarks on NVMe/SSD, not universal performance guarantees.
          </p>
        </div>

        <div className="pt-2">
          <Link
            href="/docs/operations/benchmarks"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>See Full Results</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </div>
  );
}
