'use client';

import React from 'react';
import { Database, ShieldCheck, RefreshCw } from 'lucide-react';

function GithubIcon(props: React.SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      width="14"
      height="14"
      stroke="currentColor"
      strokeWidth="2"
      fill="none"
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M15 22v-4a4.8 4.8 0 0 0-1-3.5c3 0 6-2 6-5.5.08-1.25-.27-2.48-1-3.5.28-1.15.28-2.35 0-3.5 0 0-1 0-3 1.5-2.64-.5-5.36-.5-8 0C6 2 5 2 5 2c-.3 1.15-.3 2.35 0 3.5A5.403 5.403 0 0 0 4 9c0 3.5 3 5.5 6 5.5-.39.49-.68 1.05-.85 1.65-.17.6-.22 1.23-.15 1.85v4" />
      <path d="M9 18c-4.51 2-5-2-7-2" />
    </svg>
  );
}

// Isometric Node Chassis
function IsometricNode({
  label,
  sublabel,
  isHealing = false,
}: {
  label: string;
  sublabel: string;
  isHealing?: boolean;
}) {
  return (
    <div className="relative flex flex-col items-center group">
      {/* 3D Isometric Server Chassis SVG */}
      <div className="w-20 h-16 sm:w-24 sm:h-20 relative">
        <svg viewBox="0 0 100 80" className="w-full h-full">
          {/* Top Face */}
          <polygon
            points="50,6 88,24 50,42 12,24"
            fill="var(--surface-2)"
            stroke="var(--border)"
            strokeWidth="1.2"
          />
          {/* Left Face */}
          <polygon
            points="12,24 50,42 50,72 12,54"
            fill="var(--surface)"
            stroke="var(--border)"
            strokeWidth="1.2"
          />
          {/* Right Face */}
          <polygon
            points="50,42 88,24 88,54 50,72"
            fill="var(--bg)"
            stroke="var(--border)"
            strokeWidth="1.2"
          />
          {/* Drive Bay Slots */}
          <line x1="20" y1="36" x2="42" y2="46" stroke="var(--border)" strokeWidth="1.5" />
          <line x1="20" y1="44" x2="42" y2="54" stroke="var(--border)" strokeWidth="1.5" />
          <line x1="58" y1="46" x2="80" y2="36" stroke="var(--border)" strokeWidth="1.5" />
          <line x1="58" y1="54" x2="80" y2="44" stroke="var(--border)" strokeWidth="1.5" />
          {/* Status LEDs */}
          <circle cx="45" cy="46" r="1.5" fill={isHealing ? 'var(--brand)' : 'var(--success)'} />
          <circle cx="45" cy="54" r="1.5" fill={isHealing ? 'var(--brand)' : 'var(--success)'} />
        </svg>
      </div>

      {/* Node Mini Label */}
      <div className="mt-1 px-2 py-0.5 rounded-lg bg-surface border border-border text-[10px] font-mono text-muted flex items-center gap-1.5 shadow-none">
        <span className={`w-1.5 h-1.5 rounded-full ${isHealing ? 'bg-brand animate-pulse' : 'bg-success'}`} />
        <span className="text-text font-medium">{label}</span>
        <span className="text-subtle">{sublabel}</span>
      </div>
    </div>
  );
}

