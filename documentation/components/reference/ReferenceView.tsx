'use client'

import React, { useState } from 'react'
import { Terminal, FileCode, Server } from 'lucide-react'
import { CodeBlock } from '../ui/CodeBlock'

interface Endpoint {
  method: 'PUT' | 'GET' | 'DELETE' | 'HEAD' | 'POST'
  path: string
  desc: string
  auth: string
  response: string
}

const s3Endpoints: Endpoint[] = [
  { method: 'PUT', path: '/:bucket/:key', desc: 'Streams object payload into FastCDC and writes to quorum N=3', auth: 'AWS SigV4 / Master Key', response: '201 Created (ETag: "cw-hash")' },
  { method: 'GET', path: '/:bucket/:key', desc: 'Queries R=2 replicas, resolves vector clock, and streams reassembled data', auth: 'AWS SigV4 / Master Key', response: '200 OK (binary byte stream)' },
  { method: 'HEAD', path: '/:bucket/:key', desc: 'Returns object metadata, size, vector clock, and ETag without data body', auth: 'AWS SigV4 / Master Key', response: '200 OK' },
  { method: 'DELETE', path: '/:bucket/:key', desc: 'Appends tombstone marker to WAL and schedules chunk garbage collection', auth: 'AWS SigV4 / Master Key', response: '204 No Content' },
  { method: 'GET', path: '/:bucket?list-type=2', desc: 'Lists all stored object keys in S3 ListObjectsV2 XML/JSON format', auth: 'AWS SigV4 / Master Key', response: '200 OK (XML payload)' },
]

const internalEndpoints: Endpoint[] = [
  { method: 'PUT', path: '/internal/chunks/:chunkID', desc: 'Stores raw CAS chunk to local disk and memory LRU cache', auth: 'Mutual TLS (mTLS)', response: '200 OK' },
  { method: 'GET', path: '/internal/chunks/:chunkID', desc: 'Fetches raw CAS chunk from local disk or LRU RAM cache', auth: 'Mutual TLS (mTLS)', response: '200 OK' },
  { method: 'POST', path: '/internal/heartbeat', desc: 'Mesh heartbeat ping carrying node load and active virtual nodes', auth: 'Mutual TLS (mTLS)', response: '200 OK (Pong)' },
  { method: 'POST', path: '/internal/manifest', desc: 'Gossip broadcast of newly committed file manifest to active cluster peers', auth: 'Mutual TLS (mTLS)', response: '200 OK' },
]

