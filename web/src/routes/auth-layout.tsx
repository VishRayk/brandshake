import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { Logo } from '@/components/brandshake/logo'
import { DealPreview } from '@/components/brandshake/deal-preview'

export function AuthLayout({
  title,
  subtitle,
  footer,
  children,
}: {
  title: string
  subtitle: string
  footer: ReactNode
  children: ReactNode
}) {
  return (
    <div className="grid min-h-dvh lg:grid-cols-2">
      <div className="flex flex-col px-6 py-8 sm:px-12">
        <Link to="/" aria-label="Brandshake home" className="self-start">
          <Logo />
        </Link>
        <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center py-12">
          <h1 className="text-2xl font-semibold text-fg">{title}</h1>
          <p className="mt-1.5 text-sm text-muted">{subtitle}</p>
          <div className="mt-8">{children}</div>
          <p className="mt-6 text-sm text-muted">{footer}</p>
        </div>
      </div>
      <div className="hidden items-center justify-center border-l border-line bg-surface p-12 lg:flex">
        <DealPreview />
      </div>
    </div>
  )
}

export function FormError({ message }: { message?: string }) {
  if (!message) return null
  return (
    <p
      role="alert"
      className="rounded-md border border-danger/20 bg-danger-subtle px-3 py-2 text-[13px] text-danger"
    >
      {message}
    </p>
  )
}
