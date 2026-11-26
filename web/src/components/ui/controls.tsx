import {
  useEffect,
  useRef,
  type ReactNode,
  type SelectHTMLAttributes,
  type TextareaHTMLAttributes,
} from 'react'
import { cn } from '@/lib/cn'

const fieldBase =
  'w-full rounded-md border border-line-strong bg-raised px-3 text-sm text-fg placeholder:text-subtle transition-colors duration-150 focus-visible:border-ring focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/20 aria-[invalid=true]:border-danger disabled:opacity-50'

export function Textarea({ className, ...props }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(fieldBase, 'min-h-24 py-2 leading-5', className)} {...props} />
}

export function Select({ className, children, ...props }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={cn(fieldBase, 'h-9 appearance-none bg-[length:16px] pr-8', className)}
      {...props}
    >
      {children}
    </select>
  )
}

/** Multi-select as toggle chips. */
export function ChipGroup({
  options,
  value,
  onChange,
  max,
  label,
}: {
  options: readonly (readonly [string, string])[]
  value: string[]
  onChange: (next: string[]) => void
  max?: number
  label: string
}) {
  const full = max !== undefined && value.length >= max
  return (
    <div role="group" aria-label={label} className="flex flex-wrap gap-1.5">
      {options.map(([v, text]) => {
        const on = value.includes(v)
        return (
          <button
            key={v}
            type="button"
            aria-pressed={on}
            disabled={!on && full}
            onClick={() => onChange(on ? value.filter((x) => x !== v) : [...value, v])}
            className={cn(
              'h-7 rounded-sm border px-2.5 text-[13px] transition-colors duration-150 disabled:opacity-40',
              on
                ? 'border-fg bg-fg text-background'
                : 'border-line-strong bg-raised text-muted hover:text-fg',
            )}
          >
            {text}
          </button>
        )
      })}
    </div>
  )
}

export function Switch({
  checked,
  onChange,
  label,
  description,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  label: string
  description?: string
}) {
  return (
    <label className="flex cursor-pointer items-start justify-between gap-4">
      <span>
        <span className="block text-[13px] font-medium text-fg">{label}</span>
        {description && <span className="block text-xs text-subtle">{description}</span>}
      </span>
      <input
        type="checkbox"
        role="switch"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="peer sr-only"
      />
      <span
        aria-hidden
        className={cn(
          'relative mt-0.5 h-5 w-9 shrink-0 rounded-full transition-colors duration-150 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ring',
          checked ? 'bg-accent' : 'bg-line-strong',
        )}
      >
        <span
          className={cn(
            'absolute top-0.5 left-0.5 size-4 rounded-full bg-white transition-transform duration-150',
            checked && 'translate-x-4',
          )}
        />
      </span>
    </label>
  )
}

/** Modal built on the native <dialog> element (focus trap and Esc for free). */
export function Dialog({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean
  onClose: () => void
  title: string
  children: ReactNode
}) {
  const ref = useRef<HTMLDialogElement>(null)
  useEffect(() => {
    const d = ref.current
    if (!d) return
    if (open && !d.open) d.showModal()
    if (!open && d.open) d.close()
  }, [open])
  return (
    <dialog
      ref={ref}
      onClose={onClose}
      aria-labelledby="dialog-title"
      className="m-auto w-full max-w-md rounded-lg border border-line bg-raised p-0 text-fg shadow-float backdrop:bg-black/40"
    >
      <div className="px-6 py-5">
        <h2 id="dialog-title" className="text-lg font-semibold">
          {title}
        </h2>
        <div className="mt-3">{children}</div>
      </div>
    </dialog>
  )
}

export function SaveStatus({ state }: { state: 'idle' | 'saving' | 'saved' | 'error' }) {
  if (state === 'idle' || state === 'saving') return null
  return (
    <span
      role="status"
      className={cn('text-xs', state === 'saved' ? 'text-success' : 'text-danger')}
    >
      {state === 'saved' ? 'Saved' : 'Could not save. Check the fields above.'}
    </span>
  )
}
