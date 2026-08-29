import React from 'react'

interface BadgeProps {
  children: React.ReactNode
  variant?: 'primary' | 'cyan' | 'success' | 'warning' | 'danger' | 'neutral'
  className?: string
}

export function Badge({ children, variant = 'primary', className = '' }: BadgeProps) {
  const variantStyles = {
    primary: 'bg-[#19191C] text-[#EDE8DF] border-[#7A1F2B]',
    cyan: 'bg-[#19191C] text-[#A6A9AE] border-[#26262B]',
    success: 'bg-[#19191C] text-[#EDE8DF] border-[#26262B]',
    warning: 'bg-[#19191C] text-[#A6A9AE] border-[#26262B]',
    danger: 'bg-[#7A1F2B] text-[#EDE8DF] border-[#7A1F2B]',
    neutral: 'bg-[#19191C] text-[#8B8B8F] border-[#26262B]',
  }

  return (
    <span
      className={`inline-flex items-center gap-1 rounded px-2 py-0.5 text-[10px] font-mono font-semibold tracking-wide border uppercase ${variantStyles[variant]} ${className}`}
    >
      {children}
    </span>
  )
}
