const ZERO_DECIMAL = new Set(['JPY', 'KRW'])

/** Formats an amount in minor units (paise, cents). Money is never a float in transit. */
export function formatMoney(amountMinor: number, currency: string, locale = 'en-IN'): string {
  const digits = ZERO_DECIMAL.has(currency) ? 0 : 2
  const major = amountMinor / 10 ** digits
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
    minimumFractionDigits: Number.isInteger(major) ? 0 : digits,
    maximumFractionDigits: digits,
  }).format(major)
}
