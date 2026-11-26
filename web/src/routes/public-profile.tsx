import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowUpRight, Check, Globe, Link2, MapPin, UserRoundX } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Avatar } from '@/components/brandshake/avatar'
import { EmptyState } from '@/components/brandshake/empty-state'
import { Logo } from '@/components/brandshake/logo'
import { MoneyAmount } from '@/components/brandshake/money'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { ApiError, profileApi, type PublicProfile } from '@/lib/api'
import { useAuth } from '@/lib/auth'
import { formatCount, monthYear } from '@/lib/format'
import { keys } from '@/lib/queries'
import { INDUSTRIES, LANGUAGES, NICHES, PLATFORMS, labelOf } from '@/lib/vocab'

function TopBar() {
  const { user } = useAuth()
  return (
    <header className="border-b border-line bg-background">
      <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4 sm:px-6">
        <Link to="/" aria-label="Brandshake home">
          <Logo />
        </Link>
        {user ? (
          <Link to="/dashboard" className={buttonVariants({ variant: 'secondary', size: 'sm' })}>
            Dashboard
          </Link>
        ) : (
          <nav className="flex gap-2">
            <Link to="/login" className={buttonVariants({ variant: 'ghost', size: 'sm' })}>
              Sign in
            </Link>
            <Link to="/signup" className={buttonVariants({ size: 'sm' })}>
              Join Brandshake
            </Link>
          </nav>
        )}
      </div>
    </header>
  )
}

function CopyLink() {
  const [copied, setCopied] = useState(false)
  return (
    <Button
      variant="secondary"
      size="sm"
      onClick={async () => {
        await navigator.clipboard.writeText(window.location.href)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }}
    >
      {copied ? <Check aria-hidden /> : <Link2 aria-hidden />}
      {copied ? 'Link copied' : 'Copy link'}
    </Button>
  )
}

function host(url: string) {
  try {
    return new URL(url).host.replace(/^www\./, '')
  } catch {
    return url
  }
}

