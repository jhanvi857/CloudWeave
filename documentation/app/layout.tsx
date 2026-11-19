import { RootProvider } from 'fumadocs-ui/provider/next';
import localFont from 'next/font/local';
import './global.css';
import './theme.css';
import type { Metadata } from 'next';

const geistSans = localFont({
  src: './fonts/geist-sans.woff2',
  variable: '--font-sans',
  display: 'swap',
  weight: '400 500 600',
});

const geistMono = localFont({
  src: './fonts/geist-mono.woff2',
  variable: '--font-mono',
  display: 'swap',
  weight: '400 500 600',
});

export const metadata: Metadata = {
  metadataBase: new URL('https://cloudweave.dev'),
  title: {
    template: '%s | CloudWeave Documentation',
    default: 'CloudWeave | Distributed S3-Compatible Object Store in Go',
  },
  description: 'Technical documentation for CloudWeave, a high-performance distributed object storage engine written in Go.',
};

export default function Layout({ children }: LayoutProps<'/'>) {
  return (
    <html lang="en" suppressHydrationWarning className={`${geistSans.variable} ${geistMono.variable} dark`}>
      <head />
      <body className="flex flex-col min-h-screen bg-bg text-muted font-sans antialiased selection:bg-brand-soft selection:text-brand">
        <RootProvider>{children}</RootProvider>
      </body>
    </html>
  );
}
