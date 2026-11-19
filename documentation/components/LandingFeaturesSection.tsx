'use client';

import React from 'react';
import Link from 'next/link';
import {
  Database,
  Zap,
  GitBranch,
  ShieldCheck,
  Box,
  Activity,
  HardDrive,
  Clock,
  BarChart3,
  ArrowRight,
} from 'lucide-react';

const features = [
  {
    title: 'S3-Compatible API',
    desc: 'AWS CLI, boto3 and more.',
    icon: Database,
    href: '/docs/reference/s3-api',
  },
  {
    title: 'FastCDC Chunking',
    desc: 'SHA-256 deduplication.',
    icon: Zap,
    href: '/docs/architecture/chunking-deduplication',
  },
  {
    title: 'Consistent Hashing',
    desc: '150 virtual nodes.',
    icon: GitBranch,
    href: '/docs/architecture/consistent-hashing',
  },
  {
    title: 'N/W/R Quorum',
    desc: 'Reliable reads and writes.',
    icon: ShieldCheck,
    href: '/docs/architecture/replication-quorums',
  },
  {
    title: 'Replication or Erasure Coding',
    desc: 'Flexible redundancy modes.',
    icon: Box,
    href: '/docs/architecture/erasure-coding',
  },
  {
    title: 'Failure Detection & Auto Repair',
    desc: 'Gossip heartbeat & healing.',
    icon: Activity,
    href: '/docs/core-concepts/failure-detection',
  },
  {
    title: 'WAL Metadata',
    desc: 'Durable and resilient.',
    icon: HardDrive,
    href: '/docs/core-concepts/wal-metadata',
  },
  {
    title: 'Vector Clocks',
    desc: 'Object versioning.',
    icon: Clock,
    href: '/docs/core-concepts/vector-clocks',
  },
  {
    title: 'Live Cluster Dashboard',
    desc: 'Real-time telemetry & chaos.',
    icon: BarChart3,
    href: '/docs/operations/cluster-configuration',
  },
];

export function LandingFeaturesSection() {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
      {/* Left Info Panel (4 cols) */}
      <div className="lg:col-span-4 space-y-4">
        <div className="eyebrow">
          KEY FEATURES
        </div>

        <h2 className="text-[36px] font-sans font-semibold text-heading leading-[1.15] tracking-[-0.02em]">
          Powerful. Flexible. <br className="hidden sm:inline" />
          Educational.
        </h2>

        <p className="text-[16px] font-sans text-muted leading-[1.7]">
          CloudWeave exposes the core ideas behind distributed object storage without the enterprise bloat. Learn quorum protocols, consistent hashing, and self-healing systems directly on your laptop.
        </p>

        <div className="pt-2">
          <Link
            href="/docs"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>Explore All Features</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      {/* Right 3x3 Feature Card Grid (8 cols) */}
      <div className="lg:col-span-8 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
        {features.map((feat) => {
          const Icon = feat.icon;
          return (
            <Link
              key={feat.title}
              href={feat.href}
              className="p-6 rounded-xl border border-border bg-surface hover:border-brand-border flex items-start gap-3 transition-colors shadow-none group cursor-pointer"
            >
              <div className="w-8 h-8 rounded-lg bg-surface-2 group-hover:bg-brand-soft border border-border group-hover:border-brand-border flex items-center justify-center text-brand shrink-0 mt-0.5 transition-colors">
                <Icon className="w-4 h-4" />
              </div>
              <div className="space-y-1 min-w-0">
                <h3 className="text-[14px] font-sans font-semibold text-heading group-hover:text-heading transition-colors truncate">
                  {feat.title}
                </h3>
                <p className="text-[12px] font-sans text-muted truncate">
                  {feat.desc}
                </p>
              </div>
            </Link>
          );
        })}
      </div>
    </div>
  );
}
