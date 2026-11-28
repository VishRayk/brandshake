import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, ArrowUpRight, Pencil, ReceiptIndianRupee, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { DealForm, toFormValues } from '@/components/deals/deal-form'
import { StatusBadge, StatusSelect } from '@/components/deals/deal-status'
import { MoneyAmount } from '@/components/brandshake/money'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog } from '@/components/ui/controls'
import { ApiError, dealApi, type Deal, type DealEvent, type DealInput } from '@/lib/api'
import { cn } from '@/lib/cn'
import { STATUS_HINT, STATUS_LABEL, daysUntil, dueLabel } from '@/lib/deals'
import { useAuth } from '@/lib/auth'
import { useDeal, useDealEvents, useDealMutation } from '@/lib/queries'
import { PLATFORMS, labelOf } from '@/lib/vocab'

function Term({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 px-5 py-2.5">
      <dt className="text-[13px] text-muted">{label}</dt>
      <dd className="text-right text-[13px] font-medium text-fg">{value}</dd>
    </div>
  )
}

function usageText(d: Deal) {
  if (d.usage_perpetual) return 'Forever'
  if (d.usage_rights_days != null)
    return d.usage_rights_days === 0 ? 'None' : `${d.usage_rights_days} days`
  return 'None'
}

function eventText(e: DealEvent) {
  switch (e.kind) {
    case 'created':
      return e.detail === 'ai_draft' ? 'Deal created from a pasted message' : 'Deal created'
    case 'updated':
      return 'Details updated'
    case 'status_changed':
      return `Moved from ${STATUS_LABEL[e.from_status!]} to ${STATUS_LABEL[e.to_status!]}`
    case 'deliverable_done':
      return `Marked “${e.detail}” done`
    case 'deliverable_reopened':
      return `Reopened “${e.detail}”`
  }
}