export function LandingHeroCluster() {
  return (
    <div className="relative w-full max-w-[520px] h-[400px] sm:h-[450px] flex items-center justify-center select-none">
      {/* Background isometric grid overlay */}
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,var(--brand-glow)_0,transparent_70%)] pointer-events-none" />

      {/* Subtle Coordinate Grid Lines */}
      <svg className="absolute inset-0 w-full h-full pointer-events-none" viewBox="0 0 520 450">
        <defs>
          <pattern id="isoGrid" width="40" height="40" patternUnits="userSpaceOnUse">
            <path d="M 40 0 L 0 40 M 0 0 L 40 40" fill="none" stroke="var(--border)" strokeWidth="0.75" strokeOpacity="0.4" />
          </pattern>
          <linearGradient id="beamGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="var(--brand)" stopOpacity="0.7" />
            <stop offset="100%" stopColor="var(--brand)" stopOpacity="0.1" />
          </linearGradient>
        </defs>

        <rect width="100%" height="100%" fill="url(#isoGrid)" />

        {/* Outer Ring Inter-node Connections (Consistent Hash Ring) */}
        <line x1="260" y1="75" x2="430" y2="200" stroke="var(--border)" strokeWidth="1.2" strokeDasharray="3 3" />
        <line x1="430" y1="200" x2="260" y2="340" stroke="var(--border)" strokeWidth="1.2" strokeDasharray="3 3" />
        <line x1="260" y1="340" x2="90" y2="200" stroke="var(--border)" strokeWidth="1.2" strokeDasharray="3 3" />
        <line x1="90" y1="200" x2="260" y2="75" stroke="var(--border)" strokeWidth="1.2" strokeDasharray="3 3" />

        {/* Spokes Connecting 4 Nodes to Center Chunk/Quorum */}
        <line x1="260" y1="85" x2="260" y2="205" stroke="url(#beamGrad)" strokeWidth="1.8" />
        <line x1="415" y1="200" x2="275" y2="205" stroke="url(#beamGrad)" strokeWidth="1.8" />
        <line x1="260" y1="325" x2="260" y2="215" stroke="url(#beamGrad)" strokeWidth="1.8" />
        <line x1="105" y1="200" x2="245" y2="205" stroke="url(#beamGrad)" strokeWidth="1.8" />

        {/* Small Data Packet Indicators */}
        <circle cx="260" cy="140" r="2.5" fill="var(--brand)" />
        <circle cx="340" cy="202" r="2.5" fill="var(--brand)" />
        <circle cx="260" cy="270" r="2.5" fill="var(--brand)" />
        <circle cx="175" cy="202" r="2.5" fill="var(--brand)" />
      </svg>

      {/* TOP NODE (Node 1) */}
      <div className="absolute top-2 left-1/2 -translate-x-1/2 z-20">
        <IsometricNode label="Node 1" sublabel=":9001" />
      </div>

      {/* LEFT NODE (Node 4) */}
      <div className="absolute top-1/2 left-1 sm:left-4 -translate-y-1/2 z-20">
        <IsometricNode label="Node 4" sublabel=":9004" />
      </div>

      {/* RIGHT NODE (Node 2) */}
      <div className="absolute top-1/2 right-1 sm:right-4 -translate-y-1/2 z-20">
        <IsometricNode label="Node 2" sublabel=":9002" />
      </div>

      {/* BOTTOM NODE (Node 3 - Auto-repairing/Simulated) */}
      <div className="absolute bottom-4 left-1/2 -translate-x-1/2 z-20">
        <IsometricNode label="Node 3" sublabel=":9003" isHealing />
      </div>

      {/* CENTERPIECE: CENTRAL CHUNK / ISOMETRIC DATA CUBE */}
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-30 flex items-center justify-center">
        <div className="relative w-20 h-20 sm:w-22 sm:h-22 flex items-center justify-center">
          {/* Brand Ambient Glow */}
          <div className="absolute inset-0 bg-brand rounded-2xl blur-lg opacity-25" />

          {/* Isometric Data Cube SVG */}
          <svg viewBox="0 0 100 100" className="w-full h-full">
            {/* Top Glowing Face */}
            <polygon
              points="50,14 84,32 50,50 16,32"
              fill="var(--brand)"
              fillOpacity="0.9"
              stroke="var(--brand-fg)"
              strokeWidth="1.2"
            />
            {/* Left Face */}
            <polygon
              points="16,32 50,50 50,86 16,68"
              fill="var(--brand-border)"
              fillOpacity="0.95"
              stroke="var(--brand)"
              strokeWidth="1.2"
            />
            {/* Right Face */}
            <polygon
              points="50,50 84,32 84,68 50,86"
              fill="var(--brand-soft)"
              fillOpacity="0.98"
              stroke="var(--brand-border)"
              strokeWidth="1.2"
            />
            {/* Chunk Subdivision Wirelines */}
            <line x1="33" y1="23" x2="67" y2="41" stroke="var(--brand-fg)" strokeOpacity="0.4" strokeWidth="1" />
            <line x1="33" y1="41" x2="33" y2="77" stroke="var(--brand-fg)" strokeOpacity="0.25" strokeWidth="1" />
            <line x1="67" y1="41" x2="67" y2="77" stroke="var(--brand-fg)" strokeOpacity="0.2" strokeWidth="1" />
          </svg>
        </div>
      </div>

      {/* 4 TECHNICAL ANNOTATION PANELS */}
      {/* 1. Top Left: S3 Compatible */}
      <div className="absolute top-4 left-0 sm:left-2 z-40 flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface border border-border text-[11px] font-mono text-text shadow-none">
        <Database className="w-3.5 h-3.5 text-brand" />
        <span>S3 Compatible</span>
      </div>

      {/* 2. Top Right: Replication / Erasure Coding */}
      <div className="absolute top-4 right-0 sm:right-2 z-40 flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface border border-border text-[11px] font-mono text-text shadow-none">
        <ShieldCheck className="w-3.5 h-3.5 text-brand" />
        <span>Replication / Erasure</span>
      </div>

      {/* 3. Bottom Left: Self-Healing */}
      <div className="absolute bottom-4 left-0 sm:left-2 z-40 flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface border border-border text-[11px] font-mono text-text shadow-none">
        <RefreshCw className="w-3.5 h-3.5 text-brand" />
        <span>Self-Healing</span>
      </div>

      {/* 4. Bottom Right: Open Source */}
      <div className="absolute bottom-4 right-0 sm:right-2 z-40 flex items-center gap-2 px-3 py-1.5 rounded-xl bg-surface border border-border text-[11px] font-mono text-text shadow-none">
        <GithubIcon className="w-3.5 h-3.5 text-muted" />
        <span>Open Source</span>
      </div>
    </div>
  );
}
