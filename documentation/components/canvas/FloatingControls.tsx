'use client'

import React from 'react'
import { Play, Pause, SkipForward, RotateCcw, Gauge } from 'lucide-react'

interface FloatingControlsProps {
  step: number
  totalSteps: number
  stepName: string
  isPlaying: boolean
  onTogglePlay: () => void
  onStep: () => void
  onReset: () => void
  speed: number
  onSpeedChange: (speed: number) => void
  eventNotice?: string
}

export function FloatingControls({
  step,
  totalSteps,
  stepName,
  isPlaying,
  onTogglePlay,
  onStep,
  onReset,
  speed,
  onSpeedChange,
  eventNotice,
}: FloatingControlsProps) {
  return (
    <div className="flex flex-col items-center gap-2 w-full max-w-xl mx-auto">
      {/* Event notification indicator */}
      {eventNotice && (
        <div className="text-[11px] font-mono text-[#0077B6] bg-[#CAF0F8]/80 px-3 py-1 rounded-full border border-[#90E0EF]/60 shadow-xs animate-in fade-in slide-in-from-bottom-1">
          {eventNotice}
        </div>
      )}

      {/* Floating Pill Control Bar */}
      <div className="flex items-center justify-between gap-4 px-4 py-2 rounded-full border border-[#CBD5E1] bg-white/95 backdrop-blur-md shadow-sm w-full font-mono text-xs text-[#03045E]">
        {/* Step status */}
        <div className="flex items-center gap-1.5 shrink-0 pl-1">
          <span className="h-2 w-2 rounded-full bg-[#00B4D8] animate-pulse" />
          <span className="text-[11px] font-bold tracking-tight uppercase">
            STEP {step}/{totalSteps} · {stepName}
          </span>
        </div>

        {/* Buttons */}
        <div className="flex items-center gap-1">
          <button
            onClick={onStep}
            disabled={isPlaying}
            className="p-1.5 rounded-full text-[#475569] hover:text-[#03045E] hover:bg-[#F1F5F9] disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
            title="Step Forward"
          >
            <SkipForward className="w-4 h-4" />
          </button>

          <button
            onClick={onTogglePlay}
            className="flex items-center justify-center h-8 w-8 rounded-full bg-[#0077B6] text-white hover:bg-[#03045E] transition-colors shadow-xs"
            title={isPlaying ? 'Pause' : 'Play'}
          >
            {isPlaying ? (
              <Pause className="w-3.5 h-3.5 fill-current" />
            ) : (
              <Play className="w-3.5 h-3.5 fill-current ml-0.5" />
            )}
          </button>

          <button
            onClick={onReset}
            className="p-1.5 rounded-full text-[#475569] hover:text-[#03045E] hover:bg-[#F1F5F9] transition-colors"
            title="Reset"
          >
            <RotateCcw className="w-4 h-4" />
          </button>
        </div>

        {/* Speed Slider */}
        <div className="hidden sm:flex items-center gap-1.5 shrink-0 text-[10px] text-[#64748B] pr-1">
          <Gauge className="w-3.5 h-3.5 text-[#0077B6]" />
          <span>{speed}x</span>
          <input
            type="range"
            min="0.5"
            max="2.5"
            step="0.5"
            value={speed}
            onChange={(e) => onSpeedChange(parseFloat(e.target.value))}
            className="w-14 h-1 bg-[#E2E8F0] rounded-lg appearance-none cursor-pointer accent-[#0077B6]"
          />
        </div>
      </div>
    </div>
  )
}
