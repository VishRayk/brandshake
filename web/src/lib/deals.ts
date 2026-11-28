import type { DealStatus } from './api'

export const STATUS_ORDER: DealStatus[] = [
  'lead',
  'negotiating',
  'confirmed',
  'in_progress',
  'posted',
  'paid',
  'lost',
]

export const STATUS_LABEL: Record<DealStatus, string> = {
  lead: 'Lead',
  negotiating: 'Negotiating',
  confirmed: 'Confirmed',
  in_progress: 'Creating',
  posted: 'Posted',
  paid: 'Paid',
  lost: 'Lost',
}

export const STATUS_TONE: Record<DealStatus, 'neutral' | 'info' | 'warning' | 'success'> = {
  lead: 'neutral',
  negotiating: 'info',
  confirmed: 'info',
  in_progress: 'info',
  posted: 'warning',
  paid: 'success',
  lost: 'neutral',
}

export const STATUS_HINT: Record<DealStatus, string> = {
  lead: 'First contact, nothing discussed yet',
  negotiating: 'Discussing terms and price',
  confirmed: 'Terms agreed, work not started',
  in_progress: 'Creating the content',
  posted: 'Content is live, payment pending',
  paid: 'Payment received',
  lost: 'Did not go ahead',
}

/** Days from today (IST calendar date) to an ISO date; negative when past. */
export function daysUntil(isoDate: string, now = new Date()): number {
  const today = new Date(now.toLocaleDateString('en-CA', { timeZone: 'Asia/Kolkata' }))
  return Math.round((new Date(isoDate).getTime() - today.getTime()) / 86_400_000)
}

export function dueLabel(isoDate: string, now = new Date()): string {
  const d = daysUntil(isoDate, now)
  if (d < -1) return `${-d} days overdue`
  if (d === -1) return 'Due yesterday'
  if (d === 0) return 'Due today'
  if (d === 1) return 'Due tomorrow'
  if (d < 7) return `Due in ${d} days`
  return `Due ${new Date(isoDate).toLocaleDateString('en-IN', { day: 'numeric', month: 'short' })}`
}
