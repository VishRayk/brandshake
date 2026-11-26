import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { Logo } from '@/components/brandshake/logo'
import { PactPreview } from '@/components/brandshake/pact-preview'
import { buttonVariants } from '@/components/ui/button'

const steps = [
  {
    title: 'Verified profiles',
    body: 'Creators connect their accounts. Stats come straight from the platform, not from screenshots.',
  },
  {
    title: 'Clear terms',
    body: 'Deliverables, deadlines, usage rights and price are written down and agreed by both sides.',
  },
  {
    title: 'Escrowed payment',
    body: 'Brands fund the deal up front. Creators start work knowing the money is there.',
  },
  {
    title: 'Checked delivery',
    body: 'Each post is checked against the brief: live, disclosed, with the right link and code. Then payment releases.',
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
              For creators and the brands they work with
            </p>
            <h1 className="mt-3 text-4xl leading-[44px] font-semibold text-fg">
              Creator deals, without the chasing.
            </h1>
            <p className="mt-4 text-base leading-6 text-muted">
              Brandshake turns a sponsorship from a DM thread into a pact: agreed terms, money held in
              escrow, and delivery checked automatically before anyone gets paid.
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
          <PactPreview />
        </section>

        <section className="border-t border-line bg-surface">
          <div className="mx-auto max-w-6xl px-6 py-16">
            <h2 className="text-lg font-semibold text-fg">How a pact works</h2>
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
          <span>Payments are held by a licensed payment provider.</span>
        </div>
      </footer>
    </div>
  )
}
