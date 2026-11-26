import { cn } from '@/lib/cn'

export function Logo({ className }: { className?: string }) {
  return (
    <span
      className={cn('inline-flex items-center gap-2 text-[15px] font-semibold text-fg', className)}
    >
      <svg viewBox="0 0 32 32" className="size-6" aria-hidden>
        <rect width="32" height="32" rx="7" className="fill-primary" />
        <path
          fillRule="evenodd"
          d="M7.5 8H14.5A3.75 3.75 0 0 1 14.5 15.5H15A4.25 4.25 0 0 1 15 24H7.5ZM11 10.6H14.3A1.15 1.15 0 0 1 14.3 12.9H11ZM11 18.1H14.8A1.65 1.65 0 0 1 14.8 21.4H11Z"
          className="fill-primary-fg"
        />
        <rect x="21" y="20.5" width="3.5" height="3.5" rx="1" className="fill-accent" />
      </svg>
      Brandshake
    </span>
  )
}
