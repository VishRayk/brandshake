import { describe, expect, it } from 'vitest'
import { daysUntil, dueLabel } from './deals'

// 2026-12-01 10:00 IST
const now = new Date('2026-12-01T04:30:00Z')

describe('due dates', () => {
  it('counts calendar days in IST', () => {
    expect(daysUntil('2026-12-01', now)).toBe(0)
    expect(daysUntil('2026-12-04', now)).toBe(3)
    expect(daysUntil('2026-11-28', now)).toBe(-3)
  })
  it('labels due dates', () => {
    expect(dueLabel('2026-12-01', now)).toBe('Due today')
    expect(dueLabel('2026-12-02', now)).toBe('Due tomorrow')
    expect(dueLabel('2026-11-30', now)).toBe('Due yesterday')
    expect(dueLabel('2026-11-28', now)).toBe('3 days overdue')
    expect(dueLabel('2026-12-20', now)).toBe('Due 20 Dec')
  })
})
