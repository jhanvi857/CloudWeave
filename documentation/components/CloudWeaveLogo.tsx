import React from 'react';

export function CloudWeaveCubeLogo({ className = 'w-6 h-6' }: { className?: string }) {
  return (
    <div className={`${className} relative flex items-center justify-center shrink-0`}>
      <svg viewBox="0 0 32 32" className="w-full h-full">
        {/* Layer 1 (bottom) */}
        <path
          d="M16 26L6 20L16 14L26 20L16 26Z"
          fill="var(--surface-2)"
          stroke="var(--border)"
          strokeWidth="1"
        />
        {/* Layer 2 (middle) */}
        <path
          d="M16 20L6 14L16 8L26 14L16 20Z"
          fill="var(--brand-soft)"
          stroke="var(--brand-border)"
          strokeWidth="1"
        />
        {/* Layer 3 (top) */}
        <path
          d="M16 14L6 8L16 2L26 8L16 14Z"
          fill="var(--brand)"
          stroke="var(--brand-fg)"
          strokeWidth="1"
        />
      </svg>
    </div>
  );
}

export function CloudWeaveLogo({ className = '' }: { className?: string }) {
  return (
    <div className={`flex items-center gap-2.5 ${className}`}>
      <CloudWeaveCubeLogo className="w-6 h-6" />
      <span className="font-sans font-semibold tracking-tight text-[16px] text-text">
        Cloud<span className="text-brand">Weave</span>
      </span>
    </div>
  );
}
