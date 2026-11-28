import { describe, expect, it } from 'vitest'
import { computeTotals, currentFY, gstMode, invoiceStatus } from './invoices'

describe('GST maths (mirrors the Go implementation)', () => {
  it('matches the server for every mode', () => {
    expect(computeTotals(1_500_000, 'none', 18)).toEqual({
      subtotal: 1_500_000,
      cgst: 0,
      sgst: 0,
      igst: 0,
      total: 1_500_000,
    })
    expect(computeTotals(1_500_000, 'intra', 18)).toMatchObject({
      cgst: 135_000,
      sgst: 135_000,
      total: 1_770_000,
    })
    expect(computeTotals(1_500_000, 'inter', 18)).toMatchObject({ igst: 270_000, total: 1_770_000 })
    expect(computeTotals(33_333, 'intra', 5)).toMatchObject({ cgst: 833, sgst: 833, total: 34_999 })
    expect(computeTotals(125, 'inter', 18)).toMatchObject({ igst: 23, total: 148 })
  })
  it('picks the GST mode from the states', () => {
    expect(gstMode({ gst_registered: false, state: 'Goa' }, 'Goa')).toBe('none')
    expect(gstMode({ gst_registered: true, state: 'Goa' }, 'Goa')).toBe('intra')
    expect(gstMode({ gst_registered: true, state: 'Goa' }, 'Kerala')).toBe('inter')
  })
})

describe('invoice helpers', () => {
  it('computes the financial year in IST', () => {
    expect(currentFY(new Date('2026-11-28T10:00:00Z'))).toBe('2026-27')
    expect(currentFY(new Date('2027-03-31T17:00:00Z'))).toBe('2026-27')
    expect(currentFY(new Date('2027-03-31T19:00:00Z'))).toBe('2027-28') // 1 April, 00:30 IST
  })
  it('labels status', () => {
    expect(
      invoiceStatus({ status: 'issued', overdue: true, payer_claimed_paid_at: null }).label,
    ).toBe('Overdue')
    expect(
      invoiceStatus({ status: 'issued', overdue: true, payer_claimed_paid_at: 'x' }).label,
    ).toBe('Payment reported')
    expect(
      invoiceStatus({ status: 'paid', overdue: false, payer_claimed_paid_at: null }).label,
    ).toBe('Paid')
  })
})
