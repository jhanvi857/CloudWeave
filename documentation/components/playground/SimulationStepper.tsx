'use client'

import React from 'react'
import { CheckCircle, ArrowRight } from 'lucide-react'

export type SimulationStep = 'upload' | 'chunking' | 'distribution' | 'replication' | 'quorum' | 'complete'

interface SimulationStepperProps {
  currentStep: SimulationStep
  onStepSelect: (step: SimulationStep) => void
}

const steps: { id: SimulationStep; label: string; number: string; desc: string }[] = [
  { id: 'upload', label: 'Upload', number: '01', desc: 'S3 PUT /video.mp4' },
  { id: 'chunking', label: 'Chunking', number: '02', desc: 'FastCDC + SHA-256' },
  { id: 'distribution', label: 'Distribution', number: '03', desc: '150 VNode Ring' },
  { id: 'replication', label: 'Replication', number: '04', desc: 'N=3 Multi-Node' },
  { id: 'quorum', label: 'Quorum', number: '05', desc: 'W=2 ACKs' },
  { id: 'complete', label: 'Complete', number: '06', desc: 'WAL Commit & 201' },
]

export function SimulationStepper({ currentStep, onStepSelect }: SimulationStepperProps) {
  const currentIndex = steps.findIndex((s) => s.id === currentStep)

  return (
    <div className="w-full border-b border-[#E2E8F0] bg-white py-3 px-4 sm:px-6">
      <div className="flex items-center justify-between overflow-x-auto gap-2 py-1">
        {steps.map((step, index) => {
          const isCurrent = step.id === currentStep
          const isCompleted = index < currentIndex
          const isUpcoming = index > currentIndex

          return (
            <button
              key={step.id}
              onClick={() => onStepSelect(step.id)}
              className={`flex items-center gap-2.5 px-3 py-2 rounded-lg border text-left shrink-0 transition-all ${
                isCurrent
                  ? 'border-[#00B4D8] bg-[#CAF0F8] text-[#03045E] shadow-2xs'
                  : isCompleted
                  ? 'border-[#A8E6CF] bg-[#E8F8F0] text-[#22A06B]'
                  : 'border-[#E2E8F0] bg-white text-[#64748B] hover:bg-[#F8FAFC]'
              }`}
            >
              <div
                className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-[11px] font-mono font-bold ${
                  isCurrent
                    ? 'bg-[#00B4D8] text-white animate-pulse'
                    : isCompleted
                    ? 'bg-[#22A06B] text-white'
                    : 'bg-[#F1F5F9] text-[#94A3B8]'
                }`}
              >
                {isCompleted ? <CheckCircle className="w-3.5 h-3.5" /> : step.number}
              </div>

              <div>
                <div className="flex items-center gap-1.5">
                  <span
                    className={`text-xs font-bold font-mono tracking-tight ${
                      isCurrent
                        ? 'text-[#0077B6]'
                        : isCompleted
                        ? 'text-[#22A06B]'
                        : 'text-[#475569]'
                    }`}
                  >
                    {step.label}
                  </span>
                </div>
                <span className="text-[10px] font-mono text-[#64748B] hidden sm:block">
                  {step.desc}
                </span>
              </div>

              {index < steps.length - 1 && (
                <ArrowRight className="w-3 h-3 text-[#CBD5E1] ml-1 hidden lg:block" />
              )}
            </button>
          )
        })}
      </div>
    </div>
  )
}
