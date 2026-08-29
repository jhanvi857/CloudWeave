'use client'

import React, { useState } from 'react'
import Link from 'next/link'
import { motion, AnimatePresence } from 'framer-motion'
import { VaultRing3D } from '../canvas/VaultRing3D'
import { FullBleedVaultLock3D } from '../canvas/FullBleedVaultLock3D'
import {
  ArrowRight,
  ShieldCheck,
  Layers,
  Server,
  Lock,
  Cpu,
  Zap,
  HardDrive,
  Copy,
  Check,
} from 'lucide-react'

function GithubIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 24 24" width="16" height="16" stroke="currentColor" strokeWidth="2" fill="none" strokeLinecap="round" strokeLinejoin="round" {...props}>
      <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
      <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
  )
}

export function LandingPage() {
  const [copied, setCopied] = useState(false)
  const [isCtaVaultLocked, setIsCtaVaultLocked] = useState(false)

  const copyInstall = () => {
    navigator.clipboard.writeText('go install github.com/jhanvi857/cloudweave/cmd/cweave@latest')
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const features = [
    {
      title: 'FastCDC Chunking',
      desc: 'Rolling Gear hash eliminates boundary-shift problems, preserving 80%+ duplicate blocks across file edits.',
      icon: Layers,
    },
    {
      title: 'SHA-256 CAS & Dedup',
      desc: 'Cryptographic hash identifies immutable content chunks, skipping disk writes for existing payloads.',
      icon: Lock,
    },
    {
      title: 'Consistent Hash Ring',
      desc: '150 virtual tumblers per node balance keys evenly across 32-bit integer space with minimal remapping on changes.',
      icon: Server,
    },
    {
      title: 'Dynamo Quorum (N/W/R)',
      desc: 'Decentralized fan-out quorum writes and reads guarantee strong consistency without a bottleneck single master.',
      icon: ShieldCheck,
    },
    {
      title: 'Reed-Solomon Erasure Coding',
      desc: 'Cauchy matrix reconstruction (K=4, M=2) survives 2 dead nodes with only 1.5x storage overhead versus 3.0x replication.',
      icon: Cpu,
    },
    {
      title: 'Self-Healing Anti-Entropy',
      desc: '300ms gossip heartbeat failure detection prompts repair workers to re-replicate under-replicated chunks.',
      icon: Zap,
    },
    {
      title: 'Mutual TLS Zero-Trust',
      desc: 'Peer-to-peer transport with pooled persistent HTTP keep-alive connections eliminates TLS handshake penalties.',
      icon: Lock,
    },
    {
      title: 'AWS S3 & SigV4 REST API',
      desc: 'Full drop-in compatibility with standard AWS SDKs, boto3, and AWS CLI using HMAC-SHA256 signature verification.',
      icon: HardDrive,
    },
  ]

  const pipelineKeyframes = [
    {
      step: '01',
      title: 'Ingestion & SigV4',
      desc: 'Incoming S3 PUT requests are cryptographically verified and streamed into sync.Pool byte buffers.',
    },
    {
      step: '02',
      title: 'FastCDC Chunking',
      desc: 'Gear rolling hash identifies dynamic boundary cuts to produce 16 KB - 256 KB content chunks.',
    },
    {
      step: '03',
      title: '32-Bit Ring Placement',
      desc: 'SHA-256 CAS chunk IDs are mapped to target primary and replica tumbler nodes on the hash ring.',
    },
    {
      step: '04',
      title: 'Quorum & WAL Commit',
      desc: 'Parallel fan-out collects W=2 fsync acknowledgments and atomically writes manifest to the WAL.',
    },
  ]

  return (
    <div className="w-full flex flex-col items-center bg-[#000000] text-[#EDE8DF] selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF] font-sans">
      {/* ========================================================================= */}
      {/* SECTION 1: HERO WITH 3D VAULT DIAL                                       */}
      {/* ========================================================================= */}
      <section className="relative w-full pt-12 pb-16 lg:pt-16 lg:pb-24 border-b border-[#26262B] bg-[#000000] overflow-hidden">
        <div className="max-w-[1380px] mx-auto px-6 w-full flex flex-col lg:flex-row items-center gap-12 lg:gap-14">
          {/* Left: 45% Text */}
          <div className="w-full lg:w-[45%] flex flex-col items-start text-left space-y-6">
            <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
              <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
              <span>Go Distributed Storage Engine</span>
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-serif font-bold tracking-tight text-[#EDE8DF] leading-[1.12]">
              Deterministic storage. <br />
              <span className="text-[#EDE8DF] italic">Mechanically sound.</span>
            </h1>

            <p className="text-sm sm:text-base font-mono text-[#8B8B8F] leading-relaxed">
              A high-performance S3-compatible distributed object store in Go. Built on FastCDC chunking, SHA-256 CAS deduplication, 150-vnode consistent hashing, and self-healing quorum repair.
            </p>

            <div className="flex flex-wrap items-center gap-3 pt-2">
              <Link
                href="/playground"
                className="inline-flex items-center justify-center h-11 px-6 text-xs font-mono font-semibold bg-[#7A1F2B] text-[#EDE8DF] hover:bg-[#912735] transition-all rounded-md shadow-sm gap-2 cursor-pointer active:scale-98"
              >
                <span>Launch Playground</span>
                <ArrowRight className="w-3.5 h-3.5 stroke-[2.5]" />
              </Link>

              <Link
                href="/how-it-works"
                className="inline-flex items-center justify-center h-11 px-5 text-xs font-mono text-[#EDE8DF] bg-[#19191C] border border-[#26262B] hover:border-[#7A1F2B] transition-all rounded-md cursor-pointer"
              >
                <span>Explore How It Works</span>
              </Link>
            </div>
          </div>

          {/* Right: 55% 3D Signature Vault Dial */}
          <div className="w-full lg:w-[55%]">
            <VaultRing3D />
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* SECTION 2: PROOF STRIP (3 MONO STATS - NO ICONS, THIN RULES ONLY)        */}
      {/* ========================================================================= */}
      <section className="w-full border-b border-[#26262B] bg-[#000000]">
        <div className="max-w-[1380px] mx-auto px-6 py-8">
          <div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-[#26262B] font-mono">
            {/* Stat 1 */}
            <div className="py-4 md:py-0 md:px-8 first:pl-0 flex flex-col space-y-1">
              <span className="text-2xl sm:text-3xl font-bold text-[#EDE8DF]">151.6 MB/s</span>
              <span className="text-xs text-[#EDE8DF] font-semibold">Median Parallel Upload Throughput</span>
              <span className="text-[11px] text-[#8B8B8F]">Bounded 8-worker streaming with zero-alloc sync.Pool buffers</span>
            </div>

            {/* Stat 2 */}
            <div className="py-4 md:py-0 md:px-8 flex flex-col space-y-1">
              <span className="text-2xl sm:text-3xl font-bold text-[#EDE8DF]">87.3%</span>
              <span className="text-xs text-[#EDE8DF] font-semibold">Memory Headroom under Load</span>
              <span className="text-[11px] text-[#8B8B8F]">32.5 MB heap usage under 5 concurrent 50 MB streams</span>
            </div>

            {/* Stat 3 */}
            <div className="py-4 md:py-0 md:px-8 last:pr-0 flex flex-col space-y-1">
              <span className="text-2xl sm:text-3xl font-bold text-[#EDE8DF]">1.5x</span>
              <span className="text-xs text-[#EDE8DF] font-semibold">Reed-Solomon (4,2) Parity Overhead</span>
              <span className="text-[11px] text-[#8B8B8F]">50% lower raw storage cost vs 3.0x raw replication</span>
            </div>
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* SECTION 3: CONDENSED 4-KEYFRAME PIPELINE PREVIEW                         */}
      {/* ========================================================================= */}
      <section className="w-full py-20 border-b border-[#26262B] bg-[#000000]">
        <div className="max-w-[1380px] mx-auto px-6 space-y-10">
          <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
            <div className="space-y-2">
              <div className="text-xs font-mono text-[#A6A9AE] font-semibold">
                SYSTEM PIPELINE
              </div>
              <h2 className="text-3xl font-serif font-bold text-[#EDE8DF]">
                How an Object is Sealed
              </h2>
            </div>
            <Link
              href="/how-it-works"
              className="inline-flex items-center gap-1.5 text-xs font-mono text-[#EDE8DF] hover:underline"
            >
              <span>View full 6-stage interactive walkthrough</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </Link>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 font-mono">
            {pipelineKeyframes.map((kf) => (
              <div
                key={kf.step}
                className="p-5 rounded-lg border border-[#26262B] bg-[#19191C] space-y-2 flex flex-col justify-between"
              >
                <div className="space-y-2">
                  <div className="text-xs text-[#A6A9AE] font-bold">STAGE {kf.step}</div>
                  <div className="text-sm font-serif font-bold text-[#EDE8DF]">{kf.title}</div>
                  <p className="text-xs font-sans text-[#8B8B8F] leading-relaxed">{kf.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* SECTION 4: FEATURE GRID (8 FLAT BORDERED PANELS - GLYPH + 2 LINES)       */}
      {/* ========================================================================= */}
      <section className="w-full py-20 border-b border-[#26262B] bg-[#000000]">
        <div className="max-w-[1380px] mx-auto px-6 space-y-10">
          <div className="space-y-2">
            <div className="text-xs font-mono text-[#A6A9AE] font-semibold">
              ENGINE SPECIFICATION
            </div>
            <h2 className="text-3xl font-serif font-bold text-[#EDE8DF]">
              Distributed Core Architecture
            </h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {features.map((feat) => {
              const Icon = feat.icon
              return (
                <div
                  key={feat.title}
                  className="p-5 rounded-lg border border-[#26262B] bg-[#19191C] space-y-3 hover:border-[#34343A] transition-colors"
                >
                  <div className="w-8 h-8 rounded bg-[#000000] border border-[#26262B] flex items-center justify-center text-[#A6A9AE]">
                    <Icon className="w-4 h-4" />
                  </div>
                  <div className="space-y-1">
                    <h3 className="font-serif font-bold text-sm text-[#EDE8DF]">
                      {feat.title}
                    </h3>
                    <p className="text-xs font-mono text-[#8B8B8F] leading-relaxed">
                      {feat.desc}
                    </p>
                  </div>
                </div>
              )
            })}
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* SECTION 5: FULL-BLEED 3D CLOSING CTA SECTION                             */}
      {/* ========================================================================= */}
      <section className="w-full min-h-[90vh] py-24 bg-[#000000] border-b border-[#26262B] relative flex flex-col items-center justify-center overflow-hidden">
        {/* Massive Centered Display Headline */}
        <div className="relative z-10 max-w-4xl mx-auto px-6 text-center space-y-4">
          <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
            <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
            <span>Distributed Object Storage</span>
          </div>

          <h2 className="text-5xl sm:text-6xl lg:text-7xl xl:text-8xl font-serif font-bold tracking-tight text-[#EDE8DF] leading-[1.05]">
            Every write. <br className="hidden sm:inline" />
            <span className="text-[#EDE8DF] italic">Sealed shut.</span>
          </h2>

          <p className="text-sm sm:text-base font-mono text-[#8B8B8F] max-w-xl mx-auto">
            Deterministic distributed object storage in Go. FastCDC chunking, SHA-256 CAS, and quorum replication.
          </p>

          <p className="text-xs font-mono text-[#A6A9AE]">
            {isCtaVaultLocked
              ? '✓ Cluster lock engaged — ready for deployment.'
              : 'Click or drag the hash ring dial below to lock the cluster state.'}
          </p>
        </div>

        {/* Centerpiece Full-Bleed 3D Hash Ring Lock Mechanism */}
        <div className="relative z-10 w-full max-w-3xl h-[380px] sm:h-[440px] lg:h-[480px] my-4">
          <FullBleedVaultLock3D
            isLocked={isCtaVaultLocked}
            setIsLocked={setIsCtaVaultLocked}
          />
        </div>

        {/* Revealed Content: Copyable Install & Centered Actions */}
        <div className="relative z-10 w-full max-w-2xl px-6 flex flex-col items-center space-y-6">
          <AnimatePresence>
            {isCtaVaultLocked ? (
              <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 20 }}
                transition={{ duration: 0.4, ease: 'easeOut' }}
                className="w-full flex flex-col items-center space-y-5"
              >
                {/* Copyable Install Box */}
                <div className="flex items-center justify-between gap-3 px-5 py-3.5 rounded-xl bg-[#19191C] border border-[#7A1F2B] font-mono text-xs text-[#EDE8DF] w-full shadow-2xl">
                  <span className="text-[#8B8B8F] select-none">$</span>
                  <span className="text-[#EDE8DF] font-bold select-all truncate">
                    go install github.com/jhanvi857/cloudweave/cmd/cweave@latest
                  </span>
                  <button
                    onClick={copyInstall}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-[#000000] hover:bg-[#7A1F2B] text-[#EDE8DF] text-xs transition-colors border border-[#26262B] cursor-pointer shrink-0 ml-2 shadow-xs"
                    title="Copy install command"
                  >
                    {copied ? <Check className="w-3.5 h-3.5 text-[#4ADE80]" /> : <Copy className="w-3.5 h-3.5" />}
                    <span className="text-xs font-semibold">{copied ? 'Copied' : 'Copy'}</span>
                  </button>
                </div>

                {/* Two Centered Action Buttons */}
                <div className="flex flex-wrap items-center justify-center gap-4 pt-1">
                  <Link
                    href="/docs"
                    className="inline-flex items-center justify-center h-12 px-8 text-xs font-mono font-semibold bg-[#7A1F2B] text-[#EDE8DF] hover:bg-[#912735] transition-all rounded-md shadow-md gap-2 cursor-pointer active:scale-98"
                  >
                    <span>Quickstart Guide</span>
                    <ArrowRight className="w-4 h-4 stroke-[2.5]" />
                  </Link>

                  <a
                    href="https://github.com/jhanvi857/CloudWeave"
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center justify-center h-12 px-7 text-xs font-mono text-[#EDE8DF] bg-[#19191C] border border-[#26262B] hover:border-[#EDE8DF] transition-colors rounded-md gap-2 shadow-xs"
                  >
                    <GithubIcon className="w-4 h-4 text-[#8B8B8F]" />
                    <span>GitHub Repository</span>
                  </a>
                </div>
              </motion.div>
            ) : (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="flex flex-col items-center"
              >
                <button
                  onClick={() => setIsCtaVaultLocked(true)}
                  className="px-6 py-3 rounded-lg bg-[#19191C] border border-[#26262B] hover:border-[#7A1F2B] text-xs font-mono text-[#EDE8DF] transition-all cursor-pointer shadow-md hover:scale-102"
                >
                  Click to Lock Cluster & Reveal Quickstart
                </button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </section>
    </div>
  )
}
