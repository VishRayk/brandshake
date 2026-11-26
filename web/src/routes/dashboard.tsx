import { useNavigate } from '@tanstack/react-router'
import { Handshake, LogOut } from 'lucide-react'
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
      </main>
    </div>
  )
}
