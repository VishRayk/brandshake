import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { Logo } from '@/components/brandshake/logo'
import { DealPreview } from '@/components/brandshake/deal-preview'
import { buttonVariants } from '@/components/ui/button'

const steps = [
  {
    title: 'A profile that works for you',
    body: 'One link with your niche, audience, rates and best work. Brands see everything they need at a glance.',
  },
  {
    title: 'Every deal in one place',
    body: 'Track each collaboration from first message to payment, even with brands that are not on Brandshake yet.',
  },
  {
    title: 'Invoices that get paid',
    body: 'GST-ready invoices with a UPI QR code, shared as a link. Friendly reminders go out when a payment is late.',
  },
  {
    title: 'Find the right partners',
    body: 'Brands discover creators by niche, city and audience. Creators find brands that sponsor work like theirs.',
  },
]

const YEAR = new Date().getFullYear()

export function LandingPage() {
  return (
    <div className="min-h-dvh bg-background">
      <header className="border-b border-line">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-6">
          <Logo />
          <nav className="flex items-center gap-2">
            <Link to="/login" className={buttonVariants({ variant: 'ghost', size: 'sm' })}>
              Sign in
            </Link>
            <Link to="/signup" className={buttonVariants({ size: 'sm' })}>
              Get started
            </Link>
          </nav>
        </div>
      </header>

      <main>
        <section className="mx-auto grid max-w-6xl items-center gap-12 px-6 py-20 lg:grid-cols-[1fr_auto] lg:py-28">
          <div className="max-w-xl">
            <p className="text-[13px] font-medium text-accent">
              For India's creators and the brands they work with
            </p>
            <h1 className="mt-3 text-4xl leading-[44px] font-semibold text-fg">
              Where creators and brands do business.
            </h1>
            <p className="mt-4 text-base leading-6 text-muted">
              Brandshake is the workspace and network for creator–brand partnerships. Manage every
              deal, send invoices that get paid, and meet your next collaborator.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <Link
                to="/signup"
                search={{ role: 'creator' }}
                className={buttonVariants({ size: 'lg' })}
              >
                I'm a creator
                <ArrowRight aria-hidden />
              </Link>
              <Link
                to="/signup"
                search={{ role: 'brand' }}
                className={buttonVariants({ variant: 'secondary', size: 'lg' })}
              >
                I'm a brand
              </Link>
            </div>
          </div>
          <DealPreview />
        </section>

        <section className="border-t border-line bg-surface">
          <div className="mx-auto max-w-6xl px-6 py-16">
            <h2 className="text-lg font-semibold text-fg">How Brandshake works</h2>
            <ol className="mt-8 grid gap-x-10 gap-y-8 sm:grid-cols-2 lg:grid-cols-4">
              {steps.map((s, i) => (
                <li key={s.title}>
                  <span className="font-mono text-xs text-subtle">0{i + 1}</span>
                  <h3 className="mt-2 text-[15px] font-semibold text-fg">{s.title}</h3>
                  <p className="mt-1.5 text-sm leading-5 text-muted">{s.body}</p>
                </li>
              ))}
            </ol>
          </div>
        </section>
      </main>

      <footer className="border-t border-line">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-6 text-xs text-subtle">
          <span>© {YEAR} Brandshake</span>
          <span>Built for India's creator economy</span>
        </div>
      </footer>
    </div>
  )
}
