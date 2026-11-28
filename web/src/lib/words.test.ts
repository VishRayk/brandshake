import { describe, expect, it } from 'vitest'
import { amountInWords, numberToWordsIN } from './words'

describe('amount in words (Indian system)', () => {
  it('uses lakh and crore', () => {
    expect(numberToWordsIN(15000)).toBe('Fifteen Thousand')
    expect(numberToWordsIN(17700)).toBe('Seventeen Thousand Seven Hundred')
    expect(numberToWordsIN(1_25_000)).toBe('One Lakh Twenty Five Thousand')
    expect(numberToWordsIN(10_00_00_000)).toBe('Ten Crore')
    expect(numberToWordsIN(1_77_00_050)).toBe('One Crore Seventy Seven Lakh Fifty')
    expect(numberToWordsIN(119)).toBe('One Hundred Nineteen')
  })
  it('formats rupees and paise', () => {
    expect(amountInWords(1_770_050)).toBe(
      'Rupees Seventeen Thousand Seven Hundred and Fifty Paise Only',
    )
    expect(amountInWords(100)).toBe('Rupees One Only')
  })
})
