import {
  createRootRouteWithContext,
  createRoute,
  createRouter,
  lazyRouteComponent,
  Outlet,
  redirect,
} from '@tanstack/react-router'
import { z } from 'zod'
import type { AuthState } from '@/lib/auth'
import { AppShell } from '@/components/shell/app-shell'

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
  component: lazyRouteComponent(() => import('@/routes/landing'), 'LandingPage'),
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  beforeLoad: redirectIfSignedIn,
  component: lazyRouteComponent(() => import('@/routes/login'), 'LoginPage'),
})

const SignupPage = lazyRouteComponent(() => import('@/routes/signup'), 'SignupPage')
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

// Pages behind sign-in are code-split so the landing page stays small.
const dashboardRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/dashboard',
  component: lazyRouteComponent(() => import('@/routes/dashboard'), 'DashboardPage'),
})

const DealsPage = lazyRouteComponent(() => import('@/routes/deals'), 'DealsPage')
const dealsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/deals',
  validateSearch: z.object({
    view: z.enum(['board', 'list']).catch('board').default('board'),
    q: z.string().optional(),
  }),
  component: function Deals() {
    const { view, q } = dealsRoute.useSearch()
    return <DealsPage view={view} q={q ?? ''} />
  },
})

const newDealRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/deals/new',
  component: lazyRouteComponent(() => import('@/routes/deal-new'), 'NewDealPage'),
})

const DealDetailPage = lazyRouteComponent(() => import('@/routes/deal-detail'), 'DealDetailPage')
const dealRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/deals/$dealId',
  component: function DealDetail() {
    const { dealId } = dealRoute.useParams()
    return <DealDetailPage key={dealId} dealId={dealId} />
  },
})

const InvoicesPage = lazyRouteComponent(() => import('@/routes/invoices'), 'InvoicesPage')
const invoicesRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/invoices',
  validateSearch: z.object({
    status: z.enum(['issued', 'overdue', 'paid', 'void']).optional().catch(undefined),
  }),
  component: function Invoices() {
    const { status } = invoicesRoute.useSearch()
    return <InvoicesPage status={status ?? ''} />
  },
})

const NewInvoicePage = lazyRouteComponent(() => import('@/routes/invoice-new'), 'NewInvoicePage')
const newInvoiceRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/invoices/new',
  validateSearch: z.object({ deal: z.string().optional() }),
  component: function NewInvoice() {
    const { deal } = newInvoiceRoute.useSearch()
    return <NewInvoicePage key={deal ?? 'blank'} dealId={deal} />
  },
})

const billingRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/invoices/billing',
  component: lazyRouteComponent(() => import('@/routes/billing'), 'BillingPage'),
})

const InvoiceDetailPage = lazyRouteComponent(
  () => import('@/routes/invoice-detail'),
  'InvoiceDetailPage',
)
const invoiceRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/invoices/$invoiceId',
  component: function InvoiceDetail() {
    const { invoiceId } = invoiceRoute.useParams()
    return <InvoiceDetailPage key={invoiceId} invoiceId={invoiceId} />
  },
})

const PublicInvoicePage = lazyRouteComponent(
  () => import('@/routes/public-invoice'),
  'PublicInvoicePage',
)
const publicInvoiceRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/i/$token',
  component: function PublicInvoice() {
    const { token } = publicInvoiceRoute.useParams()
    return <PublicInvoicePage token={token} />
  },
})

const profileRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/profile',
  component: lazyRouteComponent(() => import('@/routes/profile-edit'), 'ProfileEditPage'),
})

const settingsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/settings',
  component: lazyRouteComponent(() => import('@/routes/settings'), 'SettingsPage'),
})

const PublicProfilePage = lazyRouteComponent(
  () => import('@/routes/public-profile'),
  'PublicProfilePage',
)
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
  appRoute.addChildren([
    dashboardRoute,
    dealsRoute,
    newDealRoute,
    dealRoute,
    invoicesRoute,
    newInvoiceRoute,
    billingRoute,
    invoiceRoute,
    profileRoute,
    settingsRoute,
  ]),
  publicProfileRoute,
  publicInvoiceRoute,
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
