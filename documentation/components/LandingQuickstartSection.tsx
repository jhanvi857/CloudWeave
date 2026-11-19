'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Copy, Check, ArrowRight, Activity, RotateCcw } from 'lucide-react';

export function LandingQuickstartSection() {
  const [activeTab, setActiveTab] = useState<'docker' | 'compose' | 'cli'>('docker');
  const [copied, setCopied] = useState(false);

  // Simulation state for the "See It in Action" widget
  const [simState, setSimState] = useState<'normal' | 'failed' | 'healing' | 'recovered'>('normal');

  const runSimulation = () => {
    if (simState !== 'normal' && simState !== 'recovered') return;
    setSimState('failed');
    setTimeout(() => {
      setSimState('healing');
      setTimeout(() => {
        setSimState('recovered');
      }, 1600);
    }, 1200);
  };

  const resetSimulation = () => {
    setSimState('normal');
  };

  const snippets: Record<string, string[]> = {
    docker: [
      'docker pull ghcr.io/jhanvi857/cloudweave:latest',
      '',
      'docker run -d \\',
      '  --name cloudweave \\',
      '  -p 9000:9000 \\',
      '  --memory=256m \\',
      '  -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" \\',
      '  -v cloudweave_data:/data \\',
      '  ghcr.io/jhanvi857/cloudweave:latest',
    ],
    compose: [
      'git clone https://github.com/jhanvi857/CloudWeave.git',
      'cd CloudWeave',
      '# Spin up a 5-node cluster with quorum and web dashboard',
      'docker compose up -d',
      '# Dashboard running at http://localhost:9000/dashboard',
    ],
    cli: [
      'go install github.com/jhanvi857/cloudweave/cmd/cweave@latest',
      'cweave put demo.mp4 --key videos/demo.mp4',
      'cweave stat videos/demo.mp4',
      'cweave get videos/demo.mp4 --out recovered.mp4',
    ],
  };

  const currentLines = snippets[activeTab];
  const fullText = currentLines.join('\n');

  const handleCopy = () => {
    navigator.clipboard.writeText(fullText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
      {/* 1. LEFT COLUMN: Heading & Description (3 cols) */}
      <div className="lg:col-span-3 space-y-4">
        <div className="eyebrow">
          QUICKSTART
        </div>

        <h2 className="text-[36px] font-sans font-semibold text-heading leading-[1.15] tracking-[-0.02em]">
          Get Started in Minutes
        </h2>

        <p className="text-[16px] font-sans text-muted leading-[1.7]">
          Spin up an S3-compatible cluster on your local machine with Docker or Docker Compose in seconds.
        </p>

        <div className="pt-2">
          <Link
            href="/docs/getting-started"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>View All Options</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      {/* 2. CENTER COLUMN: Terminal Code Panel (5 cols) */}
      <div className="lg:col-span-5 rounded-xl border border-border bg-surface overflow-hidden shadow-none flex flex-col hover:border-brand-border transition-colors">
        {/* Terminal Header Bar */}
        <div className="flex items-center justify-between border-b border-border bg-surface-2 px-4 py-2.5">
          {/* Three Window Indicators */}
          <div className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-danger opacity-80" />
            <span className="w-2.5 h-2.5 rounded-full bg-brand opacity-80" />
            <span className="w-2.5 h-2.5 rounded-full bg-success opacity-80" />
            <span className="ml-2 text-[11px] font-mono text-subtle">bash</span>
          </div>

          {/* Tab switcher & Copy Button */}
          <div className="flex items-center gap-2">
            <div className="flex items-center bg-surface rounded-lg p-0.5 border border-border text-[11px] font-mono">
              <button
                type="button"
                onClick={() => setActiveTab('docker')}
                className={`px-2 py-0.5 rounded transition-colors cursor-pointer ${activeTab === 'docker' ? 'bg-surface-2 text-text font-medium' : 'text-muted hover:text-text'
                  }`}
              >
                Docker
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('compose')}
                className={`px-2 py-0.5 rounded transition-colors cursor-pointer ${activeTab === 'compose' ? 'bg-surface-2 text-text font-medium' : 'text-muted hover:text-text'
                  }`}
              >
                Compose
              </button>
              <button
                type="button"
                onClick={() => setActiveTab('cli')}
                className={`px-2 py-0.5 rounded transition-colors cursor-pointer ${activeTab === 'cli' ? 'bg-surface-2 text-text font-medium' : 'text-muted hover:text-text'
                  }`}
              >
                CLI
              </button>
            </div>

            <button
              type="button"
              onClick={handleCopy}
              className="flex items-center gap-1 px-2 py-1 rounded bg-surface hover:bg-surface-2 border border-border text-[11px] font-mono text-muted hover:text-text transition-colors cursor-pointer"
              title="Copy to clipboard"
            >
              {copied ? <Check className="w-3 h-3 text-success" /> : <Copy className="w-3 h-3" />}
              <span>{copied ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
        </div>

        {/* Terminal Body */}
        <div className="p-6 font-mono text-[14px] leading-[1.6] overflow-x-auto selection:bg-brand-soft">
          <table className="w-full border-collapse">
            <tbody>
              {currentLines.map((line, idx) => (
                <tr key={idx} className="hover:bg-surface-2/60 transition-colors">
                  <td className="w-6 pr-3 text-right text-subtle select-none text-[12px]">
                    {idx + 1}
                  </td>
                  <td className="text-text whitespace-pre font-mono">
                    {line.startsWith('#') ? (
                      <span className="text-subtle italic">{line}</span>
                    ) : line.startsWith('docker') || line.startsWith('git') || line.startsWith('go') || line.startsWith('cweave') ? (
                      <span>
                        <span className="text-brand">{line.split(' ')[0]}</span>{' '}
                        <span>{line.substring(line.indexOf(' ') + 1)}</span>
                      </span>
                    ) : (
                      line
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* 3. RIGHT COLUMN: "See It in Action" Panel (4 cols) */}
      <div className="lg:col-span-4 rounded-xl border border-border bg-surface p-6 flex flex-col justify-between shadow-none hover:border-brand-border transition-colors">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="eyebrow flex items-center gap-1.5">
              <Activity className="w-3.5 h-3.5" />
              <span>See It in Action</span>
            </div>
            {simState !== 'normal' && (
              <button
                type="button"
                onClick={resetSimulation}
                className="text-[11px] font-mono text-muted hover:text-text flex items-center gap-1 cursor-pointer"
              >
                <RotateCcw className="w-3 h-3" />
                <span>Reset</span>
              </button>
            )}
          </div>

          <p className="text-[14px] font-sans text-muted leading-relaxed">
            Upload a file, click <strong className="text-text font-medium">Simulate Failure</strong> and watch the cluster heal!
          </p>

          {/* Mini Interactive Dashboard Preview Box */}
          <div className="rounded-lg border border-border bg-surface-2 p-3 space-y-2.5 font-mono text-[11px]">
            <div className="flex items-center justify-between border-b border-border pb-1.5 text-subtle">
              <span>CLUSTER TOPOLOGY</span>
              <span>QUORUM: W=2 / R=2</span>
            </div>

            {/* 4 Node status rows */}
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <span className="text-text">node-1 (:9001)</span>
                <span className="text-success">HEALTHY</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-text">node-2 (:9002)</span>
                <span className="text-success">HEALTHY</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-text">node-3 (:9003)</span>
                {simState === 'normal' && <span className="text-success">HEALTHY</span>}
                {simState === 'failed' && <span className="text-danger font-semibold animate-pulse">OFFLINE</span>}
                {simState === 'healing' && <span className="text-brand animate-pulse">REPAIRING CHUNKS</span>}
                {simState === 'recovered' && <span className="text-success font-medium">RECOVERED (N=3)</span>}
              </div>
              <div className="flex items-center justify-between">
                <span className="text-text">node-4 (:9004)</span>
                <span className="text-success">HEALTHY</span>
              </div>
            </div>

            {/* Live Status indicator */}
            <div className="border-t border-border pt-2 flex items-center justify-between text-[10px]">
              <span className="text-subtle">
                {simState === 'normal' && 'All 4 nodes in sync.'}
                {simState === 'failed' && 'Heartbeat missed: triggering re-replication.'}
                {simState === 'healing' && 'Replication worker fanning out parity chunks.'}
                {simState === 'recovered' && 'Self-healing complete. Full quorum intact.'}
              </span>
            </div>
          </div>

          {/* Simulate Failure Button */}
          <button
            type="button"
            onClick={runSimulation}
            disabled={simState === 'failed' || simState === 'healing'}
            className={`w-full py-2.5 px-4 rounded-lg text-[13px] font-mono font-medium transition-colors flex items-center justify-center gap-1.5 ${simState === 'normal' || simState === 'recovered'
                ? 'bg-transparent border border-border hover:bg-surface-2 text-text hover:border-brand-border cursor-pointer'
                : 'bg-surface border border-border text-subtle cursor-not-allowed'
              }`}
          >
            {simState === 'normal' && 'Simulate Failure'}
            {simState === 'failed' && 'Node Failed — Detecting...'}
            {simState === 'healing' && 'Self-Healing in Progress...'}
            {simState === 'recovered' && 'Simulate Again'}
          </button>
        </div>

        <div className="pt-4 border-t border-border mt-4">
          <Link
            href="/docs/getting-started/your-first-object"
            className="inline-flex items-center gap-1.5 text-[14px] font-sans font-medium text-brand hover:underline transition-colors"
          >
            <span>Try the Demo</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </div>
  );
}
