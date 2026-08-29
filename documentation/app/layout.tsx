import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'CloudWeave Documentation',
  description: 'Understand distributed object storage through CloudWeave.',
}

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en" className="bg-[var(--paper)]"><body>{children}</body></html>
}