function Deliverables({ d }: { d: Deal }) {
  const toggle = useDealMutation(({ i, done }: { i: number; done: boolean }) =>
    dealApi.setDone(d.id, i, done),
  )
  const closed = d.status === 'paid' || d.status === 'lost'
  return (
    <Card>
      <CardHeader>
        <CardTitle>Deliverables</CardTitle>
        {d.deliverables.length > 0 && (
          <span className="text-xs tabular-nums text-subtle">
            {d.deliverables.filter((x) => x.done).length} of {d.deliverables.length} done
          </span>
        )}
      </CardHeader>
      {d.deliverables.length === 0 ? (
        <p className="px-5 py-4 text-[13px] text-muted">
          No deliverables yet. Edit the deal to add them.
        </p>
      ) : (
        <ul className="divide-y divide-line">
          {d.deliverables.map((x, i) => {
            const overdue = !x.done && !closed && x.due_date && daysUntil(x.due_date) < 0
            return (
              <li key={`${i}-${x.title}`} className="flex items-center gap-3 px-5 py-3">
                <input
                  type="checkbox"
                  checked={x.done}
                  disabled={toggle.isPending}
                  onChange={(e) => toggle.mutate({ i, done: e.target.checked })}
                  aria-label={`${x.title} done`}
                  className="size-4 shrink-0 accent-[var(--accent)]"
                />
                <span className="min-w-0 flex-1">
                  <span
                    className={cn(
                      'block text-[13px]',
                      x.done ? 'text-subtle line-through' : 'text-fg',
                    )}
                  >
                    {x.title}
                  </span>
                  <span className="block text-xs text-subtle">
                    {labelOf([...PLATFORMS, ['other', 'Other']], x.platform)}
                  </span>
                </span>
                {x.due_date && !x.done && (
                  <span
                    className={cn('text-xs', overdue ? 'font-medium text-danger' : 'text-muted')}
                  >
                    {dueLabel(x.due_date)}
                  </span>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </Card>
  )
}

function DeleteDeal({ d }: { d: Deal }) {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const del = useDealMutation(() => dealApi.remove(d.id))
  return (
    <>
      <Button variant="ghost" size="sm" onClick={() => setOpen(true)} aria-label="Delete deal">
        <Trash2 aria-hidden />
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title="Delete this deal?">
        <p className="text-sm text-muted">“{d.title}” and its history are deleted permanently.</p>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" size="sm" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={del.isPending}
            onClick={() => del.mutate(undefined, { onSuccess: () => navigate({ to: '/deals' }) })}
          >
            Delete
          </Button>
        </div>
      </Dialog>
    </>
  )
}

export function DealDetailPage({ dealId }: { dealId: string }) {
  const { user } = useAuth()
  const { data: d, isPending, error } = useDeal(dealId)
  const { data: events } = useDealEvents(dealId)
  const [editing, setEditing] = useState(false)
  const setStatus = useDealMutation((s: Deal['status']) => dealApi.setStatus(dealId, s))
  const update = useDealMutation((input: DealInput) => dealApi.update(dealId, input))

  const back = (
    <Link
      to="/deals"
      className="flex items-center gap-1.5 self-start text-[13px] text-muted hover:text-fg"
    >
      <ArrowLeft className="size-4" aria-hidden /> Deals
    </Link>
  )

  if (isPending)
    return <div className="h-64 animate-pulse rounded-md border border-line bg-raised" aria-busy />
  if (error || !d) {
    return (
      <div className="flex flex-col gap-4">
        {back}
        <p className="text-sm text-muted">
          {error instanceof ApiError && error.status === 404
            ? 'This deal does not exist or was deleted.'
            : 'This deal could not be loaded.'}
        </p>
      </div>
    )
  }

  if (editing) {
    return (
      <div className="flex flex-col gap-6">
        {back}
        <h1 className="text-2xl font-semibold text-fg">Edit deal</h1>
        <DealForm
          initial={toFormValues(d)}
          submitLabel="Save changes"
          showDone
          onSubmit={async (input) => {
            await update.mutateAsync(input)
            setEditing(false)
          }}
          onCancel={() => setEditing(false)}
        />
      </div>
    )
  }

  const cp = d.counterparty
  return (
    <div className="flex flex-col gap-6">
      {back}
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold text-fg">{d.title}</h1>
            <StatusBadge status={d.status} />
          </div>
          <p className="mt-1 text-sm text-muted">
            with{' '}
            {cp.user ? (
              <Link
                to="/@{$handle}"
                params={{ handle: cp.user.handle }}
                className="inline-flex items-center gap-0.5 font-medium text-fg hover:underline"
              >
                {cp.name} <ArrowUpRight className="size-3.5" aria-hidden />
              </Link>
            ) : (
              <span className="font-medium text-fg">{cp.name}</span>
            )}
            {cp.email && <span className="text-subtle"> · {cp.email}</span>}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <StatusSelect
            value={d.status}
            onChange={(s) => setStatus.mutate(s)}
            disabled={setStatus.isPending}
          />
          {user?.role === 'creator' && d.status !== 'lost' && (
            <Link
              to="/invoices/new"
              search={{ deal: d.id }}
              className={buttonVariants({ variant: 'secondary', size: 'sm' })}
            >
              <ReceiptIndianRupee aria-hidden /> Create invoice
            </Link>
          )}
          <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>
            <Pencil aria-hidden /> Edit
          </Button>
          <DeleteDeal d={d} />
        </div>
      </div>
      <p className="-mt-3 text-xs text-subtle">{STATUS_HINT[d.status]}</p>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-6">
          <Deliverables d={d} />
          {d.notes && (
            <Card>
              <CardHeader>
                <CardTitle>Notes</CardTitle>
              </CardHeader>
              <p className="px-5 py-4 text-[13px] leading-6 whitespace-pre-line text-fg">
                {d.notes}
              </p>
            </Card>
          )}
        </div>

        <aside className="flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle>Terms</CardTitle>
            </CardHeader>
            <dl className="divide-y divide-line">
              <Term
                label="Fee"
                value={
                  d.amount_minor != null ? (
                    <MoneyAmount amountMinor={d.amount_minor} currency="INR" />
                  ) : (
                    '—'
                  )
                }
              />
              <Term label="Revisions" value={d.revisions ?? '—'} />
              <Term label="Usage rights" value={usageText(d)} />
              <Term
                label="Exclusivity"
                value={d.exclusivity_days ? `${d.exclusivity_days} days` : 'None'}
              />
            </dl>
            <p className="border-t border-line px-5 py-2.5 text-[11px] text-subtle">
              Paid directly between you; Brandshake never handles the money.
            </p>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Activity</CardTitle>
            </CardHeader>
            <ol className="flex flex-col gap-3 px-5 py-4">
              {(events ?? []).map((e) => (
                <li key={e.id} className="text-[13px]">
                  <p className="text-fg">{eventText(e)}</p>
                  <p className="text-xs text-subtle">
                    {new Date(e.created_at).toLocaleString('en-IN', {
                      day: 'numeric',
                      month: 'short',
                      hour: 'numeric',
                      minute: '2-digit',
                    })}
                  </p>
                </li>
              ))}
            </ol>
          </Card>
        </aside>
      </div>
    </div>
  )
}