function ProfileView({ p }: { p: PublicProfile }) {
  const isCreator = p.role === 'creator'
  const location = [p.city, p.state].filter(Boolean).join(', ')
  const reach = p.socials.reduce((n, s) => n + s.followers, 0)

  useEffect(() => {
    document.title = `${p.display_name} (@${p.handle}) · Brandshake`
    return () => {
      document.title = 'Brandshake: where creators and brands do business'
    }
  }, [p.display_name, p.handle])

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6 px-4 py-8 sm:px-6">
      <Card className="px-5 py-6 sm:px-6">
        <div className="flex flex-col gap-5 sm:flex-row sm:items-start">
          <Avatar name={p.display_name} size="lg" />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-semibold text-fg">{p.display_name}</h1>
              <Badge tone={isCreator ? 'info' : 'neutral'}>{isCreator ? 'Creator' : 'Brand'}</Badge>
              {isCreator && p.open_to_collabs && (
                <Badge tone="success" dot>
                  Open to collaborations
                </Badge>
              )}
            </div>
            <p className="mt-0.5 text-sm text-subtle">@{p.handle}</p>
            {p.headline && <p className="mt-3 text-[15px] text-fg">{p.headline}</p>}
            <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-[13px] text-muted">
              {location && (
                <span className="flex items-center gap-1.5">
                  <MapPin className="size-3.5" aria-hidden /> {location}
                </span>
              )}
              {p.website && (
                <a
                  href={p.website}
                  target="_blank"
                  rel="noopener noreferrer nofollow"
                  className="flex items-center gap-1.5 hover:text-fg"
                >
                  <Globe className="size-3.5" aria-hidden /> {host(p.website)}
                </a>
              )}
              {!isCreator && p.industry && <span>{labelOf(INDUSTRIES, p.industry)}</span>}
            </div>
          </div>
          <CopyLink />
        </div>

        {isCreator && p.niches.length > 0 && (
          <ul className="mt-5 flex flex-wrap gap-1.5" aria-label="Niches">
            {p.niches.map((n) => (
              <li
                key={n}
                className="rounded-sm border border-line bg-surface px-2 py-0.5 text-xs text-muted"
              >
                {labelOf(NICHES, n)}
              </li>
            ))}
          </ul>
        )}
      </Card>

      <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
        <div className="flex flex-col gap-6">
          {p.bio && (
            <Card>
              <CardHeader>
                <CardTitle>About</CardTitle>
              </CardHeader>
              <p className="px-5 py-4 text-sm leading-6 whitespace-pre-line text-fg">{p.bio}</p>
            </Card>
          )}

          {isCreator && (
            <Card>
              <CardHeader>
                <CardTitle>Work</CardTitle>
              </CardHeader>
              {p.samples.length ? (
                <ul className="divide-y divide-line">
                  {p.samples.map((s) => (
                    <li key={s.url}>
                      <a
                        href={s.url}
                        target="_blank"
                        rel="noopener noreferrer nofollow"
                        className="group flex items-center justify-between gap-4 px-5 py-3.5 transition-colors duration-150 hover:bg-surface"
                      >
                        <span className="min-w-0">
                          <span className="block truncate text-sm font-medium text-fg">
                            {s.title}
                          </span>
                          <span className="block text-xs text-subtle">
                            {[
                              s.brand_name && `with ${s.brand_name}`,
                              s.platform && labelOf(PLATFORMS, s.platform),
                            ]
                              .filter(Boolean)
                              .join(' · ') || host(s.url)}
                          </span>
                        </span>
                        <ArrowUpRight
                          className="size-4 shrink-0 text-subtle group-hover:text-fg"
                          aria-hidden
                        />
                      </a>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="px-5 py-4 text-sm text-muted">No work samples yet.</p>
              )}
            </Card>
          )}

          {isCreator && p.rates.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Rates</CardTitle>
                <span className="text-xs text-subtle">Starting prices, before GST</span>
              </CardHeader>
              <table className="w-full text-sm">
                <tbody className="divide-y divide-line">
                  {p.rates.map((r) => (
                    <tr key={r.deliverable}>
                      <td className="px-5 py-3">
                        <span className="block text-fg">{r.deliverable}</span>
                        {r.note && <span className="block text-xs text-subtle">{r.note}</span>}
                      </td>
                      <td className="px-5 py-3 text-right font-medium text-fg">
                        <MoneyAmount amountMinor={r.price_minor} currency="INR" />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </Card>
          )}
        </div>

        <aside className="flex flex-col gap-6">
          {p.socials.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle>Audience</CardTitle>
                {isCreator && reach > 0 && (
                  <span className="text-xs text-subtle tabular-nums">
                    {formatCount(reach)} total
                  </span>
                )}
              </CardHeader>
              <ul className="divide-y divide-line">
                {p.socials.map((s) => (
                  <li key={s.platform}>
                    <a
                      href={s.url}
                      target="_blank"
                      rel="noopener noreferrer nofollow"
                      className="flex items-center justify-between gap-3 px-5 py-3 transition-colors duration-150 hover:bg-surface"
                    >
                      <span className="min-w-0">
                        <span className="block text-[13px] font-medium text-fg">
                          {labelOf(PLATFORMS, s.platform)}
                        </span>
                        <span className="block truncate text-xs text-subtle">@{s.handle}</span>
                      </span>
                      {s.followers > 0 && (
                        <span className="text-sm font-medium tabular-nums text-fg">
                          {formatCount(s.followers)}
                        </span>
                      )}
                    </a>
                  </li>
                ))}
              </ul>
              <p className="border-t border-line px-5 py-2.5 text-[11px] text-subtle">
                Follower counts as listed by the profile owner.
              </p>
            </Card>
          )}

          <Card className="px-5 py-4">
            <dl className="flex flex-col gap-3 text-[13px]">
              {isCreator && p.languages.length > 0 && (
                <div>
                  <dt className="text-xs text-subtle">Creates in</dt>
                  <dd className="mt-0.5 text-fg">
                    {p.languages.map((l) => labelOf(LANGUAGES, l)).join(', ')}
                  </dd>
                </div>
              )}
              <div>
                <dt className="text-xs text-subtle">On Brandshake since</dt>
                <dd className="mt-0.5 text-fg">{monthYear(p.member_since)}</dd>
              </div>
            </dl>
          </Card>
        </aside>
      </div>
    </div>
  )
}

export function PublicProfilePage({ handle }: { handle: string }) {
  const { data, isPending, error } = useQuery({
    queryKey: keys.publicProfile(handle),
    queryFn: () => profileApi.public(handle),
    retry: (count, err) => !(err instanceof ApiError && err.status === 404) && count < 2,
  })

  return (
    <div className="min-h-dvh bg-surface">
      <TopBar />
      {isPending && (
        <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6" aria-busy>
          <div className="h-44 animate-pulse rounded-md border border-line bg-raised" />
        </div>
      )}
      {error && (
        <div className="mx-auto max-w-5xl px-4 py-16 sm:px-6">
          <Card>
            <EmptyState
              icon={UserRoundX}
              title={
                error instanceof ApiError && error.status === 404
                  ? 'Profile not found'
                  : 'Something went wrong'
              }
              description={
                error instanceof ApiError && error.status === 404
                  ? `There's no one on Brandshake with the handle @${handle}.`
                  : 'This profile could not be loaded. Please try again in a moment.'
              }
              action={
                <Link to="/" className={buttonVariants({ variant: 'secondary', size: 'sm' })}>
                  Go to Brandshake
                </Link>
              }
            />
          </Card>
        </div>
      )}
      {data && <ProfileView p={data.profile} />}
    </div>
  )
}
