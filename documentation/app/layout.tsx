import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = { title: 'CloudWeave Docs', description: 'Distributed object storage, explained and observable.' }

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en" className="bg-[#f7f9fc]"><body>{children}</body></html>
}
