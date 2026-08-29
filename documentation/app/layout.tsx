import type { Metadata } from 'next'
import { Inter, IBM_Plex_Mono, Fraunces } from 'next/font/google'
import './globals.css'

const inter = Inter({
  subsets: ['latin'],
  variable: '--font-sans',
  display: 'swap',
})

const ibmPlexMono = IBM_Plex_Mono({
  weight: ['400', '500', '600', '700'],
  subsets: ['latin'],
  variable: '--font-mono',
  display: 'swap',
})

const fraunces = Fraunces({
  subsets: ['latin'],
  variable: '--font-serif',
  display: 'swap',
})

export const metadata: Metadata = {
  title: 'CloudWeave | Deterministic Distributed Storage in Go',
  description: 'A distributed S3-compatible object store built like a bank vault. FastCDC chunking, SHA-256 CAS deduplication, consistent hash ring, N/W/R quorum, and self-healing repair in Go.',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en" className={`${inter.variable} ${ibmPlexMono.variable} ${fraunces.variable} dark`} suppressHydrationWarning>
      <body className="min-h-screen bg-[#000000] text-[#EDE8DF] font-sans antialiased selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]" suppressHydrationWarning>
        {children}
      </body>
    </html>
  )
}


