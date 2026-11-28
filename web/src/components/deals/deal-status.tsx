import { Badge } from '@/components/ui/badge'
import { Select } from '@/components/ui/controls'
import type { DealStatus } from '@/lib/api'
import { cn } from '@/lib/cn'
import { STATUS_HINT, STATUS_LABEL, STATUS_ORDER, STATUS_TONE } from '@/lib/deals'

export function StatusBadge({ status }: { status: DealStatus }) {
  return (
    <Badge tone={STATUS_TONE[status]} dot>
      {STATUS_LABEL[status]}
    </Badge>
  )
}

export function StatusSelect({
  value,
  onChange,
  disabled,
  className,
  label = 'Deal status',
}: {
  value: DealStatus
  onChange: (s: DealStatus) => void
  disabled?: boolean
  className?: string
  label?: string
}) {
  return (
    <Select
      aria-label={label}
      value={value}
      disabled={disabled}
      onChange={(e) => onChange(e.target.value as DealStatus)}
      className={cn('h-8 w-auto text-[13px]', className)}
    >
      {STATUS_ORDER.map((s) => (
        <option key={s} value={s} title={STATUS_HINT[s]}>
          {STATUS_LABEL[s]}
        </option>
      ))}
    </Select>
  )
}
