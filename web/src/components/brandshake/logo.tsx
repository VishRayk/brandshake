import { cn } from '@/lib/cn'

export function Logo({ className }: { className?: string }) {
  return (
    <span
      className={cn('inline-flex items-center gap-2 text-[15px] font-semibold text-fg', className)}
    >
      <svg viewBox="0 0 32 32" className="size-6" aria-hidden>
        <rect width="32" height="32" rx="7" className="fill-primary" />
        <path
          d="M10 9h7.5a5.5 5.5 0 0 1 0 11H14v3h-4V9Zm4 3.6v3.8h3.3a1.9 1.9 0 0 0 0-3.8H14Z"
          className="fill-primary-fg"
        />
        <rect x="20" y="20" width="4" height="4" rx="1" className="fill-accent" />
      </svg>
      Brandshake
    </span>
  )
}
