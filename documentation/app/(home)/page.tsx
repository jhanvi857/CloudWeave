import React from 'react';
import Link from 'next/link';
import { ArrowRight, BookOpen } from 'lucide-react';
import { LandingNavbar } from '@/components/LandingNavbar';
import { LandingHeroCluster } from '@/components/LandingHeroCluster';
import { LandingQuickstartSection } from '@/components/LandingQuickstartSection';
import { LandingArchitectureSection } from '@/components/LandingArchitectureSection';
import { LandingBottomSection } from '@/components/LandingBottomSection';
import { LandingTerminalPreview } from '@/components/LandingTerminalPreview';
import { LandingFooter } from '@/components/LandingFooter';

export const metadata = {
  title: 'CloudWeave — S3-Compatible Distributed Object Storage',
  description:
    'A local, S3-compatible distributed object store for learning and prototyping distributed storage. Run multi-node clusters with consistent hashing, FastCDC chunking, quorum reads/writes, and automatic self-healing.',
};

export default function HomePage() {
  return (
    <div className="min-h-screen w-full bg-bg text-text font-sans flex flex-col">
      {/* 1. TOP NAVIGATION */}
      <LandingNavbar />

      {/* 2. MAIN CONTENT SECTIONS */}
      <main className="flex-1 w-full flex flex-col">
        {/* ========================================================================= */}
        {/* SECTION 1: HERO (existing node-diagram art)                               */}
        {/* ========================================================================= */}
        <section className="min-h-[100svh] w-full flex items-center justify-center py-24 border-b border-border">
          <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10">
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-8 items-center">
              {/* Left Column (7 cols) */}
              <div className="lg:col-span-7 space-y-6">
                <div className="eyebrow">
                  DISTRIBUTED STORAGE. MADE SIMPLE.
                </div>

                <h1 className="text-[36px] md:text-[56px] leading-[1.1] font-sans font-semibold text-heading tracking-[-0.02em]">
                  CloudWeave
                </h1>

                <p className="text-[16px] font-sans font-normal text-text leading-[1.7]">
                  A local, S3-compatible distributed object store for learning and prototyping distributed storage.
                </p>

                <p className="text-[16px] font-sans font-normal text-muted leading-[1.7] max-w-xl">
                  Run a multi-node object-storage cluster on your own machine and use it like S3. Upload, distribute, replicate, and repair — while understanding what&apos;s happening under the hood.
                </p>

                {/* CTAs */}
                <div className="flex flex-wrap items-center gap-3.5 pt-3">
                  <Link
                    href="/docs/getting-started"
                    className="inline-flex items-center justify-center gap-2 h-11 px-6 rounded-lg bg-brand hover:opacity-90 text-brand-fg text-[14px] font-sans font-semibold transition-colors cursor-pointer shadow-sm"
                  >
                    <span>Get Started</span>
                    <ArrowRight className="w-4 h-4" />
                  </Link>

                  <Link
                    href="/docs"
                    className="inline-flex items-center justify-center gap-2 h-11 px-5 rounded-lg bg-transparent hover:bg-surface-2 text-text text-[14px] font-sans font-medium border border-border transition-colors cursor-pointer"
                  >
                    <BookOpen className="w-4 h-4 text-muted" />
                    <span>Read the Docs</span>
                  </Link>
                </div>
              </div>

              {/* Right Column: Custom Isometric Illustration (5 cols) */}
              <div className="lg:col-span-5 flex items-center justify-center">
                <LandingHeroCluster />
              </div>
            </div>
          </div>
        </section>

        {/* ========================================================================= */}
        {/* SECTION 2: QUICKSTART (tabs + code + demo card)                           */}
        {/* ========================================================================= */}
        <section className="min-h-[100svh] w-full flex items-center justify-center py-24 border-b border-border">
          <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10">
            <LandingQuickstartSection />
          </div>
        </section>

        {/* ========================================================================= */}
        {/* SECTION 3: ARCHITECTURE (diagram + storage modes)                          */}
        {/* ========================================================================= */}
        <section className="min-h-[100svh] w-full flex items-center justify-center py-24 border-b border-border">
          <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10">
            <LandingArchitectureSection />
          </div>
        </section>

        {/* ========================================================================= */}
        {/* SECTION 4: FEATURES (Use like S3 / Docker / Benchmarks, 3 cards)          */}
        {/* ========================================================================= */}
        <section className="min-h-[100svh] w-full flex items-center justify-center py-24 border-b border-border">
          <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10">
            <div className="space-y-8">
              <div className="space-y-2">
                <div className="eyebrow">FEATURES</div>
                <h2 className="text-[36px] font-sans font-semibold text-heading leading-[1.15] tracking-[-0.02em]">
                  Built for Developers & Explorers
                </h2>
                <p className="text-[16px] font-sans text-muted leading-[1.7] max-w-2xl">
                  Full Amazon S3 compatibility, local Docker orchestration, and transparent performance benchmarks out of the box.
                </p>
              </div>
              <LandingBottomSection />
            </div>
          </div>
        </section>

        {/* ========================================================================= */}
        {/* SECTION 5: FINAL CTA + FOOTER                                             */}
        {/* ========================================================================= */}
        <section className="min-h-[100svh] w-full flex flex-col justify-between pt-24 pb-0">
          <div className="max-w-[1200px] w-full mx-auto px-6 md:px-10 flex-1 flex items-center">
            <div className="grid grid-cols-1 lg:grid-cols-12 gap-10 lg:gap-8 items-center w-full py-12">
              {/* Left Column: CTA Details & Buttons */}
              <div className="lg:col-span-6 space-y-6 text-left">
                <div className="eyebrow">
                  DISTRIBUTED STORAGE FOR EVERYONE
                </div>

                <h2 className="text-[36px] md:text-[52px] font-sans font-semibold text-heading leading-[1.1] tracking-[-0.02em]">
                  Ready to Experiment with Distributed Storage?
                </h2>

                <p className="text-[16px] font-sans text-muted leading-[1.7] max-w-xl">
                  Run a multi-node object-storage cluster on your own machine in seconds, explore the architecture, or start building with our S3-compatible API.
                </p>

                <div className="flex flex-wrap items-center gap-3.5 pt-2">
                  <Link
                    href="/docs/getting-started"
                    className="inline-flex items-center justify-center gap-2 h-11 px-6 rounded-lg bg-brand hover:opacity-90 text-brand-fg text-[14px] font-sans font-semibold transition-colors cursor-pointer shadow-sm"
                  >
                    <span>Get Started</span>
                    <ArrowRight className="w-4 h-4" />
                  </Link>

                  <Link
                    href="/docs"
                    className="inline-flex items-center justify-center gap-2 h-11 px-5 rounded-lg bg-transparent hover:bg-surface-2 text-text text-[14px] font-sans font-medium border border-border transition-colors cursor-pointer"
                  >
                    <BookOpen className="w-4 h-4 text-muted" />
                    <span>Read the Docs</span>
                  </Link>
                </div>
              </div>

              {/* Right Column: Layered Terminal Preview */}
              <div className="lg:col-span-6 flex items-center justify-center">
                <LandingTerminalPreview />
              </div>
            </div>
          </div>

          {/* Footer at bottom of the section */}
          <LandingFooter />
        </section>
      </main>
    </div>
  );
}
