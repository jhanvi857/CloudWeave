'use client'

import React, { useState } from 'react'
import { motion } from 'framer-motion'
import { CodeBlock } from '../ui/CodeBlock'
import {
  Terminal,
  Server,
  ShieldCheck,
  Zap,
  HardDrive,
  Copy,
  Check,
  ArrowRight,
  Layers,
  Cpu,
  Lock,
  Sparkles,
} from 'lucide-react'

export function GetStartedView() {
  const [copiedOneLiner, setCopiedOneLiner] = useState(false)
  const [clusterSize, setClusterSize] = useState<'1-node' | '3-node' | '5-node'>('5-node')
  const [storageMode, setStorageMode] = useState<'replication' | 'erasure'>('replication')
  const [enableMTLS, setEnableMTLS] = useState<boolean>(true)
  const [sdkTab, setSdkTab] = useState<'aws-cli' | 'python' | 'go' | 'cweave'>('aws-cli')

  const handleCopyOneLiner = () => {
    navigator.clipboard.writeText('docker run -d -p 9000:9000 -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" ghcr.io/jhanvi857/cloudweave:latest')
    setCopiedOneLiner(true)
    setTimeout(() => setCopiedOneLiner(false), 2000)
  }

  // Dynamic Compose Generator
  const generateCompose = () => {
    if (clusterSize === '1-node') {
      return `version: '3.8'
services:
  cloudweave:
    image: ghcr.io/jhanvi857/cloudweave:latest
    container_name: cloudweave-node
    ports:
      - "9000:9000"
    environment:
      - PORT=9000
      - DATA_DIR=/data
      - CLOUDWEAVE_API_KEYS=master-secret-key=admin
    volumes:
      - cw_data:/data
    restart: unless-stopped

volumes:
  cw_data:`
    }

    const nodeCount = clusterSize === '3-node' ? 3 : 5
    const peers = Array.from({ length: nodeCount })
      .map((_, i) => `http://node${i + 1}:${9000 + i}`)
      .join(',')

    return `version: '3.8'
services:
${Array.from({ length: nodeCount })
  .map(
    (_, i) => `  node${i + 1}:
    image: ghcr.io/jhanvi857/cloudweave:latest
    container_name: cloudweave-node${i + 1}
    ports:
      - "${9000 + i}:${9000 + i}"
    environment:
      - PORT=${9000 + i}
      - DATA_DIR=/data/node${i + 1}
      - PEERS=${peers}
      - QUORUM_N=${nodeCount === 3 ? 3 : 5}
      - QUORUM_W=${nodeCount === 3 ? 2 : 3}
      - QUORUM_R=${nodeCount === 3 ? 2 : 3}
      - MTLS_ENABLED=${enableMTLS}
      - STORAGE_ENGINE=${storageMode === 'replication' ? 'cas_replication' : 'reed_solomon_k4_m2'}
      - CLOUDWEAVE_API_KEYS=master-secret-key=admin
    volumes:
      - node${i + 1}_data:/data/node${i + 1}
    restart: unless-stopped`
  )
  .join('\n\n')}

volumes:
${Array.from({ length: nodeCount })
  .map((_, i) => `  node${i + 1}_data:`)
  .join('\n')}`
  }

  return (
    <div className="w-full max-w-5xl mx-auto space-y-16 py-6 font-sans">
      {/* 1. Animated Hero Section */}
      <div className="text-center space-y-4 max-w-3xl mx-auto">
        <motion.div
          initial={{ opacity: 0, scale: 0.95 }}
          animate={{ opacity: 1, scale: 1 }}
          className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full border border-[#90E0EF] bg-[#CAF0F8] text-[11px] font-mono font-bold text-[#0E7FB4]"
        >
          <Sparkles className="w-3.5 h-3.5" />
          <span>PRODUCTION-READY DISTRIBUTED S3 STORAGE</span>
        </motion.div>

        <motion.h1
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          className="text-4xl sm:text-5xl font-extrabold tracking-tight text-[#03045E] leading-tight"
        >
          Deploy CloudWeave <span className="text-[#0E7FB4]">in Seconds.</span>
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="text-base text-[#64748B] max-w-2xl mx-auto leading-relaxed"
        >
          Zero cloud lock-in. Full Amazon S3 API compatibility. FastCDC rolling chunking, consistent hashing, quorum replication, and automatic self-healing.
        </motion.p>

        {/* 1-Click Copy Box */}
        <motion.div
          initial={{ opacity: 0, y: 15 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.15 }}
          className="pt-2 max-w-2xl mx-auto"
        >
          <div className="flex items-center justify-between p-3 rounded-2xl border-2 border-[#0E7FB4] bg-[#F8FAFC] shadow-md font-mono text-xs text-left">
            <div className="flex items-center gap-2.5 overflow-x-auto text-[#03045E]">
              <span className="text-[#0E7FB4] font-bold select-none">$</span>
              <span className="truncate">
                docker run -d -p 9000:9000 ghcr.io/jhanvi857/cloudweave:latest
              </span>
            </div>
            <button
              onClick={handleCopyOneLiner}
              className="flex items-center gap-1.5 shrink-0 px-3 py-1.5 rounded-xl bg-[#0E7FB4] text-white font-bold hover:bg-[#03045E] transition-colors"
            >
              {copiedOneLiner ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
              <span>{copiedOneLiner ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
        </motion.div>
      </div>

      {/* 2. Key Architecture Pillars Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs space-y-2 hover:border-[#0E7FB4] transition-all">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#CAF0F8] text-[#0E7FB4]">
            <Server className="w-5 h-5" />
          </div>
          <h3 className="font-bold text-base text-[#03045E]">S3 REST API & SigV4</h3>
          <p className="text-xs text-[#64748B] leading-relaxed">
            Drop-in compatibility with AWS CLI, boto3, MinIO Client, and standard S3 SDKs. Works out of the box with zero custom driver requirements.
          </p>
        </div>

        <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs space-y-2 hover:border-[#0E7FB4] transition-all">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#E8F8F0] text-[#22A06B]">
            <Zap className="w-5 h-5" />
          </div>
          <h3 className="font-bold text-base text-[#03045E]">FastCDC & CAS Dedup</h3>
          <p className="text-xs text-[#64748B] leading-relaxed">
            Rolling Gear hash isolates byte boundary shifts. SHA-256 addresses eliminate duplicate disk blocks, achieving 485 MB/s deduplicated uploads.
          </p>
        </div>

        <div className="p-6 rounded-2xl border border-[#CBD5E1] bg-white shadow-xs space-y-2 hover:border-[#0E7FB4] transition-all">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#CAF0F8] text-[#0E7FB4]">
            <ShieldCheck className="w-5 h-5" />
          </div>
          <h3 className="font-bold text-base text-[#03045E]">Self-Healing Anti-Entropy</h3>
          <p className="text-xs text-[#64748B] leading-relaxed">
            Sub-second failure detection with 500ms gossip pings. Background replica repair workers restore under-replicated chunks automatically.
          </p>
        </div>
      </div>

      {/* 3. Interactive Deployment Config Builder */}
      <div className="p-6 sm:p-8 rounded-3xl border border-[#CBD5E1] bg-white shadow-sm space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-[#E2E8F0] pb-4">
          <div>
            <span className="text-[10px] font-mono font-bold uppercase text-[#0E7FB4]">
              Interactive Cluster Configurator
            </span>
            <h2 className="text-xl font-bold text-[#03045E]">
              Generate Your Production Cluster Setup
            </h2>
          </div>
          <span className="text-xs font-mono text-[#64748B]">
            Outputs clean docker-compose.yml
          </span>
        </div>

        {/* Config Options Row */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 font-mono text-xs">
          {/* Cluster Size */}
          <div className="space-y-1.5">
            <span className="text-[#64748B] font-bold text-[11px] block uppercase">Cluster Topology</span>
            <div className="flex gap-1 bg-[#F8FAFC] p-1 rounded-xl border border-[#E2E8F0]">
              {(['1-node', '3-node', '5-node'] as const).map((size) => (
                <button
                  key={size}
                  onClick={() => setClusterSize(size)}
                  className={`flex-1 py-1.5 rounded-lg font-bold transition-all ${
                    clusterSize === size
                      ? 'bg-[#0E7FB4] text-white shadow-2xs'
                      : 'text-[#64748B] hover:text-[#03045E]'
                  }`}
                >
                  {size}
                </button>
              ))}
            </div>
          </div>

          {/* Storage Mode */}
          <div className="space-y-1.5">
            <span className="text-[#64748B] font-bold text-[11px] block uppercase">Durability Mode</span>
            <div className="flex gap-1 bg-[#F8FAFC] p-1 rounded-xl border border-[#E2E8F0]">
              <button
                onClick={() => setStorageMode('replication')}
                className={`flex-1 py-1.5 rounded-lg font-bold transition-all ${
                  storageMode === 'replication'
                    ? 'bg-[#0E7FB4] text-white shadow-2xs'
                    : 'text-[#64748B] hover:text-[#03045E]'
                }`}
              >
                Replication (N=3)
              </button>
              <button
                onClick={() => setStorageMode('erasure')}
                className={`flex-1 py-1.5 rounded-lg font-bold transition-all ${
                  storageMode === 'erasure'
                    ? 'bg-[#0E7FB4] text-white shadow-2xs'
                    : 'text-[#64748B] hover:text-[#03045E]'
                }`}
              >
                Erasure (K=4, M=2)
              </button>
            </div>
          </div>

          {/* Mutual TLS Toggle */}
          <div className="space-y-1.5">
            <span className="text-[#64748B] font-bold text-[11px] block uppercase">Transport Security</span>
            <div className="flex gap-1 bg-[#F8FAFC] p-1 rounded-xl border border-[#E2E8F0]">
              <button
                onClick={() => setEnableMTLS(true)}
                className={`flex-1 py-1.5 rounded-lg font-bold transition-all ${
                  enableMTLS
                    ? 'bg-[#0E7FB4] text-white shadow-2xs'
                    : 'text-[#64748B] hover:text-[#03045E]'
                }`}
              >
                mTLS Enabled
              </button>
              <button
                onClick={() => setEnableMTLS(false)}
                className={`flex-1 py-1.5 rounded-lg font-bold transition-all ${
                  !enableMTLS
                    ? 'bg-[#0E7FB4] text-white shadow-2xs'
                    : 'text-[#64748B] hover:text-[#03045E]'
                }`}
              >
                Plain HTTP
              </button>
            </div>
          </div>
        </div>

        {/* Live Generated Compose Code */}
        <div>
          <CodeBlock
            code={generateCompose()}
            language="yaml"
            filename="docker-compose.yml"
          />
        </div>
      </div>

      {/* 4. Client Integration Tabs */}
      <div className="space-y-4">
        <div className="text-center space-y-1">
          <h2 className="text-2xl font-bold text-[#03045E]">Connect with Your Favorite Tool</h2>
          <p className="text-xs text-[#64748B]">CloudWeave works with any tool that speaks S3.</p>
        </div>

        <div className="flex justify-center">
          <div className="inline-flex gap-1.5 p-1 rounded-2xl bg-[#F8FAFC] border border-[#E2E8F0] font-mono text-xs">
            <button
              onClick={() => setSdkTab('aws-cli')}
              className={`px-3.5 py-1.5 rounded-xl font-bold transition-all ${
                sdkTab === 'aws-cli' ? 'bg-[#0E7FB4] text-white shadow-xs' : 'text-[#64748B] hover:text-[#03045E]'
              }`}
            >
              AWS CLI
            </button>
            <button
              onClick={() => setSdkTab('python')}
              className={`px-3.5 py-1.5 rounded-xl font-bold transition-all ${
                sdkTab === 'python' ? 'bg-[#0E7FB4] text-white shadow-xs' : 'text-[#64748B] hover:text-[#03045E]'
              }`}
            >
              Python (boto3)
            </button>
            <button
              onClick={() => setSdkTab('go')}
              className={`px-3.5 py-1.5 rounded-xl font-bold transition-all ${
                sdkTab === 'go' ? 'bg-[#0E7FB4] text-white shadow-xs' : 'text-[#64748B] hover:text-[#03045E]'
              }`}
            >
              Go Client
            </button>
            <button
              onClick={() => setSdkTab('cweave')}
              className={`px-3.5 py-1.5 rounded-xl font-bold transition-all ${
                sdkTab === 'cweave' ? 'bg-[#0E7FB4] text-white shadow-xs' : 'text-[#64748B] hover:text-[#03045E]'
              }`}
            >
              cweave CLI
            </button>
          </div>
        </div>

        <div className="max-w-3xl mx-auto">
          {sdkTab === 'aws-cli' && (
            <CodeBlock
              code={`# Upload file to local CloudWeave
aws s3 cp video.mp4 s3://media/video.mp4 --endpoint-url http://localhost:9000

# Download back
aws s3 cp s3://media/video.mp4 local_copy.mp4 --endpoint-url http://localhost:9000`}
              language="bash"
            />
          )}

          {sdkTab === 'python' && (
            <CodeBlock
              code={`import boto3

s3 = boto3.client(
    's3',
    endpoint_url='http://localhost:9000',
    aws_access_key_id='master-secret-key',
    aws_secret_access_key='admin'
)

# Put object
s3.put_object(Bucket='datasets', Key='train.csv', Body=open('train.csv', 'rb'))`}
              language="python"
            />
          )}

          {sdkTab === 'go' && (
            <CodeBlock
              code={`c := client.NewClient("http://localhost:9000", "master-secret-key", "admin")
manifest, err := c.PutObject(ctx, "media", "video.mp4", fileStream)`}
              language="go"
            />
          )}

          {sdkTab === 'cweave' && (
            <CodeBlock
              code={`cweave put --file data.iso --key images/data.iso --node http://localhost:9000`}
              language="bash"
            />
          )}
        </div>
      </div>
    </div>
  )
}
