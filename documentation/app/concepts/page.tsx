import React from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { ConceptsView } from '../../components/concepts/ConceptsView'
import { Footer } from '../../components/layout/Footer'

export const metadata = {
  title: 'Concepts & Architecture | CloudWeave Object Storage',
  description: 'Technical breakdown of FastCDC chunking, SHA-256 CAS deduplication, consistent hashing, Dynamo quorums, and Reed-Solomon erasure coding.',
}

export default function ConceptsPage() {
  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full max-w-[1380px] mx-auto px-6 py-12">
        <ConceptsView />
      </main>
      <Footer />
    </div>
  )
}


