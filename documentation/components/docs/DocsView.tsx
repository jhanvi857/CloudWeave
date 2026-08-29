'use client'

import React, { useState } from 'react'
import { CodeBlock } from '../ui/CodeBlock'
import { Terminal, HardDrive, FileCode, Server, Sliders } from 'lucide-react'

interface DocSection {
  id: string
  label: string
  icon: any
}

export function DocsView() {
  const [activeDoc, setActiveDoc] = useState<string>('quickstart')

  const sections: DocSection[] = [
    { id: 'quickstart', label: 'Quickstart (Docker)', icon: Terminal },
    { id: 's3-api', label: 'S3 REST Endpoints', icon: Server },
    { id: 'aws-cli', label: 'AWS CLI Guide', icon: Terminal },
    { id: 'boto3', label: 'Python (boto3)', icon: FileCode },
    { id: 'go-sdk', label: 'Native Go SDK', icon: FileCode },
    { id: 'cweave-cli', label: 'cweave CLI Tool', icon: Terminal },
    { id: 'config', label: 'Configuration & Env', icon: Sliders },
  ]

  return (
    <div className="w-full max-w-4xl mx-auto space-y-10 py-4 font-sans selection:bg-[#7A1F2B]/40 selection:text-[#EDE8DF]">
      {/* Header */}
      <div className="text-center space-y-3 max-w-2xl mx-auto">
        <div className="inline-flex items-center gap-2 px-3 py-1 text-xs font-mono border border-[#26262B] rounded-full bg-[#19191C] text-[#A6A9AE]">
          <span className="w-2 h-2 rounded-full bg-[#7A1F2B]" />
          <span>API Documentation</span>
        </div>
        <h1 className="text-4xl font-serif font-bold tracking-tight text-[#EDE8DF]">
          API, SDKs & Integration Guides
        </h1>
        <p className="text-sm font-mono text-[#8B8B8F] leading-relaxed">
          Complete technical reference for running CloudWeave and integrating with standard S3 tools.
        </p>
      </div>

      {/* Doc Section Switcher */}
      <div className="flex items-center gap-2 overflow-x-auto pb-3 border-b border-[#26262B] font-mono text-xs">
        {sections.map((s) => {
          const isSelected = activeDoc === s.id
          const Icon = s.icon

          return (
            <button
              key={s.id}
              onClick={() => setActiveDoc(s.id)}
              className={`flex items-center gap-2 px-3.5 py-2 rounded-lg shrink-0 font-medium transition-all cursor-pointer ${
                isSelected
                  ? 'bg-[#7A1F2B] text-[#EDE8DF] font-bold shadow-xs'
                  : 'bg-[#19191C] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF] hover:bg-[#222226]'
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              <span>{s.label}</span>
            </button>
          )
        })}
      </div>

      {/* Selected Doc Article */}
      <article className="space-y-6 text-sm text-[#8B8B8F] leading-relaxed">
        {activeDoc === 'quickstart' && (
          <div className="space-y-4">
            <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">Quickstart with Docker & Compose</h2>
            <p>
              CloudWeave is distributed as a lightweight, multi-architecture static binary on GHCR (`linux/amd64` and `linux/arm64`).
            </p>

            <h3 className="font-bold text-[#EDE8DF] pt-2">Option A: Run Standalone Instance</h3>
            <CodeBlock
              code={`# Pull prebuilt image from GHCR
docker pull ghcr.io/jhanvi857/cloudweave:latest

# Run container on port 9000
docker run -d \\
  --name cloudweave \\
  -p 9000:9000 \\
  --memory=256m \\
  -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" \\
  -v cloudweave_data:/data \\
  ghcr.io/jhanvi857/cloudweave:latest`}
              language="bash"
              filename="standalone.sh"
            />

            <h3 className="font-bold text-[#EDE8DF] pt-2">Option B: 5-Node Distributed Mesh Cluster</h3>
            <CodeBlock
              code={`# Spin up 5 node cluster with automatic quorum and failure detection
docker compose up -d

# Check live cluster status
docker compose logs -f`}
              language="bash"
              filename="cluster.sh"
            />
          </div>
        )}

        {activeDoc === 's3-api' && (
          <div className="space-y-4">
            <h2 className="text-2xl font-serif font-bold text-[#EDE8DF]">AWS S3 REST Compatibility</h2>
            <p>
              CloudWeave implements standard S3 REST endpoints with SigV4 signature verification.
            </p>
            <CodeBlock
              code={`# Upload an object
curl -X PUT http://localhost:9000/bucket/file.mp4 \\
  -H "Authorization: AWS4-HMAC-SHA256 ..." \\
  --data-binary @file.mp4`}
              language="bash"
              filename="curl-put.sh"
            />
          </div>
        )}
      </article>
    </div>
  )
}
