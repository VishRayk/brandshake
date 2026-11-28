import { Link, useNavigate } from '@tanstack/react-router'
import { Columns3, Handshake, List, Plus, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { StatusBadge, StatusSelect } from '@/components/deals/deal-status'
import { EmptyState } from '@/components/brandshake/empty-state'
import { MoneyAmount } from '@/components/brandshake/money'
import { PageHeader } from '@/components/brandshake/page-header'
import { buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Input } from '@/components/ui/field'
import { dealApi, type Deal, type DealStatus } from '@/lib/api'
import { cn } from '@/lib/cn'
import { STATUS_LABEL, STATUS_ORDER, dueLabel, daysUntil } from '@/lib/deals'
import { useDealMutation, useDeals } from '@/lib/queries'

export type DealsView = 'board' | 'list'

function nextDue(d: Deal) {
  return d.deliverables
    .filter((x) => !x.done && x.due_date)
    .map((x) => x.due_date!)
    .sort()[0]
}

function DueText({ date }: { date?: string }) {
  if (!date) return <span className="text-subtle">No due date</span>
  const overdue = daysUntil(date) < 0
  return (
    <span className={overdue ? 'font-medium text-danger' : 'text-muted'}>{dueLabel(date)}</span>
  )
}

function DealCard({ d }: { d: Deal }) {
  const move = useDealMutation((s: DealStatus) => dealApi.setStatus(d.id, s))
  const done = d.deliverables.filter((x) => x.done).length
  return (
    <li className="rounded-md border border-line bg-raised p-3 transition-colors duration-150 hover:border-line-strong">
      <Link to="/deals/$dealId" params={{ dealId: d.id }} className="block">
        <p className="text-[13px] font-medium text-fg">{d.title}</p>
        <p className="mt-0.5 truncate text-xs text-muted">{d.counterparty.name}</p>
        <div className="mt-2.5 flex items-center justify-between text-xs">
          {d.amount_minor != null ? (
            <MoneyAmount
              amountMinor={d.amount_minor}
              currency="INR"
              className="font-medium text-fg"
            />
          ) : (
            <span className="text-subtle">No fee yet</span>
          )}
          {d.deliverables.length > 0 && (
            <span className="tabular-nums text-subtle">
              {done}/{d.deliverables.length} done
            </span>
          )}
        </div>
        {d.status !== 'paid' && d.status !== 'lost' && (
          <p className="mt-1.5 text-xs">
            <DueText date={nextDue(d)} />
          </p>
        )}
      </Link>
      <StatusSelect
        value={d.status}
        onChange={(s) => move.mutate(s)}
        disabled={move.isPending}
        label={`Move ${d.title}`}
        className="mt-2.5 w-full"
      />
    </li>
  )
}

function Board({ deals }: { deals: Deal[] }) {
  const columns = STATUS_ORDER.filter((s) => s !== 'lost')
  const lost = deals.filter((d) => d.status === 'lost').length
  return (
    <div className="flex flex-col gap-3">
      <div className="-mx-4 overflow-x-auto px-4 pb-2 sm:-mx-6 sm:px-6">
        <div className="grid min-w-[1100px] grid-cols-6 gap-3">
          {columns.map((s) => {
            const items = deals.filter((d) => d.status === s)
            const total = items.reduce((n, d) => n + (d.amount_minor ?? 0), 0)
            return (
              <section
                key={s}
                aria-label={STATUS_LABEL[s]}
                className="flex flex-col gap-2 rounded-md bg-neutral-subtle/60 p-2"
              >
                <header className="flex items-baseline justify-between px-1 pt-1">
                  <h2 className="text-xs font-semibold text-fg">
                    {STATUS_LABEL[s]}{' '}
                    <span className="font-normal text-subtle">{items.length}</span>
                  </h2>
                  {total > 0 && (
                    <MoneyAmount
                      amountMinor={total}
                      currency="INR"
                      className="text-[11px] text-subtle"
                    />
                  )}
                </header>
                <ul className="flex flex-col gap-2">
                  {items.map((d) => (
                    <DealCard key={d.id} d={d} />
                  ))}
                </ul>
              </section>
            )
          })}
        </div>
      </div>
      {lost > 0 && (
        <p className="text-xs text-subtle">
          {lost} lost deal{lost === 1 ? '' : 's'} hidden. Switch to the list to see them.
        </p>
      )}
    </div>
  )
}

function ListView({ deals }: { deals: Deal[] }) {
  return (
    <Card className="overflow-x-auto">
      <table className="w-full min-w-[640px] text-[13px]">
        <thead>
          <tr className="border-b border-line text-left text-xs text-subtle">
            <th className="px-4 py-2.5 font-medium">Deal</th>
            <th className="px-4 py-2.5 font-medium">Status</th>
            <th className="px-4 py-2.5 text-right font-medium">Fee</th>
            <th className="px-4 py-2.5 font-medium">Next due</th>
            <th className="px-4 py-2.5 font-medium">Updated</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line">
          {deals.map((d) => (
            <tr key={d.id} className="transition-colors duration-150 hover:bg-surface">
              <td className="px-4 py-3">
                <Link
                  to="/deals/$dealId"
                  params={{ dealId: d.id }}
                  className="font-medium text-fg hover:underline"
                >
                  {d.title}
                </Link>
                <span className="block text-xs text-muted">{d.counterparty.name}</span>
              </td>
              <td className="px-4 py-3">
                <StatusBadge status={d.status} />
              </td>
              <td className="px-4 py-3 text-right">
                {d.amount_minor != null ? (
                  <MoneyAmount amountMinor={d.amount_minor} currency="INR" />
                ) : (
                  <span className="text-subtle">—</span>
                )}
              </td>
              <td className="px-4 py-3 text-xs">
                {d.status === 'paid' || d.status === 'lost' ? (
                  <span className="text-subtle">—</span>
                ) : (
                  <DueText date={nextDue(d)} />
                )}
              </td>
              <td className="px-4 py-3 text-xs text-muted">
                {new Date(d.updated_at).toLocaleDateString('en-IN', {
                  day: 'numeric',
                  month: 'short',
                })}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  )
}

export function DealsPage({ view, q }: { view: DealsView; q: string }) {
  const navigate = useNavigate({ from: '/deals' })
  const [search, setSearch] = useState(q)
  useEffect(() => {
    const t = setTimeout(() => {
      if (search !== q)
        navigate({ search: (s) => ({ ...s, q: search || undefined }), replace: true })
    }, 300)
    return () => clearTimeout(t)
  }, [search, q, navigate])

  const { data: deals, isPending, isError } = useDeals({ q: q || undefined })

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Deals"
        description="Every collaboration, from first message to final payment."
        actions={
          <Link to="/deals/new" className={buttonVariants({ size: 'sm' })}>
            <Plus aria-hidden /> New deal
          </Link>
        }
      />

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative w-full max-w-xs">
          <Search
            className="pointer-events-none absolute top-2.5 left-3 size-4 text-subtle"
            aria-hidden
          />
          <Input
            aria-label="Search deals"
            placeholder="Search by title or name"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <div
          className="ml-auto flex rounded-md border border-line-strong p-0.5"
          role="group"
          aria-label="View"
        >
          {(
            [
              ['board', 'Board', Columns3],
              ['list', 'List', List],
            ] as const
          ).map(([v, label, Icon]) => (
            <Link
              key={v}
              to="/deals"
              search={(s) => ({ ...s, view: v })}
              aria-pressed={view === v}
              className={cn(
                'flex h-7 items-center gap-1.5 rounded-sm px-2.5 text-xs font-medium transition-colors duration-150',
                view === v ? 'bg-fg text-background' : 'text-muted hover:text-fg',
              )}
            >
              <Icon className="size-3.5" aria-hidden />
              {label}
            </Link>
          ))}
        </div>
      </div>

      {isPending && (
        <div className="h-64 animate-pulse rounded-md border border-line bg-raised" aria-busy />
      )}
      {isError && (
        <p className="text-sm text-danger">Deals couldn't be loaded. Refresh to try again.</p>
      )}
      {deals && deals.length === 0 && (
        <Card>
          <EmptyState
            icon={Handshake}
            title={q ? 'No deals match your search' : 'Track your first deal'}
            description={
              q
                ? 'Try a different name or title.'
                : 'Paste a brand email or chat and Brandshake drafts the deal for you, or add one by hand.'
            }
            action={
              !q && (
                <Link to="/deals/new" className={buttonVariants({ size: 'sm' })}>
                  <Plus aria-hidden /> New deal
                </Link>
              )
            }
          />
        </Card>
      )}
      {deals &&
        deals.length > 0 &&
        (view === 'board' ? <Board deals={deals} /> : <ListView deals={deals} />)}
    </div>
  )
}
