import type { Billing, Invoice } from './api'

export const GST_RATES = [0, 5, 18, 40] as const
export const DEFAULT_SAC = '998361'

const roundDiv = (n: number, d: number) => Math.floor((2 * n + d) / (2 * d))

export type GstMode = 'none' | 'intra' | 'inter'

/** Mirrors go/internal/invoice.ModeFor. */
export function gstMode(b: Pick<Billing, 'gst_registered' | 'state'>, buyerState: string): GstMode {
  if (!b.gst_registered) return 'none'
  return buyerState === b.state ? 'intra' : 'inter'
}

/** Mirrors go/internal/invoice.Compute (paise, half-up rounding). */
export function computeTotals(subtotal: number, mode: GstMode, ratePercent: number) {
  const bp = ratePercent * 100
  let cgst = 0
  let sgst = 0
  let igst = 0
  if (mode === 'intra') {
    cgst = roundDiv((subtotal * bp) / 2, 10_000)
    sgst = cgst
  } else if (mode === 'inter') {
    igst = roundDiv(subtotal * bp, 10_000)
  }
  return { subtotal, cgst, sgst, igst, total: subtotal + cgst + sgst + igst }
}

export function invoiceStatus(inv: Pick<Invoice, 'status' | 'overdue' | 'payer_claimed_paid_at'>) {
  if (inv.status === 'paid') return { label: 'Paid', tone: 'success' as const }
  if (inv.status === 'void') return { label: 'Void', tone: 'neutral' as const }
  if (inv.payer_claimed_paid_at) return { label: 'Payment reported', tone: 'info' as const }
  if (inv.overdue) return { label: 'Overdue', tone: 'danger' as const }
  return { label: 'Unpaid', tone: 'warning' as const }
}

/** Current Indian financial year, e.g. "2026-27". */
export function currentFY(now = new Date()): string {
  const d = new Date(now.toLocaleDateString('en-CA', { timeZone: 'Asia/Kolkata' }))
  const start = d.getMonth() < 3 ? d.getFullYear() - 1 : d.getFullYear()
  return `${start}-${String((start + 1) % 100).padStart(2, '0')}`
}

export function todayIST(now = new Date()): string {
  return now.toLocaleDateString('en-CA', { timeZone: 'Asia/Kolkata' })
}

export function longDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })
}
