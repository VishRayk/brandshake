import { useNavigate } from '@tanstack/react-router'
import { FileSignature, LogOut } from 'lucide-react'
import { EmptyState } from '@/components/brandshake/empty-state'
import { Logo } from '@/components/brandshake/logo'
import { MoneyAmount } from '@/components/brandshake/money'
import { StatTile } from '@/components/brandshake/stat-tile'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import type { User } from '@/lib/api'
import { useAuth } from '@/lib/auth'

function greeting(now = new Date()) {
  const h = now.getHours()
  return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'
}

function Stats({ user }: { user: User }) {
  const zero = <MoneyAmount amountMinor={0} currency="INR" />
  if (user.role === 'creator') {
    return (
      <>
        <StatTile label="In escrow" value={zero} hint="Held for your active pacts" />
        <StatTile label="Paid out" value={zero} hint="All time" />
        <StatTile label="Active pacts" value="0" />
      </>
    )
  }
  return (
    <>
      <StatTile label="Committed spend" value={zero} hint="Funded, not yet released" />
      <StatTile label="Active pacts" value="0" />
      <StatTile label="Awaiting your review" value="0" />
    </>
  )
}

export function DashboardPage() {
  const { user, signOut } = useAuth()
  const navigate = useNavigate()
  if (!user) return null

  const firstName = user.display_name.split(' ')[0]

  return (
    <div className="min-h-dvh bg-surface">
      <header className="border-b border-line bg-background">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
          <Logo />
          <div className="flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <p className="text-[13px] font-medium text-fg">{user.display_name}</p>
              <p className="text-xs text-subtle">{user.email}</p>
            </div>
            <Badge tone={user.role === 'creator' ? 'info' : 'neutral'}>
              {user.role === 'creator' ? 'Creator' : 'Brand'}
            </Badge>
            <Button
              variant="ghost"
              size="sm"
              aria-label="Sign out"
              onClick={() => {
                signOut()
                navigate({ to: '/login' })
              }}
            >
              <LogOut aria-hidden />
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-6 py-8">
        <h1 className="text-2xl font-semibold text-fg">
          {greeting()}, {firstName}
        </h1>
        <p className="mt-1 text-sm text-muted">
          {user.role === 'creator'
            ? 'Here is where your sponsorships and payouts stand.'
            : 'Here is where your creator campaigns stand.'}
        </p>

        <section className="mt-6 grid gap-4 sm:grid-cols-3" aria-label="Summary">
          <Stats user={user} />
        </section>

        <Card className="mt-6">
          <CardHeader>
            <CardTitle>Pacts</CardTitle>
          </CardHeader>
          <EmptyState
            icon={FileSignature}
            title="No pacts yet"
            description={
              user.role === 'creator'
                ? 'When a brand invites you and you agree on terms, the pact shows up here with its status and payout.'
                : 'Pacts you agree with creators show up here, with delivery checks and payment status.'
            }
          />
        </Card>
      </main>
    </div>
  )
}
