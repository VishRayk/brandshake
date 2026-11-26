import { describe, expect, it } from 'vitest'
import { formatMoney } from '@/lib/money'
import { ApiError, parseError } from './api'

describe('parseError', () => {
  it('reads the API error envelope', async () => {
    const res = new Response(
      JSON.stringify({
        error: {
          code: 'validation_failed',
          message: 'Some fields are invalid.',
          fields: { email: 'Bad' },
        },
      }),
      { status: 422 },
    )
    const err = await parseError(res)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.status).toBe(422)
    expect(err.code).toBe('validation_failed')
    expect(err.fields).toEqual({ email: 'Bad' })
  })

  it('falls back for non-JSON bodies', async () => {
    const err = await parseError(new Response('<html>bad gateway</html>', { status: 502 }))
    expect(err.code).toBe('http_error')
    expect(err.status).toBe(502)
  })
})

describe('formatMoney', () => {
  it('formats minor units without floating-point drift', () => {
    expect(formatMoney(1_200_000, 'INR')).toBe('₹12,000')
    expect(formatMoney(1_999, 'USD', 'en-US')).toBe('$19.99')
    expect(formatMoney(0, 'INR')).toBe('₹0')
  })
})
