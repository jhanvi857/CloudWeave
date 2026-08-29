import React from 'react'
import { Navbar } from '../components/layout/Navbar'
import { LandingPage } from '../components/landing/LandingPage'
import { Footer } from '../components/layout/Footer'

export default function Home() {
  return (
    <div className="min-h-screen bg-[#000000] text-[#EDE8DF] flex flex-col font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      <Navbar />
      <main className="flex-1 w-full">
        <LandingPage />
      </main>
      <Footer />
    </div>
  )
}


