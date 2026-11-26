import { Link } from '@tanstack/react-router'
import { ArrowRight, Check, Handshake } from 'lucide-react'
import { EmptyState } from '@/components/brandshake/empty-state'
import { MoneyAmount } from '@/components/brandshake/money'
import { PageHeader } from '@/components/brandshake/page-header'
import { StatTile } from '@/components/brandshake/stat-tile'
import { buttonVariants } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import type { MyProfile, User } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { cn } from '@/lib/cn'
import { useMyProfile } from '@/lib/queries'

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

function Stats({ user }: { user: User }) {
  const zero = <MoneyAmount amountMinor={0} currency="INR" />
  if (user.role === 'creator') {
    return (
      <>
        <StatTile label="Unpaid invoices" value={zero} hint="Sent and awaiting payment" />
        <StatTile label="Paid this month" value={zero} />
        <StatTile label="Active deals" value="0" />
      </>
    )
  }
  return (
    <>
      <StatTile label="Active deals" value="0" />
      <StatTile label="Invoices to pay" value={zero} hint="Received from creators" />
      <StatTile label="Creators in your roster" value="0" />
    </>
  )
}

export function DashboardPage() {
  const { user } = useAuth()
  const { data } = useMyProfile()
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
        <Stats user={user} />
      </section>

      <Card>
        <CardHeader>
          <CardTitle>Deals</CardTitle>
        </CardHeader>
        <EmptyState
          icon={Handshake}
          title="No deals yet"
          description={
            user.role === 'creator'
              ? 'Deals you make with brands show up here, from first message to final payment.'
              : 'Deals you make with creators show up here, with deliverables and invoice status.'
          }
        />
      </Card>
    </div>
  )
}
