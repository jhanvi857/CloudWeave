'use client';

import React, { useState, useEffect } from 'react';
import Link from 'next/link';
import {
  ShieldCheck,
  Server,
  Lock,
  ArrowRight,
  Copy,
  Check,
  RefreshCw,
  AlertTriangle,
  Layers,
  Cpu,
  Database,
  ExternalLink
} from 'lucide-react';

interface NodeState {
  id: string;
  name: string;
  port: number;
  status: 'healthy' | 'dead' | 'replicating';
  angle: number;
}

export function InteractiveClusterRing() {
  const [activeMode, setActiveMode] = useState<'normal' | 'quorum' | 'failure' | 'locked'>('quorum');
  const [copied, setCopied] = useState(false);
  const [selectedSnippet, setSelectedSnippet] = useState<'go' | 'docker' | 'compose'>('compose');
  const [repairProgress, setRepairProgress] = useState(0);

  const nodes: NodeState[] = [
    { id: 'node1', name: 'Node 1 (Coordinator)', port: 9000, status: 'healthy', angle: 270 },
    { id: 'node2', name: 'Node 2', port: 9001, status: 'healthy', angle: 342 },
    { id: 'node3', name: 'Node 3', port: 9002, status: activeMode === 'failure' ? 'dead' : 'healthy', angle: 54 },
    { id: 'node4', name: 'Node 4', port: 9003, status: activeMode === 'failure' ? 'replicating' : 'healthy', angle: 126 },
    { id: 'node5', name: 'Node 5', port: 9004, status: 'healthy', angle: 198 },
  ];

  useEffect(() => {
    if (activeMode === 'failure') {
      setRepairProgress(0);
      const interval = setInterval(() => {
        setRepairProgress((prev) => {
          if (prev >= 100) {
            clearInterval(interval);
            return 100;
          }
          return prev + 10;
        });
      }, 300);
      return () => clearInterval(interval);
    }
  }, [activeMode]);

  const snippets = {
    compose: 'git clone https://github.com/jhanvi857/CloudWeave.git && cd CloudWeave && docker compose up -d',
    docker: 'docker run -d -p 9000:9000 -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" ghcr.io/jhanvi857/cloudweave:latest',
    go: 'go install github.com/jhanvi857/cloudweave/cmd/cweave@latest'
  };

  const copyCommand = () => {
    navigator.clipboard.writeText(snippets[selectedSnippet]);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="w-full max-w-5xl mx-auto my-12 rounded-xl border border-border bg-surface p-6 md:p-8 shadow-none relative overflow-hidden">
      {/* Ambient glowing background aura */}
      <div className="absolute -top-32 -left-32 w-80 h-80 bg-brand-soft rounded-full blur-3xl pointer-events-none" />
      <div className="absolute -bottom-32 -right-32 w-80 h-80 bg-brand-soft rounded-full blur-3xl pointer-events-none" />

      {/* Header controls */}
      <div className="flex flex-col md:flex-row items-start md:items-center justify-between gap-4 border-b border-border pb-6">
        <div>
          <div className="eyebrow inline-flex items-center gap-2 px-2.5 py-1 rounded-full bg-surface-2 border border-border mb-2">
            <span className="w-2 h-2 rounded-full bg-success animate-pulse" />
            <span>Interactive Cluster Simulator</span>
          </div>
          <h3 className="text-xl md:text-2xl font-bold tracking-tight text-heading">
            Consistent Hash Ring & Quorum Engine
          </h3>
          <p className="text-sm text-muted mt-1">
            Real-time visualization of 150 virtual nodes, parallel fanout, and deterministic replica recovery.
          </p>
        </div>

        {/* State selector buttons */}
        <div className="flex flex-wrap items-center gap-2 bg-surface-2 p-1.5 rounded-lg border border-border text-xs font-medium">
          <button
            type="button"
            onClick={() => setActiveMode('quorum')}
            className={`px-3 py-1.5 rounded-md transition-colors cursor-pointer ${
              activeMode === 'quorum'
                ? 'bg-brand text-brand-fg font-semibold shadow-none'
                : 'text-muted hover:text-text'
            }`}
          >
            Quorum Write (N=3, W=2)
          </button>
          <button
            type="button"
            onClick={() => setActiveMode('failure')}
            className={`px-3 py-1.5 rounded-md transition-colors cursor-pointer flex items-center gap-1.5 ${
              activeMode === 'failure'
                ? 'bg-danger text-text font-semibold shadow-none'
                : 'text-muted hover:text-text'
            }`}
          >
            <AlertTriangle className="w-3.5 h-3.5" />
            Simulate Node Failure
          </button>
          <button
            type="button"
            onClick={() => setActiveMode('locked')}
            className={`px-3 py-1.5 rounded-md transition-colors cursor-pointer flex items-center gap-1.5 ${
              activeMode === 'locked'
                ? 'bg-success text-bg font-semibold shadow-none'
                : 'text-muted hover:text-text'
            }`}
          >
            <Lock className="w-3.5 h-3.5" />
            Lock Vault State
          </button>
        </div>
      </div>

      {/* Visualizer Canvas Area */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 my-8 items-center">
        {/* Ring Graphic (7 cols) */}
        <div className="lg:col-span-7 flex flex-col items-center justify-center relative min-h-[340px] sm:min-h-[380px]">
          <div className="relative w-[300px] h-[300px] sm:w-[340px] sm:h-[340px] flex items-center justify-center">
            {/* Outer Ring */}
            <div className={`absolute inset-0 rounded-full border-2 transition-all duration-700 ${
              activeMode === 'locked'
                ? 'border-success'
                : activeMode === 'failure'
                ? 'border-danger'
                : 'border-border'
            }`} />

            {/* Inner dashed ring */}
            <div className="absolute inset-8 rounded-full border border-dashed border-border animate-[spin_60s_linear_infinite]" />

            {/* Center Core Display */}
            <div className="z-10 text-center p-5 rounded-full bg-surface-2 border border-border w-36 h-36 flex flex-col items-center justify-center">
              {activeMode === 'locked' ? (
                <>
                  <Lock className="w-7 h-7 text-success mb-1 animate-bounce" />
                  <span className="text-[11px] font-mono font-bold text-success">VAULT SEALED</span>
                  <span className="text-[9px] font-mono text-muted">fsync Committed</span>
                </>
              ) : activeMode === 'failure' ? (
                <>
                  <RefreshCw className="w-7 h-7 text-brand mb-1 animate-spin" />
                  <span className="text-[11px] font-mono font-bold text-brand">SELF-HEALING</span>
                  <span className="text-[9px] font-mono text-muted">Repair {repairProgress}%</span>
                </>
              ) : (
                <>
                  <Database className="w-7 h-7 text-brand mb-1" />
                  <span className="text-[11px] font-mono font-bold text-text">SHA-256 CAS</span>
                  <span className="text-[9px] font-mono text-muted">150 Virtual Nodes</span>
                </>
              )}
            </div>

            {/* 5 Nodes on the Ring */}
            {nodes.map((node) => {
              const rad = (node.angle * Math.PI) / 180;
              const radius = 135;
              const x = Math.cos(rad) * radius;
              const y = Math.sin(rad) * radius;

              let nodeColor = 'bg-brand-soft border-brand text-brand';
              let badgeText = `:${node.port}`;

              if (node.status === 'dead') {
                nodeColor = 'bg-surface-2 border-danger text-danger animate-pulse';
                badgeText = 'DEAD';
              } else if (node.status === 'replicating') {
                nodeColor = 'bg-surface-2 border-brand text-brand';
                badgeText = '+REPLICA';
              } else if (activeMode === 'locked') {
                nodeColor = 'bg-surface-2 border-success text-success';
              }

              return (
                <div
                  key={node.id}
                  style={{
                    transform: `translate(${x}px, ${y}px)`,
                  }}
                  className={`absolute z-20 flex flex-col items-center justify-center w-12 h-12 rounded-xl border-2 backdrop-blur-md shadow-none transition-all duration-500 cursor-pointer hover:scale-110 ${nodeColor}`}
                  title={`${node.name} on port ${node.port}`}
                >
                  <Server className="w-5 h-5" />
                  <span className="text-[8px] font-mono font-bold mt-0.5">{badgeText}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Status log & explanations (5 cols) */}
        <div className="lg:col-span-5 flex flex-col gap-4">
          <div className="p-4 rounded-xl border border-border bg-surface-2 font-mono text-xs space-y-3">
            <div className="flex items-center justify-between text-muted text-[11px] border-b border-border pb-2">
              <span>CLUSTER TELEMETRY</span>
              <span className="text-success">LIVE FEED</span>
            </div>

            {activeMode === 'quorum' && (
              <div className="space-y-2 text-text">
                <div className="flex items-center gap-2 text-success">
                  <Check className="w-3.5 h-3.5" />
                  <span>Incoming PutObject: 10 MB payload</span>
                </div>
                <p className="text-[11px] text-muted leading-relaxed">
                  FastCDC split stream into 38 content-defined chunks. Ring selected Nodes [1, 2, 5] as primary target replicas.
                </p>
                <div className="p-2 rounded bg-surface border border-border text-[11px] text-success">
                  Quorum: 2/2 Acks received. WAL flushed to disk via fsync. Returned 201 Created.
                </div>
              </div>
            )}

            {activeMode === 'failure' && (
              <div className="space-y-2 text-text">
                <div className="flex items-center gap-2 text-danger">
                  <AlertTriangle className="w-3.5 h-3.5" />
                  <span>Node 3 (:9002) unreachable</span>
                </div>
                <p className="text-[11px] text-muted leading-relaxed">
                  Flap-damping detected 4 consecutive missed heartbeats (12s). Node declared DEAD.
                </p>
                <div className="p-2 rounded bg-surface border border-border text-[11px] text-brand">
                  Lexicographical primary survivor (Node 1) streaming under-replicated chunks to Node 4. Progress: {repairProgress}%
                </div>
              </div>
            )}

            {activeMode === 'locked' && (
              <div className="space-y-2 text-text">
                <div className="flex items-center gap-2 text-success">
                  <Lock className="w-3.5 h-3.5" />
                  <span>Cryptographic Vault State Locked</span>
                </div>
                <p className="text-[11px] text-muted leading-relaxed">
                  All active object manifests are causal-ordered and committed to WAL. Cluster ready for production traffic.
                </p>
                <div className="p-2 rounded bg-surface border border-border text-[11px] text-success">
                  Ready to test with AWS CLI, Boto3, or native cweave tooling.
                </div>
              </div>
            )}
          </div>

          {/* Quickstart snippet selector */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-xs text-muted font-mono">
              <span>ONE-LINE QUICKSTART</span>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setSelectedSnippet('compose')}
                  className={`cursor-pointer ${selectedSnippet === 'compose' ? 'text-brand font-bold underline' : ''}`}
                >
                  Compose
                </button>
                <button
                  type="button"
                  onClick={() => setSelectedSnippet('docker')}
                  className={`cursor-pointer ${selectedSnippet === 'docker' ? 'text-brand font-bold underline' : ''}`}
                >
                  Docker
                </button>
                <button
                  type="button"
                  onClick={() => setSelectedSnippet('go')}
                  className={`cursor-pointer ${selectedSnippet === 'go' ? 'text-brand font-bold underline' : ''}`}
                >
                  Go
                </button>
              </div>
            </div>

            <div className="flex items-center justify-between gap-3 px-3.5 py-2.5 rounded-lg bg-surface-2 border border-border font-mono text-xs text-text shadow-none">
              <span className="text-muted select-none">$</span>
              <span className="truncate flex-1 select-all">{snippets[selectedSnippet]}</span>
              <button
                type="button"
                onClick={copyCommand}
                className="p-1.5 rounded hover:bg-surface text-muted hover:text-text transition-colors cursor-pointer shrink-0"
                title="Copy command"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-success" /> : <Copy className="w-3.5 h-3.5" />}
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Action Footer */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4 pt-6 border-t border-border">
        <div className="flex items-center gap-3 text-xs text-muted font-mono">
          <span className="inline-flex items-center gap-1.5">
            <Cpu className="w-3.5 h-3.5 text-brand" /> Pure Go (Zero CGO)
          </span>
          <span>•</span>
          <span className="inline-flex items-center gap-1.5">
            <Layers className="w-3.5 h-3.5 text-brand" /> FastCDC Dedup
          </span>
          <span>•</span>
          <span className="inline-flex items-center gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5 text-success" /> S3 SigV4
          </span>
        </div>

        <div className="flex items-center gap-3">
          <Link
            href="/docs/getting-started"
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg bg-brand text-brand-fg font-medium text-xs shadow-none hover:bg-brand/90 transition-colors cursor-pointer"
          >
            <span>Read Getting Started</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
          <a
            href="https://github.com/jhanvi857/CloudWeave"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg border border-border bg-surface hover:bg-surface-2 font-medium text-xs text-text transition-colors cursor-pointer"
          >
            <span>GitHub</span>
            <ExternalLink className="w-3 h-3 text-muted" />
          </a>
        </div>
      </div>
    </div>
  );
}
