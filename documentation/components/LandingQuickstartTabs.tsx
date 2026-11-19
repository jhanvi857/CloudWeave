'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Copy, Check, Play, ArrowRight } from 'lucide-react';

export function LandingQuickstartTabs() {
  const [activeTab, setActiveTab] = useState<'docker' | 'compose' | 'source' | 'go'>('docker');
  const [copied, setCopied] = useState(false);

  const snippets: Record<string, string[]> = {
    docker: [
      'docker pull ghcr.io/jhanvi857/cloudweave:latest',
      'docker run -d \\',
      '  --name cloudweave -p 9000:9000 \\',
      '  -e CLOUDWEAVE_API_KEYS="master-secret-key=admin" \\',
      '  -v cloudweave_data:/data ghcr.io/jhanvi857/cloudweave:latest',
    ],
    compose: [
      'git clone https://github.com/jhanvi857/CloudWeave.git',
      'cd CloudWeave',
      '# Spin up a 5-node distributed cluster with automated quorum',
      'docker compose up -d',
      '# Open dashboard at http://localhost:9000/dashboard',
    ],
    source: [
      'git clone https://github.com/jhanvi857/CloudWeave.git',
      'cd CloudWeave',
      'go build -o node ./cmd/node',
      './node -port 9000 -data ./data -api-keys "master-secret-key=admin"',
    ],
    go: [
      'go install github.com/jhanvi857/cloudweave/cmd/cweave@latest',
      'cweave put backup.tar.gz --key archives/backup.tar.gz',
      'cweave stat archives/backup.tar.gz',
      'cweave get archives/backup.tar.gz --out restored.tar.gz',
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
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-stretch">
      {/* Code Editor Box with Tabs (8 cols) */}
      <div className="lg:col-span-8 flex flex-col rounded-xl border border-border bg-surface overflow-hidden shadow-none hover:border-brand-border transition-colors">
        {/* Tab Header Bar */}
        <div className="flex items-center justify-between border-b border-border bg-surface-2 px-4 py-2.5 overflow-x-auto">
          <div className="flex items-center gap-1.5 sm:gap-2">
            <button
              type="button"
              onClick={() => setActiveTab('docker')}
              className={`px-3 py-1.5 rounded-lg text-[13px] font-sans font-medium transition-colors cursor-pointer ${
                activeTab === 'docker'
                  ? 'bg-brand text-brand-fg'
                  : 'text-muted hover:text-text hover:bg-surface'
              }`}
            >
              Docker (Recommended)
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('compose')}
              className={`px-3 py-1.5 rounded-lg text-[13px] font-sans font-medium transition-colors cursor-pointer ${
                activeTab === 'compose'
                  ? 'bg-brand text-brand-fg'
                  : 'text-muted hover:text-text hover:bg-surface'
              }`}
            >
              Docker Compose
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('source')}
              className={`px-3 py-1.5 rounded-lg text-[13px] font-sans font-medium transition-colors cursor-pointer ${
                activeTab === 'source'
                  ? 'bg-brand text-brand-fg'
                  : 'text-muted hover:text-text hover:bg-surface'
              }`}
            >
              From Source
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('go')}
              className={`px-3 py-1.5 rounded-lg text-[13px] font-sans font-medium transition-colors cursor-pointer ${
                activeTab === 'go'
                  ? 'bg-brand text-brand-fg'
                  : 'text-muted hover:text-text hover:bg-surface'
              }`}
            >
              With Go
            </button>
          </div>

          <button
            type="button"
            onClick={handleCopy}
            className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[12px] font-mono text-muted hover:text-text bg-surface hover:bg-surface-2 border border-border transition-colors cursor-pointer shrink-0 ml-2"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-success" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>
        </div>

        {/* Code Lines Display */}
        <div className="p-6 font-mono text-[14px] leading-[1.6] overflow-x-auto selection:bg-brand-soft">
          <table className="w-full border-collapse">
            <tbody>
              {currentLines.map((line, idx) => (
                <tr key={idx} className="hover:bg-surface-2/60 transition-colors">
                  <td className="w-8 pr-4 text-right text-subtle select-none text-[13px]">
                    {idx + 1}
                  </td>
                  <td className="text-text whitespace-pre font-mono">
                    {line.startsWith('#') ? (
                      <span className="text-subtle italic">{line}</span>
                    ) : line.includes('docker') || line.includes('go') || line.includes('cweave') ? (
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

      {/* "See It In Action" Card (4 cols) */}
      <div className="lg:col-span-4 rounded-xl border border-border bg-surface p-6 flex flex-col justify-between shadow-none hover:border-brand-border transition-colors relative overflow-hidden">
        <div className="space-y-4">
          <div className="w-10 h-10 rounded-full bg-brand-soft border border-brand-border flex items-center justify-center text-brand">
            <Play className="w-5 h-5 fill-current ml-0.5" />
          </div>

          <h3 className="text-[20px] font-sans font-semibold text-heading tracking-tight">
            See It in Action
          </h3>

          <p className="text-[14px] font-sans text-muted leading-relaxed">
            Upload a file, click &ldquo;Simulate Failure&rdquo; and watch the cluster detect the dead node and automatically re-replicate under-replicated chunks.
          </p>
        </div>

        <div className="pt-6">
          <Link
            href="/docs/getting-started/docker-quickstart"
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
