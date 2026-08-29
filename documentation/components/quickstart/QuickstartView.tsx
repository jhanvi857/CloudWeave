'use client'

import React, { useState } from 'react'
import { Terminal, HardDrive, FileCode, CheckCircle } from 'lucide-react'
import { CodeBlock } from '../ui/CodeBlock'

export function QuickstartView() {
  const [tab, setTab] = useState<'docker' | 'compose' | 'aws-cli' | 'boto3' | 'go-sdk'>('docker')

  return (
    <div className="space-y-6 font-sans">
      {/* Title */}
      <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C]">
        <div className="flex items-center gap-2">
          <span className="rounded px-2.5 py-1 text-[10px] font-mono font-bold text-[#A6A9AE] bg-[#000000] border border-[#26262B] uppercase">
            Quick Start Guide
          </span>
          <span className="text-xs font-mono text-[#8B8B8F]">Local Cluster Setup</span>
        </div>
        <h2 className="text-2xl font-serif font-bold tracking-tight text-[#EDE8DF] mt-2">
          Get Started in 5 Minutes
        </h2>
        <p className="text-xs sm:text-sm font-mono text-[#8B8B8F] mt-1 max-w-2xl">
          Spin up a local CloudWeave node or 5-node distributed cluster, and connect using standard AWS CLI, boto3, or the native Go SDK.
        </p>

        {/* Tab Buttons */}
        <div className="flex flex-wrap gap-2 mt-4 pt-3 border-t border-[#26262B] font-mono text-xs">
          <button
            onClick={() => setTab('docker')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${tab === 'docker'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            Docker Run
          </button>
          <button
            onClick={() => setTab('compose')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${tab === 'compose'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            Docker Compose (5 Nodes)
          </button>
          <button
            onClick={() => setTab('aws-cli')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${tab === 'aws-cli'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            AWS CLI
          </button>
          <button
            onClick={() => setTab('boto3')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${tab === 'boto3'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            Python (boto3)
          </button>
          <button
            onClick={() => setTab('go-sdk')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${tab === 'go-sdk'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            Go SDK
          </button>
        </div>
      </div>

      {/* Tab Panels */}
      {tab === 'docker' && (
        <div className="space-y-4 font-mono text-xs">
          <CodeBlock
            code={`# Pull & Run CloudWeave Standalone
docker run -d \\
  --name cloudweave \\
  -p 9000:9000 \\
  -e CLOUDWEAVE_API_KEYS="root-key=admin-secret" \\
  -v /tmp/cloudweave-data:/data \\
  ghcr.io/jhanvi857/cloudweave:latest`}
            language="bash"
            filename="Run Standalone"
          />
        </div>
      )}

      {tab === 'compose' && (
        <div className="space-y-4 font-mono text-xs">
          <CodeBlock
            code={`# Start 5-Node Quorum Mesh
git clone https://github.com/jhanvi857/CloudWeave.git
cd CloudWeave
docker compose up -d

# Verify Ring Status
docker compose logs -f node1`}
            language="bash"
            filename="Docker Compose"
          />
        </div>
      )}

      {tab === 'aws-cli' && (
        <div className="space-y-4 font-mono text-xs">
          <CodeBlock
            code={`# Configure S3 endpoint
aws configure set aws_access_key_id root-key
aws configure set aws_secret_access_key admin-secret

# Upload & Download
aws --endpoint-url=http://localhost:9000 s3 cp dataset.tar s3://backups/dataset.tar
aws --endpoint-url=http://localhost:9000 s3 cp s3://backups/dataset.tar ./restored.tar`}
            language="bash"
            filename="AWS CLI S3"
          />
        </div>
      )}

      {tab === 'boto3' && (
        <div className="space-y-4 font-mono text-xs">
          <CodeBlock
            code={`import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='root-key',
    aws_secret_access_key='admin-secret'
)

# Upload object with FastCDC and Quorum
with open('photo.png', 'rb') as f:
    s3.put_object(Bucket='media', Key='photo.png', Body=f)`}
            language="python"
            filename="boto3_example.py"
          />
        </div>
      )}

      {tab === 'go-sdk' && (
        <div className="space-y-4 font-mono text-xs">
          <CodeBlock
            code={`package main

import (
    "context"
    "os"
    "github.com/jhanvi857/cloudweave/client"
)

func main() {
    c, _ := client.New(client.Config{
        Endpoint:  "http://localhost:9000",
        AccessKey: "root-key",
        SecretKey: "admin-secret",
    })
    
    f, _ := os.Open("data.bin")
    defer f.Close()
    
    c.PutObject(context.Background(), "bucket", "data.bin", f)
}`}
            language="go"
            filename="main.go"
          />
        </div>
      )}
    </div>
  )
}
