import type { ReactNode } from 'react'

export function StatTile({
  label,
  value,
  hint,
}: {
  label: string
  value: ReactNode
  hint?: string
}) {
  return (
    <div className="rounded-md border border-line bg-raised px-5 py-4">
      <p className="text-xs font-medium text-muted">{label}</p>
      <p className="mt-2 text-[28px] leading-8 font-semibold tabular-nums text-fg">{value}</p>
      {hint && <p className="mt-1 text-xs text-subtle">{hint}</p>}
    </div>
  )
}
