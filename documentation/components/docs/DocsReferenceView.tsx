'use client'

import React, { useState } from 'react'
import { CodeBlock } from '../ui/CodeBlock'
import {
  Terminal,
  Server,
  Key,
  ShieldCheck,
  HardDrive,
  Cpu,
  BookOpen,
  ArrowRight,
  Check,
  Copy,
} from 'lucide-react'

interface DocSection {
  id: string
  title: string
  category: string
  icon: any
}

const docSections: DocSection[] = [
  { id: 'quickstart', title: '1. Quickstart & Deployment', category: 'GETTING STARTED', icon: Terminal },
  { id: 'cli', title: '2. CLI Tool (cweave)', category: 'REFERENCE', icon: Terminal },
  { id: 's3-api', title: '3. S3 REST & SigV4 API', category: 'REFERENCE', icon: HardDrive },
  { id: 'go-sdk', title: '4. Go Client SDK', category: 'LIBRARIES', icon: Server },
  { id: 'mtls-security', title: '5. mTLS & Security', category: 'OPERATIONS', icon: ShieldCheck },
]

export function DocsReferenceView() {
  const [activeSectionId, setActiveSectionId] = useState<string>('quickstart')

  return (
    <div className="w-full flex flex-col md:flex-row gap-8 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Sidebar Nav (Zero Motion, Fast) */}
      <aside className="w-full md:w-64 shrink-0 space-y-4">
        <div className="p-4 rounded-xl border border-[#26262B] bg-[#19191C] space-y-3 sticky top-24">
          <div className="font-mono text-[11px] font-bold text-[#8B8B8F] uppercase tracking-wider">
            Documentation Index
          </div>

          <nav className="space-y-1 font-mono text-xs">
            {docSections.map((sec) => {
              const isSelected = activeSectionId === sec.id
              const Icon = sec.icon
              return (
                <button
                  key={sec.id}
                  onClick={() => setActiveSectionId(sec.id)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded text-left transition-colors cursor-pointer ${isSelected
                      ? 'bg-[#000000] border border-[#7A1F2B] text-[#EDE8DF] font-bold shadow-xs'
                      : 'text-[#8B8B8F] hover:text-[#EDE8DF] hover:bg-[#222226]'
                    }`}
                >
                  <Icon className={`w-3.5 h-3.5 shrink-0 ${isSelected ? 'text-[#7A1F2B]' : 'text-[#A6A9AE]'}`} />
                  <span className="truncate">{sec.title}</span>
                </button>
              )
            })}
          </nav>
        </div>
      </aside>

      {/* Main Documentation Content Area */}
      <main className="flex-1 min-w-0 space-y-10">
        {/* ========================================================================= */}
        {/* 1. QUICKSTART                                                            */}
        {/* ========================================================================= */}
        {activeSectionId === 'quickstart' && (
          <article className="space-y-6">
            <div className="border-b border-[#26262B] pb-4 space-y-1">
              <span className="text-xs font-mono text-[#A6A9AE] font-semibold">GETTING STARTED</span>
              <h1 className="text-3xl font-serif font-bold text-[#EDE8DF]">Quickstart & Cluster Setup</h1>
              <p className="text-xs font-mono text-[#8B8B8F]">
                Run CloudWeave as a standalone developer node or as a 5-node distributed cluster in seconds.
              </p>
            </div>

            <div className="space-y-4 font-mono text-xs">
              <h3 className="text-sm font-bold text-[#EDE8DF]">Option A: Standalone Developer Node</h3>
              <p className="text-xs font-sans text-[#8B8B8F]">
                Compiles the single Go binary with zero CGO dependencies and starts storage gateway on port 8080.
              </p>
              <CodeBlock
                code={`# 1. Install CLI and Node binary
go install github.com/jhanvi857/cloudweave/cmd/node@latest
go install github.com/jhanvi857/cloudweave/cmd/cweave@latest

# 2. Start standalone storage node
node --port 8080 --data ./data-node1 --addr localhost:8080

# 3. Put and Get an object via cweave CLI
cweave put video.mp4 --endpoint http://localhost:8080
cweave get video.mp4 --output downloaded.mp4 --endpoint http://localhost:8080`}
                language="bash"
                filename="Terminal"
              />
            </div>

            <div className="space-y-4 font-mono text-xs pt-4 border-t border-[#26262B]">
              <h3 className="text-sm font-bold text-[#EDE8DF]">Option B: 5-Node Quorum Cluster with Docker Compose</h3>
              <p className="text-xs font-sans text-[#8B8B8F]">
                Spins up 5 containerized storage nodes connected via consistent hash ring with gossip failure detection.
              </p>
              <CodeBlock
                code={`# Clone repo & boot cluster
git clone https://github.com/jhanvi857/CloudWeave.git
cd CloudWeave

# Spin up 5 storage nodes + gateway
docker-compose up -d

# Verify cluster status
cweave cluster status --endpoint http://localhost:8080`}
                language="bash"
                filename="Terminal"
              />
            </div>
          </article>
        )}

        {/* ========================================================================= */}
        {/* 2. CLI TOOL                                                              */}
        {/* ========================================================================= */}
        {activeSectionId === 'cli' && (
          <article className="space-y-6">
            <div className="border-b border-[#26262B] pb-4 space-y-1">
              <span className="text-xs font-mono text-[#A6A9AE] font-semibold">REFERENCE</span>
              <h1 className="text-3xl font-serif font-bold text-[#EDE8DF]">Command-Line Interface (cweave)</h1>
              <p className="text-xs font-mono text-[#8B8B8F]">
                Complete CLI command reference for managing objects, verifying hashes, and monitoring cluster health.
              </p>
            </div>

            <div className="space-y-6 font-mono text-xs">
              <div className="p-4 rounded-lg bg-[#19191C] border border-[#26262B] space-y-3">
                <span className="font-bold text-[#EDE8DF] block">cweave put &lt;file&gt;</span>
                <p className="font-sans text-xs text-[#8B8B8F]">
                  Splits local file using FastCDC, generates SHA-256 CAS IDs, and writes chunks with N=3 quorum.
                </p>
                <CodeBlock
                  code={`cweave put ./large_dataset.parquet --bucket warehouse --endpoint http://localhost:8080`}
                  language="bash"
                />
              </div>

              <div className="p-4 rounded-lg bg-[#19191C] border border-[#26262B] space-y-3">
                <span className="font-bold text-[#EDE8DF] block">cweave get &lt;key&gt;</span>
                <p className="font-sans text-xs text-[#8B8B8F]">
                  Queries R=2 read quorum, verifies CAS checksums, reassembles chunk stream, and writes to disk.
                </p>
                <CodeBlock
                  code={`cweave get large_dataset.parquet --output ./restored.parquet --endpoint http://localhost:8080`}
                  language="bash"
                />
              </div>

              <div className="p-4 rounded-lg bg-[#19191C] border border-[#26262B] space-y-3">
                <span className="font-bold text-[#EDE8DF] block">cweave cluster status</span>
                <p className="font-sans text-xs text-[#8B8B8F]">
                  Inspects live ring membership, active vnodes, heartbeat latency, and replication status.
                </p>
                <CodeBlock
                  code={`cweave cluster status --endpoint http://localhost:8080`}
                  language="bash"
                />
              </div>
            </div>
          </article>
        )}

        {/* ========================================================================= */}
        {/* 3. S3 REST & SIGV4 API                                                   */}
        {/* ========================================================================= */}
        {activeSectionId === 's3-api' && (
          <article className="space-y-6">
            <div className="border-b border-[#26262B] pb-4 space-y-1">
              <span className="text-xs font-mono text-[#A6A9AE] font-semibold">REST SPECIFICATION</span>
              <h1 className="text-3xl font-serif font-bold text-[#EDE8DF]">AWS S3 REST & SigV4 API</h1>
              <p className="text-xs font-mono text-[#8B8B8F]">
                Standard S3 endpoints compatible with AWS CLI, boto3, MinIO Client, and official AWS SDKs.
              </p>
            </div>

            <div className="space-y-6 font-mono text-xs">
              <div className="p-4 rounded-lg bg-[#19191C] border border-[#26262B] space-y-3">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 rounded bg-[#7A1F2B] text-[#EDE8DF] font-bold">PUT</span>
                  <span className="text-[#EDE8DF] font-semibold">/&#123;bucket&#125;/&#123;key&#125;</span>
                </div>
                <p className="font-sans text-xs text-[#8B8B8F]">
                  Uploads an object stream. Returns 200 OK with ETag (SHA-256 CAS manifest hash).
                </p>
                <CodeBlock
                  code={`curl -X PUT http://localhost:8080/my-bucket/video.mp4 \\
  -H "Authorization: AWS4-HMAC-SHA256 Credential=cw-access-key/..." \\
  -H "x-amz-content-sha256: UNSIGNED-PAYLOAD" \\
  --data-binary @video.mp4`}
                  language="bash"
                />
              </div>

              <div className="p-4 rounded-lg bg-[#19191C] border border-[#26262B] space-y-3">
                <div className="flex items-center gap-2">
                  <span className="px-1.5 py-0.5 rounded bg-[#26262B] text-[#EDE8DF] font-bold">GET</span>
                  <span className="text-[#EDE8DF] font-semibold">/&#123;bucket&#125;/&#123;key&#125;</span>
                </div>
                <p className="font-sans text-xs text-[#8B8B8F]">
                  Retrieves an object stream with parallel chunk reassembly.
                </p>
                <CodeBlock
                  code={`curl -X GET http://localhost:8080/my-bucket/video.mp4 \\
  -H "Authorization: AWS4-HMAC-SHA256 Credential=cw-access-key/..." \\
  --output video.mp4`}
                  language="bash"
                />
              </div>
            </div>
          </article>
        )}

        {/* ========================================================================= */}
        {/* 4. GO CLIENT SDK                                                         */}
        {/* ========================================================================= */}
        {activeSectionId === 'go-sdk' && (
          <article className="space-y-6">
            <div className="border-b border-[#26262B] pb-4 space-y-1">
              <span className="text-xs font-mono text-[#A6A9AE] font-semibold">LIBRARIES</span>
              <h1 className="text-3xl font-serif font-bold text-[#EDE8DF]">Go Client SDK</h1>
              <p className="text-xs font-mono text-[#8B8B8F]">
                Programmatic Go API with automatic FastCDC client-side chunking, retries, and pooled keep-alive connections.
              </p>
            </div>

            <div className="space-y-4 font-mono text-xs">
              <CodeBlock
                code={`package main

import (
    "context"
    "fmt"
    "os"
    "github.com/jhanvi857/cloudweave/client"
)

func main() {
    ctx := context.Background()
    
    // Initialize CloudWeave client
    c, err := client.New(client.Config{
        Endpoint: "http://localhost:8080",
        AccessKey: "cw-root",
        SecretKey: "cw-secret-key-123",
    })
    if err != nil {
        panic(err)
    }

    // Upload file with FastCDC & Quorum
    f, _ := os.Open("photo.jpg")
    defer f.Close()

    res, err := c.PutObject(ctx, "media", "photo.jpg", f)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Uploaded: ETag=%s, Chunks=%d\\n", res.ETag, res.ChunkCount)
}`}
                language="go"
                filename="main.go"
              />
            </div>
          </article>
        )}

        {/* ========================================================================= */}
        {/* 5. MTLS & SECURITY                                                       */}
        {/* ========================================================================= */}
        {activeSectionId === 'mtls-security' && (
          <article className="space-y-6">
            <div className="border-b border-[#26262B] pb-4 space-y-1">
              <span className="text-xs font-mono text-[#A6A9AE] font-semibold">OPERATIONS</span>
              <h1 className="text-3xl font-serif font-bold text-[#EDE8DF]">Mutual TLS (mTLS) & Zero-Trust</h1>
              <p className="text-xs font-mono text-[#8B8B8F]">
                Peer-to-peer authenticated transport ensuring all node-to-node chunk transfers are encrypted and verified.
              </p>
            </div>

            <div className="space-y-4 font-mono text-xs">
              <p className="font-sans text-xs text-[#8B8B8F] leading-relaxed">
                All internal RPC and chunk replication streams between cluster nodes run over TLS 1.3 with mutual certificate verification. Persistent pooled connections prevent per-request TLS handshake latency overhead.
              </p>
              <CodeBlock
                code={`// Start node with mTLS enabled
node --port 8080 \\
  --tls-cert /etc/cloudweave/certs/node1.crt \\
  --tls-key /etc/cloudweave/certs/node1.key \\
  --tls-ca /etc/cloudweave/certs/cluster-ca.crt`}
                language="bash"
                filename="Terminal"
              />
            </div>
          </article>
        )}
      </main>
    </div>
  )
}
