import { describe, expect, it } from 'vitest'
import { paiseToRupeesText, rupeesToPaise } from './format'

describe('rupeesToPaise', () => {
  it('parses rupee amounts into paise', () => {
    expect(rupeesToPaise('15000')).toBe(1_500_000)
    expect(rupeesToPaise('₹ 15,000')).toBe(1_500_000)
    expect(rupeesToPaise('199.99')).toBe(19_999)
  })
  it('rejects invalid input', () => {
    for (const bad of ['', 'abc', '-5', '1.234', '1e5']) expect(rupeesToPaise(bad)).toBeNull()
  })
  it('round-trips', () => {
    expect(paiseToRupeesText(1_500_000)).toBe('15000')
    expect(paiseToRupeesText(19_999)).toBe('199.99')
  })
})
