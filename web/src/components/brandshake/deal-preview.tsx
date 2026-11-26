import { Check, Clock, QrCode } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { MoneyAmount } from './money'

const stages = ['Lead', 'Negotiating', 'Confirmed', 'Creating', 'Posted', 'Paid'] as const
const current = 4

const deliverables = [
  { label: '1 Instagram Reel, 30–45s', done: true },
  { label: 'Promo code FIT20 in caption', done: true },
  { label: '#ad disclosure (ASCI)', done: true },
  { label: 'Story with link sticker', done: false },
]

/** Static product preview used on marketing and auth screens. */
export function DealPreview() {
  return (
    <div className="w-full max-w-md rounded-lg border border-line bg-raised shadow-float">
      <div className="flex items-center justify-between border-b border-line px-5 py-4">
        <div>
          <p className="text-[15px] font-semibold text-fg">FitBar × Maya Rao</p>
          <p className="text-xs text-muted">Protein launch · due 4 Dec</p>
        </div>
        <Badge tone="info" dot>
          Posted
        </Badge>
      </div>

      <ol className="flex items-center gap-1.5 px-5 pt-4" aria-label="Deal stage">
        {stages.map((s, i) => (
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
        {deliverables.map((d) => (
          <li key={d.label} className="flex items-center gap-2.5 text-[13px] text-fg">
            <span
              className={
                d.done
                  ? 'flex size-4 items-center justify-center rounded-sm bg-success-subtle text-success'
                  : 'size-4 rounded-sm border border-line-strong'
              }
            >
              {d.done && <Check className="size-3" strokeWidth={2.5} aria-hidden />}
            </span>
            <span className={d.done ? undefined : 'text-muted'}>{d.label}</span>
          </li>
        ))}
      </ul>

      <div className="flex items-center justify-between border-t border-line bg-surface px-5 py-3.5">
        <div>
          <p className="text-xs text-muted">Invoice BS-0014</p>
          <MoneyAmount
            amountMinor={1_500_000}
            currency="INR"
            className="text-base font-semibold text-fg"
          />
        </div>
        <div className="flex flex-col items-end gap-1">
          <span className="flex items-center gap-1.5 text-xs font-medium text-fg">
            <QrCode className="size-3.5" aria-hidden />
            Pay by UPI
          </span>
          <span className="flex items-center gap-1.5 text-xs text-muted">
            <Clock className="size-3.5" aria-hidden />
            Due in 5 days
          </span>
        </div>
      </div>
    </div>
  )
}
