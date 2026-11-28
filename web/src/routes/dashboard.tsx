import { Link } from '@tanstack/react-router'
import { ArrowRight, Check, Handshake, Plus } from 'lucide-react'
import { StatusBadge } from '@/components/deals/deal-status'
import { EmptyState } from '@/components/brandshake/empty-state'
import { MoneyAmount } from '@/components/brandshake/money'
import { PageHeader } from '@/components/brandshake/page-header'
import { StatTile } from '@/components/brandshake/stat-tile'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import type { DealSummary, MyProfile, User } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { cn } from '@/lib/cn'
import { dueLabel } from '@/lib/deals'
import { useDealSummary, useDeals, useMyProfile } from '@/lib/queries'

function greeting(now = new Date()) {
  const h = now.getHours()
  return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'
}

function setupSteps(user: User, data: MyProfile | undefined) {
  const p = data?.profile
  const steps = [
    { label: 'Claim your handle', done: !!p },
    {
      label:
        user.role === 'creator'
          ? 'Add a headline and your niches'
          : 'Add your industry and website',
      done:
        user.role === 'creator'
          ? !!p?.headline && (p?.niches.length ?? 0) > 0
          : !!p?.industry && !!p?.website,
    },
    { label: 'Link your social accounts', done: (data?.socials.length ?? 0) > 0 },
  ]
  if (user.role === 'creator') {
    steps.push(
      { label: 'Publish your rate card', done: (data?.rates.length ?? 0) > 0 },
      { label: 'Show your best work', done: (data?.samples.length ?? 0) > 0 },
    )
  }
  return steps
}

function SetupCard({ user, data }: { user: User; data: MyProfile | undefined }) {
  const steps = setupSteps(user, data)
  const done = steps.filter((s) => s.done).length
  if (done === steps.length) return null
  return (
    <Card>
      <CardHeader>
        <div>
          <CardTitle>Set up your profile</CardTitle>
          <p className="mt-0.5 text-xs text-muted">
            {user.role === 'creator'
              ? 'A complete profile is the link you send brands instead of screenshots.'
              : 'Creators check your page before they reply to an offer.'}
          </p>
        </div>
        <span className="text-xs font-medium tabular-nums text-muted">
          {done} of {steps.length}
        </span>
      </CardHeader>
      <div className="px-5 pt-3">
        <div className="h-1 rounded-full bg-line">
          <div
            className="h-1 rounded-full bg-accent transition-[width] duration-300"
            style={{ width: `${(done / steps.length) * 100}%` }}
          />
        </div>
      </div>
      <ul className="flex flex-col gap-2.5 px-5 py-4">
        {steps.map((s) => (
          <li key={s.label} className="flex items-center gap-2.5 text-[13px]">
            <span
              className={cn(
                'flex size-4 items-center justify-center rounded-full',
                s.done ? 'bg-success-subtle text-success' : 'border border-line-strong',
              )}
            >
              {s.done && <Check className="size-3" strokeWidth={2.5} aria-hidden />}
            </span>
            <span className={s.done ? 'text-subtle line-through' : 'text-fg'}>{s.label}</span>
          </li>
        ))}
      </ul>
      <div className="border-t border-line px-5 py-3">
        <Link to="/profile" className={buttonVariants({ size: 'sm' })}>
          Continue setup
          <ArrowRight aria-hidden />
        </Link>
      </div>
    </Card>
  )
}

function Stats({ user, summary }: { user: User; summary: DealSummary | undefined }) {
  const money = (v: number | undefined) => <MoneyAmount amountMinor={v ?? 0} currency="INR" />
  const active = summary?.active_count ?? 0
  const overdue = summary?.overdue_deliverables ?? 0
  return (
    <>
      <StatTile label="Active deals" value={String(active)} hint="Confirmed, creating or posted" />
      <StatTile
        label={user.role === 'creator' ? 'Awaiting payment' : 'To pay creators'}
        value={money(summary?.awaiting_payment_minor)}
        hint="Posted deals not yet marked paid"
      />
      <StatTile
        label={user.role === 'creator' ? 'Paid this month' : 'Settled this month'}
        value={money(summary?.paid_this_month_minor)}
        hint={
          overdue > 0
            ? `${overdue} overdue deliverable${overdue === 1 ? '' : 's'}`
            : 'Deals marked paid'
        }
      />
    </>
  )
}

function ComingUp({ summary }: { summary: DealSummary }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Coming up</CardTitle>
        <span className="text-xs text-subtle">Next 14 days</span>
      </CardHeader>
      {summary.due_soon.length === 0 ? (
        <p className="px-5 py-4 text-[13px] text-muted">Nothing due in the next two weeks.</p>
      ) : (
        <ul className="divide-y divide-line">
          {summary.due_soon.map((it) => (
            <li key={`${it.deal_id}-${it.title}`}>
              <Link
                to="/deals/$dealId"
                params={{ dealId: it.deal_id }}
                className="flex items-center justify-between gap-4 px-5 py-3 transition-colors duration-150 hover:bg-surface"
              >
                <span className="min-w-0">
                  <span className="block truncate text-[13px] font-medium text-fg">{it.title}</span>
                  <span className="block truncate text-xs text-subtle">
                    {it.deal_title} · {it.counterparty_name}
                  </span>
                </span>
                <span
                  className={cn(
                    'shrink-0 text-xs',
                    it.overdue ? 'font-medium text-danger' : 'text-muted',
                  )}
                >
                  {dueLabel(it.due_date)}
                </span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

function RecentDeals({ user }: { user: User }) {
  const { data: deals } = useDeals()
  const recent = (deals ?? []).slice(0, 5)
  return (
    <Card>
      <CardHeader>
        <CardTitle>Recent deals</CardTitle>
        {recent.length > 0 && (
          <Link to="/deals" className="text-xs font-medium text-muted hover:text-fg">
            View all
          </Link>
        )}
      </CardHeader>
      {deals && recent.length === 0 ? (
        <EmptyState
          icon={Handshake}
          title="No deals yet"
          description={
            user.role === 'creator'
              ? 'Paste a brand email or chat and Brandshake drafts the deal for you.'
              : 'Track each creator collaboration, from offer to payment.'
          }
          action={
            <Link to="/deals/new" className={buttonVariants({ size: 'sm' })}>
              <Plus aria-hidden /> New deal
            </Link>
          }
        />
      ) : (
        <ul className="divide-y divide-line">
          {recent.map((d) => (
            <li key={d.id}>
              <Link
                to="/deals/$dealId"
                params={{ dealId: d.id }}
                className="flex items-center justify-between gap-4 px-5 py-3 transition-colors duration-150 hover:bg-surface"
              >
                <span className="min-w-0">
                  <span className="block truncate text-[13px] font-medium text-fg">{d.title}</span>
                  <span className="block truncate text-xs text-subtle">{d.counterparty.name}</span>
                </span>
                <StatusBadge status={d.status} />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const { data } = useMyProfile()
  const { data: summary } = useDealSummary()
  if (!user) return null

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={`${greeting()}, ${user.display_name.split(' ')[0]}`}
        description={
          user.role === 'creator'
            ? 'Here is where your brand deals and invoices stand.'
            : 'Here is where your creator partnerships stand.'
        }
      />

      <SetupCard user={user} data={data} />

      <section className="grid gap-4 sm:grid-cols-3" aria-label="Summary">
        <Stats user={user} summary={summary} />
      </section>

      <div className="grid gap-6 lg:grid-cols-2">
        <RecentDeals user={user} />
        {summary && <ComingUp summary={summary} />}
      </div>
    </div>
  )
}
