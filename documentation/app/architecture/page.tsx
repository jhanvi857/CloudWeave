import React from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { Footer } from '../../components/layout/Footer'
import { ConceptsView } from '../../components/concepts/ConceptsView'

export const metadata = {
  title: 'Architecture | CloudWeave Object Storage',
  description: 'Deep architectural overview of the CloudWeave distributed storage engine.',
}

export default function ArchitecturePage() {
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
