import { Check, Clock } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { MoneyAmount } from './money'

const steps = ['Agreed', 'Funded', 'Submitted', 'Verified', 'Paid'] as const
const current = 3

const checks = [
  { label: 'Post is live on the linked account', ok: true },
  { label: 'Sponsorship disclosure present', ok: true },
  { label: 'Promo code FIT20 in caption', ok: true },
  { label: 'Product shown for at least 5 seconds', ok: true },
]

/** Static product preview used on marketing and auth screens. */
export function PactPreview() {
  return (
    <div className="w-full max-w-md rounded-lg border border-line bg-raised shadow-float">
      <div className="flex items-center justify-between border-b border-line px-5 py-4">
        <div>
          <p className="text-[15px] font-semibold text-fg">FitBar × @arjunlifts</p>
          <p className="text-xs text-muted">1 Instagram Reel · due 4 Dec</p>
        </div>
        <Badge tone="success" dot>
          Verified
        </Badge>
      </div>

      <ol className="flex items-center gap-1.5 px-5 pt-4" aria-label="Pact progress">
        {steps.map((s, i) => (
          <li key={s} className="flex flex-1 flex-col gap-1.5">
            <span
              className={i <= current ? 'h-1 rounded-full bg-accent' : 'h-1 rounded-full bg-line'}
            />
            <span
              className={
                i <= current ? 'text-[11px] font-medium text-fg' : 'text-[11px] text-subtle'
              }
            >
              {s}
            </span>
          </li>
        ))}
      </ol>

      <ul className="flex flex-col gap-2.5 px-5 py-4">
        {checks.map((c) => (
          <li key={c.label} className="flex items-center gap-2.5 text-[13px] text-fg">
            <span className="flex size-4 items-center justify-center rounded-sm bg-success-subtle text-success">
              <Check className="size-3" strokeWidth={2.5} aria-hidden />
            </span>
            {c.label}
          </li>
        ))}
      </ul>

      <div className="flex items-center justify-between border-t border-line bg-surface px-5 py-3.5">
        <div>
          <p className="text-xs text-muted">In escrow</p>
          <MoneyAmount
            amountMinor={1_200_000}
            currency="INR"
            className="text-base font-semibold text-fg"
          />
        </div>
        <p className="flex items-center gap-1.5 text-xs text-muted">
          <Clock className="size-3.5" aria-hidden />
          Releases to creator in 41h
        </p>
      </div>
    </div>
  )
}
