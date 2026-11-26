import { Link, Outlet, useNavigate } from '@tanstack/react-router'
import { ExternalLink, House, LogOut, Settings, UserRound, type LucideIcon } from 'lucide-react'
import { Avatar } from '@/components/brandshake/avatar'
import { Logo } from '@/components/brandshake/logo'
import { useAuth } from '@/lib/auth'
import { cn } from '@/lib/cn'
import { useMyProfile } from '@/lib/queries'

type NavItem = { to: '/dashboard' | '/profile' | '/settings'; label: string; icon: LucideIcon }

const NAV: NavItem[] = [
  { to: '/dashboard', label: 'Home', icon: House },
  { to: '/profile', label: 'Profile', icon: UserRound },
  { to: '/settings', label: 'Settings', icon: Settings },
]

function SidebarLink({ item }: { item: NavItem }) {
  return (
    <Link
      to={item.to}
      className="flex h-8 items-center gap-2.5 rounded-md px-2.5 text-[13px] font-medium text-muted transition-colors duration-150 hover:bg-neutral-subtle hover:text-fg"
      activeProps={{ className: 'bg-neutral-subtle text-fg' }}
    >
      <item.icon className="size-4" strokeWidth={1.75} aria-hidden />
      {item.label}
    </Link>
  )
}

export function AppShell() {
  const { user, signOut } = useAuth()
  const navigate = useNavigate()
  const { data } = useMyProfile()
  if (!user) return null

  const handle = data?.profile?.handle
  const doSignOut = () => {
    signOut()
    navigate({ to: '/login' })
  }

  return (
    <div className="min-h-dvh bg-surface lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="sticky top-0 hidden h-dvh flex-col border-r border-line bg-background lg:flex">
        <div className="flex h-14 items-center px-5">
          <Link to="/dashboard" aria-label="Brandshake home">
            <Logo />
          </Link>
        </div>
        <nav className="flex flex-col gap-0.5 px-3 py-2" aria-label="Main">
          {NAV.map((item) => (
            <SidebarLink key={item.to} item={item} />
          ))}
        </nav>
        {handle && (
          <div className="px-3">
            <Link
              to="/@{$handle}"
              params={{ handle }}
              className="flex h-8 items-center gap-2.5 rounded-md px-2.5 text-[13px] text-muted transition-colors duration-150 hover:bg-neutral-subtle hover:text-fg"
            >
              <ExternalLink className="size-4" strokeWidth={1.75} aria-hidden />
              View public profile
            </Link>
          </div>
        )}
        <div className="mt-auto flex items-center gap-2.5 border-t border-line px-4 py-3">
          <Avatar name={user.display_name} size="sm" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[13px] font-medium text-fg">{user.display_name}</p>
            <p className="truncate text-xs text-subtle">{handle ? `@${handle}` : user.email}</p>
          </div>
          <button
            type="button"
            onClick={doSignOut}
            aria-label="Sign out"
            className="rounded-md p-1.5 text-muted transition-colors duration-150 hover:bg-neutral-subtle hover:text-fg"
          >
            <LogOut className="size-4" aria-hidden />
          </button>
        </div>
      </aside>

      <div className="flex min-h-dvh flex-col">
        <header className="sticky top-0 z-10 flex h-14 items-center justify-between border-b border-line bg-background px-4 lg:hidden">
          <Link to="/dashboard" aria-label="Brandshake home">
            <Logo />
          </Link>
          <button
            type="button"
            onClick={doSignOut}
            aria-label="Sign out"
            className="rounded-md p-2 text-muted hover:bg-neutral-subtle hover:text-fg"
          >
            <LogOut className="size-4" aria-hidden />
          </button>
        </header>

        <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6 pb-24 sm:px-6 lg:py-8 lg:pb-8">
          <Outlet />
        </main>

        <nav
          aria-label="Main"
          className="fixed inset-x-0 bottom-0 z-10 grid grid-cols-3 border-t border-line bg-background pb-[env(safe-area-inset-bottom)] lg:hidden"
        >
          {NAV.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                'flex h-14 flex-col items-center justify-center gap-1 text-[11px] text-subtle',
              )}
              activeProps={{ className: 'text-fg font-medium' }}
            >
              <item.icon className="size-5" strokeWidth={1.75} aria-hidden />
              {item.label}
            </Link>
          ))}
        </nav>
      </div>
    </div>
  )
}
