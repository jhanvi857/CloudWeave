import React from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { BenchmarksView } from '../../components/benchmarks/BenchmarksView'
import { Footer } from '../../components/layout/Footer'

export const metadata = {
  title: 'Empirical Benchmarks | CloudWeave Object Storage',
  description: 'Performance, IOPS, and memory scaling benchmarks for CloudWeave vs MinIO, Ceph, and SeaweedFS.',
}

export default function BenchmarksPage() {
  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full max-w-[1380px] mx-auto px-6 py-12">
        <BenchmarksView />
      </main>
      <Footer />
    </div>
  )
}


