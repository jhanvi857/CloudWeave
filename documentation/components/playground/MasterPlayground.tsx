'use client'

import React, { useState } from 'react'
import { PlaygroundShell } from './PlaygroundShell'

interface MasterPlaygroundProps {
  initialSubTab?: 'pipeline' | 'failure' | 'dedup' | 'erasure'
}

export function MasterPlayground({ initialSubTab = 'pipeline' }: MasterPlaygroundProps) {
  const [activeExperiment, setActiveExperiment] = useState<'pipeline' | 'failure' | 'dedup' | 'erasure'>(initialSubTab)

  return (
    <PlaygroundShell
      activeExperiment={activeExperiment}
      setActiveExperiment={setActiveExperiment}
    />
  )
}
