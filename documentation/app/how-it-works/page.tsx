import React from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { Footer } from '../../components/layout/Footer'
import { Pipeline3DScroll } from '../../components/canvas/Pipeline3DScroll'

export const metadata = {
  title: 'How It Works | CloudWeave Object Storage',
  description: 'A 6-stage interactive walkthrough of the CloudWeave distributed storage pipeline: S3 Ingestion, FastCDC, SHA-256 CAS, Consistent Hashing, Dynamo Quorum, and WAL Durability.',
}

export default function HowItWorksPage() {
  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full max-w-[1380px] mx-auto px-6 py-12 space-y-10">
        {/* Header */}
        <div className="text-center space-y-3 max-w-3xl mx-auto">
          <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
            <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
            <span>Interactive 3D Pipeline</span>
          </div>
          <h1 className="text-4xl sm:text-5xl font-serif font-bold tracking-tight text-[#EDE8DF]">
            How an Object Travels Through the Cluster
          </h1>
          <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
            Follow an incoming S3 byte stream from initial SigV4 authentication, through FastCDC dynamic boundary chunking, SHA-256 CAS deduplication, 150-vnode ring placement, and decentralized quorum fsync commit.
          </p>
        </div>

        {/* 3D Pipeline Visualizer with Code Panels */}
        <Pipeline3DScroll />
      </main>
      <Footer />
    </div>
  )
}

