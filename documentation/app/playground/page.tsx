'use client'

import React, { useState } from 'react'
import { Navbar } from '../../components/layout/Navbar'
import { Footer } from '../../components/layout/Footer'
import { PlaygroundShell } from '../../components/playground/PlaygroundShell'

export default function PlaygroundPage() {
  const [activeExperiment, setActiveExperiment] = useState<'pipeline' | 'failure' | 'dedup' | 'erasure'>('pipeline')

  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full max-w-[1380px] mx-auto px-6 py-10">
        <PlaygroundShell
          activeExperiment={activeExperiment}
          setActiveExperiment={setActiveExperiment}
        />
      </main>
      <Footer />
    </div>
  )
}


