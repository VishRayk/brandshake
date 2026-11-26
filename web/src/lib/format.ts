const compact = new Intl.NumberFormat('en-IN', { notation: 'compact', maximumFractionDigits: 1 })

/** 48200 → "48.2K", 1250000 → "12.5L" (Indian grouping). */
export function formatCount(n: number): string {
  return compact.format(n)
}

/** Rupee text typed by a person ("15,000", "₹1.5k" not supported) → paise, or null if invalid. */
export function rupeesToPaise(text: string): number | null {
  const cleaned = text.replace(/[₹,\s]/g, '')
  if (!/^\d+(\.\d{1,2})?$/.test(cleaned)) return null
  return Math.round(Number(cleaned) * 100)
}

export function paiseToRupeesText(paise: number): string {
  const rupees = paise / 100
  return Number.isInteger(rupees) ? String(rupees) : rupees.toFixed(2)
}

export function monthYear(iso: string): string {
  return new Date(iso).toLocaleDateString('en-IN', { month: 'short', year: 'numeric' })
}
