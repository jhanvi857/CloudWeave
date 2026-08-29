'use client'

import React from 'react'
import { Play, Pause, SkipForward, RotateCcw, Gauge, Sparkles } from 'lucide-react'

interface SimulationControlsProps {
  isPlaying: boolean
  onTogglePlay: () => void
  onStepForward: () => void
  onReset: () => void
  speed: number
  onSpeedChange: (speed: number) => void
  statusText?: string
}

export function SimulationControls({
  isPlaying,
  onTogglePlay,
  onStepForward,
  onReset,
  speed,
  onSpeedChange,
  statusText = 'Ready to simulate',
}: SimulationControlsProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[#E2E8F0] bg-[#F8FAFC] px-4 py-2.5 sm:px-6">
      {/* Primary Action Buttons */}
      <div className="flex items-center gap-2">
        <button
          onClick={onTogglePlay}
          className={`flex items-center gap-2 rounded-lg px-4 py-2 text-xs font-bold text-white shadow-xs transition-all ${
            isPlaying
              ? 'bg-[#0077B6] hover:bg-[#03045E]'
              : 'bg-[#0077B6] hover:bg-[#00B4D8]'
          }`}
          title={isPlaying ? 'Pause Simulation' : 'Play Simulation'}
        >
          {isPlaying ? (
            <>
              <Pause className="w-3.5 h-3.5 fill-current" />
              <span>Pause</span>
            </>
          ) : (
            <>
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Play</span>
            </>
          )}
        </button>

        <button
          onClick={onStepForward}
          disabled={isPlaying}
          className="flex items-center gap-1.5 rounded-lg border border-[#CBD5E1] bg-white px-3 py-2 text-xs font-semibold text-[#03045E] hover:bg-[#CAF0F8] hover:border-[#00B4D8] disabled:opacity-40 disabled:cursor-not-allowed transition-all"
          title="Advance exactly one stage"
        >
          <span>Step</span>
          <SkipForward className="w-3.5 h-3.5 text-[#0077B6]" />
        </button>

        <button
          onClick={onReset}
          className="flex items-center gap-1.5 rounded-lg border border-[#CBD5E1] bg-white px-3 py-2 text-xs font-medium text-[#475569] hover:bg-[#F1F5F9] hover:text-[#03045E] transition-all"
          title="Reset simulation to initial state"
        >
          <RotateCcw className="w-3.5 h-3.5" />
          <span>Reset</span>
        </button>
      </div>

      {/* Speed Slider & Live Status */}
      <div className="flex items-center gap-4 text-xs font-mono">
        <div className="flex items-center gap-2 bg-white px-3 py-1.5 rounded-lg border border-[#E2E8F0]">
          <Gauge className="w-3.5 h-3.5 text-[#0077B6]" />
          <span className="text-[#64748B] text-[11px]">Speed:</span>
          <input
            type="range"
            min="0.5"
            max="2"
            step="0.25"
            value={speed}
            onChange={(e) => onSpeedChange(parseFloat(e.target.value))}
            className="w-16 h-1.5 bg-[#CBD5E1] rounded-lg appearance-none cursor-pointer accent-[#0077B6]"
          />
          <span className="font-bold text-[#0077B6] min-w-[32px] text-right">
            {speed}x
          </span>
        </div>

        <div className="hidden md:flex items-center gap-2 text-[11px] text-[#64748B]">
          <span className="inline-block h-2 w-2 rounded-full bg-[#00B4D8] animate-ping" />
          <span className="font-medium text-[#03045E]">{statusText}</span>
        </div>
      </div>
    </div>
  )
}