export function ReferenceView() {
  const [activeTab, setActiveTab] = useState<'s3' | 'internal' | 'cli'>('s3')

  return (
    <div className="space-y-6 font-sans">
      {/* Title */}
      <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C]">
        <div className="flex items-center gap-2">
          <span className="rounded px-2.5 py-1 text-[10px] font-mono font-bold text-[#A6A9AE] bg-[#000000] border border-[#26262B] uppercase">
            API & CLI Reference
          </span>
          <span className="text-xs font-mono text-[#8B8B8F]">Complete Endpoints & Flags</span>
        </div>
        <h2 className="text-2xl font-serif font-bold tracking-tight text-[#EDE8DF] mt-2">
          CloudWeave Interface Reference
        </h2>
        <p className="text-xs sm:text-sm font-mono text-[#8B8B8F] mt-1 max-w-2xl">
          Specification of public S3 REST endpoints, internal node-to-node transport APIs, and the cweave CLI manual.
        </p>

        {/* Tab Buttons */}
        <div className="flex flex-wrap gap-2 mt-4 pt-3 border-t border-[#26262B] font-mono text-xs">
          <button
            onClick={() => setActiveTab('s3')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${activeTab === 's3'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            S3-Compatible REST API
          </button>
          <button
            onClick={() => setActiveTab('internal')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${activeTab === 'internal'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            Internal mTLS Transport Endpoints
          </button>
          <button
            onClick={() => setActiveTab('cli')}
            className={`px-3 py-1.5 rounded font-semibold transition-all cursor-pointer ${activeTab === 'cli'
                ? 'bg-[#7A1F2B] text-[#EDE8DF] shadow-xs'
                : 'bg-[#000000] text-[#8B8B8F] border border-[#26262B] hover:text-[#EDE8DF]'
              }`}
          >
            cweave CLI Manual
          </button>
        </div>
      </div>

      {/* Tab Panels */}
      {activeTab === 's3' && (
        <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4">
          <div className="overflow-x-auto">
            <table className="w-full text-left font-mono text-xs">
              <thead className="bg-[#000000] text-[#8B8B8F] border-b border-[#26262B]">
                <tr>
                  <th className="py-2.5 px-4 font-bold">Method</th>
                  <th className="py-2.5 px-4 font-bold">Path</th>
                  <th className="py-2.5 px-4 font-bold">Description</th>
                  <th className="py-2.5 px-4 font-bold">Auth</th>
                  <th className="py-2.5 px-4 font-bold">Response</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#26262B] text-[#EDE8DF]">
                {s3Endpoints.map((ep, i) => (
                  <tr key={i} className="hover:bg-[#222226]">
                    <td className="py-2.5 px-4">
                      <span className="px-2 py-0.5 rounded bg-[#000000] text-[#EDE8DF] font-bold border border-[#26262B]">
                        {ep.method}
                      </span>
                    </td>
                    <td className="py-2.5 px-4 font-semibold text-[#EDE8DF]">{ep.path}</td>
                    <td className="py-2.5 px-4 text-[#8B8B8F] font-sans text-xs">{ep.desc}</td>
                    <td className="py-2.5 px-4 text-[#EDE8DF]">{ep.auth}</td>
                    <td className="py-2.5 px-4 text-[#8B8B8F]">{ep.response}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'internal' && (
        <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4">
          <div className="overflow-x-auto">
            <table className="w-full text-left font-mono text-xs">
              <thead className="bg-[#000000] text-[#8B8B8F] border-b border-[#26262B]">
                <tr>
                  <th className="py-2.5 px-4 font-bold">Method</th>
                  <th className="py-2.5 px-4 font-bold">Internal Path</th>
                  <th className="py-2.5 px-4 font-bold">Functionality</th>
                  <th className="py-2.5 px-4 font-bold">Security Transport</th>
                  <th className="py-2.5 px-4 font-bold">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#26262B] text-[#EDE8DF]">
                {internalEndpoints.map((ep, i) => (
                  <tr key={i} className="hover:bg-[#222226]">
                    <td className="py-2.5 px-4">
                      <span className="px-2 py-0.5 rounded bg-[#000000] text-[#EDE8DF] font-bold border border-[#26262B]">
                        {ep.method}
                      </span>
                    </td>
                    <td className="py-2.5 px-4 font-semibold text-[#EDE8DF]">{ep.path}</td>
                    <td className="py-2.5 px-4 text-[#8B8B8F] font-sans text-xs">{ep.desc}</td>
                    <td className="py-2.5 px-4 text-[#EDE8DF]">{ep.auth}</td>
                    <td className="py-2.5 px-4 text-[#8B8B8F]">{ep.response}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {activeTab === 'cli' && (
        <div className="p-6 rounded-xl border border-[#26262B] bg-[#19191C] space-y-4 font-mono text-xs">
          <CodeBlock
            code={`# Upload an object with FastCDC and quorum verification
cweave put <filepath> [--bucket <name>] [--endpoint <url>]

# Download an object with parallel chunk reassembly
cweave get <key> [--output <path>] [--endpoint <url>]

# List stored objects in a bucket
cweave ls [--bucket <name>] [--endpoint <url>]

# Remove an object and append tombstone to WAL
cweave rm <key> [--bucket <name>] [--endpoint <url>]

# Inspect cluster health and 150-vnode distribution
cweave cluster status [--endpoint <url>]`}
            language="bash"
            filename="cweave-manual.sh"
          />
        </div>
      )}
    </div>
  )
}
