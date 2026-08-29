import React from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { DocsReferenceView } from '../../components/docs/DocsReferenceView'
import { Footer } from '../../components/layout/Footer'

export const metadata = {
  title: 'Documentation & Reference | CloudWeave Object Storage',
  description: 'Technical reference for Quickstart, S3 REST API, CLI tool (cweave), Go SDK, and mTLS configuration.',
}

export default function DocsPage() {
  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full max-w-[1380px] mx-auto px-6 py-12">
        <DocsReferenceView />
      </main>
      <Footer />
    </div>
  )
}

