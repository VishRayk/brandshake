import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from '@tanstack/react-router'
import { z } from 'zod'
import type { AuthState } from '@/lib/auth'
import { AppShell } from '@/components/shell/app-shell'
import { DashboardPage } from '@/routes/dashboard'
import { LandingPage } from '@/routes/landing'
import { LoginPage } from '@/routes/login'
import { ProfileEditPage } from '@/routes/profile-edit'
import { PublicProfilePage } from '@/routes/public-profile'
import { SettingsPage } from '@/routes/settings'
import { SignupPage } from '@/routes/signup'

interface RouterContext {
  auth: AuthState
}

const rootRoute = createRootRouteWithContext<RouterContext>()({
  component: Outlet,
})

const redirectIfSignedIn = ({ context }: { context: RouterContext }) => {
  if (context.auth.user) throw redirect({ to: '/dashboard' })
}

const landingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  beforeLoad: redirectIfSignedIn,
  component: LandingPage,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: redirectIfSignedIn,
  component: LoginPage,
})

const signupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/signup',
  validateSearch: z.object({ role: z.enum(['creator', 'brand']).optional() }),
  beforeLoad: redirectIfSignedIn,
  component: function Signup() {
    const { role } = signupRoute.useSearch()
    return <SignupPage initialRole={role} />
  },
})

// Signed-in area: every child renders inside the app shell.
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'app',
  beforeLoad: ({ context }) => {
    if (!context.auth.user) throw redirect({ to: '/login' })
  },
  component: AppShell,
})

const dashboardRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/dashboard',
  component: DashboardPage,
})

const profileRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/profile',
  component: ProfileEditPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/settings',
  component: SettingsPage,
})

const publicProfileRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/@{$handle}',
  component: function PublicProfile() {
    const { handle } = publicProfileRoute.useParams()
    return <PublicProfilePage handle={handle.toLowerCase()} />
  },
})

const routeTree = rootRoute.addChildren([
  landingRoute,
  loginRoute,
  signupRoute,
  appRoute.addChildren([dashboardRoute, profileRoute, settingsRoute]),
  publicProfileRoute,
])

export const router = createRouter({
  routeTree,
  context: { auth: undefined! },
  defaultPreload: 'intent',
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
