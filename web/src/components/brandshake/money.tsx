import { cn } from '@/lib/cn'
import { formatMoney } from '@/lib/money'

export function MoneyAmount({
  amountMinor,
  currency,
  className,
}: {
  amountMinor: number
  currency: string
  className?: string
}) {
  return <span className={cn('tabular-nums', className)}>{formatMoney(amountMinor, currency)}</span>
}
