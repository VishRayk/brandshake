import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from '@tanstack/react-router'
import { z } from 'zod'
import type { AuthState } from '@/lib/auth'
import { DashboardPage } from '@/routes/dashboard'
import { LandingPage } from '@/routes/landing'
import { LoginPage } from '@/routes/login'
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

const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  beforeLoad: ({ context }) => {
    if (!context.auth.user) throw redirect({ to: '/login' })
  },
  component: DashboardPage,
})

const routeTree = rootRoute.addChildren([landingRoute, loginRoute, signupRoute, dashboardRoute])

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
