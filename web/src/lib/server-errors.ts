import type { FieldValues, Path, UseFormReturn } from 'react-hook-form'
import { ApiError } from './api'

/** Copies server field errors ("rates[0].price_minor") onto form fields ("rates.0.price"). */
export function applyServerErrors<T extends FieldValues>(
  form: UseFormReturn<T>,
  err: unknown,
  rename: Record<string, string> = {},
): string | undefined {
  if (!(err instanceof ApiError)) return 'Something went wrong. Please try again.'
  const entries = Object.entries(err.fields)
  if (entries.length === 0) return err.message
  for (const [key, message] of entries) {
    let path = key.replace(/\[(\d+)\]/g, '.$1')
    for (const [from, to] of Object.entries(rename)) {
      path = path.replace(new RegExp(`\\.${from}$`), `.${to}`)
    }
    form.setError(path as Path<T>, { message })
  }
  return undefined
}
