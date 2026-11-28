import { zodResolver } from '@hookform/resolvers/zod'
import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, Textarea } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import type { Deal, DealInput } from '@/lib/api'
import { paiseToRupeesText, rupeesToPaise } from '@/lib/format'
import { applyServerErrors } from '@/lib/server-errors'
import { PLATFORMS } from '@/lib/vocab'

const optionalInt = (max: number, msg: string) =>
  z
    .string()
    .trim()
    .refine((v) => v === '' || (/^\d+$/.test(v) && Number(v) <= max), msg)

const schema = z
  .object({
    title: z.string().trim().min(1, 'Give the deal a title.').max(120),
    counterparty_name: z.string().trim().max(120),
    counterparty_email: z.union([z.literal(''), z.email('Enter a valid email address.')]),
    counterparty_handle: z.string().trim().max(31),
    amount: z
      .string()
      .refine(
        (v) => v.trim() === '' || rupeesToPaise(v) !== null,
        'Enter an amount in rupees, e.g. 15000.',
      ),
    revisions: optionalInt(20, 'Enter 0–20.'),
    usage: z.enum(['none', 'days', 'perpetual']),
    usage_days: optionalInt(3650, 'Enter 1–3,650 days.'),
    exclusivity_days: optionalInt(3650, 'Enter 0–3,650 days.'),
    notes: z.string().max(5000, 'Notes are limited to 5,000 characters.'),
    deliverables: z
      .array(
        z.object({
          title: z.string().trim().min(1, 'Describe the deliverable.').max(120),
          platform: z.string(),
          due_date: z.string(),
          done: z.boolean(),
        }),
      )
      .max(20),
  })
  .refine((v) => v.counterparty_name !== '' || v.counterparty_handle !== '', {
    path: ['counterparty_name'],
    message: 'Who is this deal with?',
  })
  .refine((v) => v.usage !== 'days' || v.usage_days !== '', {
    path: ['usage_days'],
    message: 'How many days can the brand reuse the content?',
  })

export type DealFormValues = z.infer<typeof schema>

const toNum = (v: string) => (v.trim() === '' ? null : Number(v))

export function toFormValues(d: Partial<DealInput> | Deal | null): DealFormValues {
  const isDeal = !!d && 'counterparty' in d
  const deliverables = (d?.deliverables ?? []).map((x) => ({
    title: x.title,
    platform: x.platform,
    due_date: x.due_date ?? '',
    done: x.done ?? false,
  }))
  const usage_rights_days = d?.usage_rights_days ?? null
  return {
    title: d?.title ?? '',
    counterparty_name: isDeal ? d.counterparty.name : (d?.counterparty_name ?? ''),
    counterparty_email: isDeal ? d.counterparty.email : (d?.counterparty_email ?? ''),
    counterparty_handle: isDeal
      ? (d.counterparty.user?.handle ?? '')
      : (d?.counterparty_handle ?? ''),
    amount: d?.amount_minor != null ? paiseToRupeesText(d.amount_minor) : '',
    revisions: d?.revisions != null ? String(d.revisions) : '',
    usage: d?.usage_perpetual ? 'perpetual' : usage_rights_days != null ? 'days' : 'none',
    usage_days: usage_rights_days != null ? String(usage_rights_days) : '',
    exclusivity_days: d?.exclusivity_days != null ? String(d.exclusivity_days) : '',
    notes: d?.notes ?? '',
    deliverables,
  }
}

function toInput(v: DealFormValues): DealInput {
  return {
    title: v.title,
    counterparty_name: v.counterparty_name,
    counterparty_email: v.counterparty_email,
    counterparty_handle: v.counterparty_handle.replace(/^@/, ''),
    amount_minor: v.amount.trim() === '' ? null : rupeesToPaise(v.amount),
    revisions: toNum(v.revisions),
    usage_perpetual: v.usage === 'perpetual',
    usage_rights_days: v.usage === 'days' ? toNum(v.usage_days) : null,
    exclusivity_days: toNum(v.exclusivity_days),
    notes: v.notes,
    deliverables: v.deliverables,
  }
}

export function DealForm({
  initial,
  submitLabel,
  showDone,
  onSubmit,
  onCancel,
}: {
  initial: DealFormValues
  submitLabel: string
  showDone?: boolean
  onSubmit: (input: DealInput) => Promise<unknown>
  onCancel?: () => void
}) {
  const form = useForm<DealFormValues>({ resolver: zodResolver(schema), defaultValues: initial })
  const rows = useFieldArray({ control: form.control, name: 'deliverables' })
  const usage = useWatch({ control: form.control, name: 'usage' })
  const [formError, setFormError] = useState<string>()
  const errors = form.formState.errors

  const submit = form.handleSubmit(async (v) => {
    setFormError(undefined)
    try {
      await onSubmit(toInput(v))
    } catch (e) {
      setFormError(
        applyServerErrors(form, e, {
          amount_minor: 'amount',
          usage_rights_days: 'usage_days',
        }),
      )
    }
  })

  return (
    <form onSubmit={submit} noValidate className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <CardTitle>Deal</CardTitle>
        </CardHeader>
        <div className="flex flex-col gap-4 px-5 py-5">
          <Field label="Title" hint='e.g. "FitBar protein launch"' error={errors.title?.message}>
            {(p) => <Input {...p} {...form.register('title')} />}
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="With (name)" error={errors.counterparty_name?.message}>
              {(p) => <Input {...p} placeholder="FitBar" {...form.register('counterparty_name')} />}
            </Field>
            <Field label="Their email (optional)" error={errors.counterparty_email?.message}>
              {(p) => <Input {...p} type="email" {...form.register('counterparty_email')} />}
            </Field>
          </div>
          <Field
            label="Their Brandshake handle (optional)"
            hint="Links the deal to their profile if they're on Brandshake."
            error={errors.counterparty_handle?.message}
          >
            {(p) => (
              <Input
                {...p}
                placeholder="@handle"
                autoComplete="off"
                {...form.register('counterparty_handle')}
              />
            )}
          </Field>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>Terms</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              For your records. The brand pays the creator directly; Brandshake never handles the
              money.
            </p>
          </div>
        </CardHeader>
        <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
          <Field label="Fee (₹)" error={errors.amount?.message}>
            {(p) => (
              <Input
                {...p}
                inputMode="decimal"
                placeholder="15000"
                className="tabular-nums"
                {...form.register('amount')}
              />
            )}
          </Field>
          <Field label="Revision rounds" error={errors.revisions?.message}>
            {(p) => (
              <Input {...p} inputMode="numeric" placeholder="1" {...form.register('revisions')} />
            )}
          </Field>
          <Field label="Usage rights" error={errors.usage?.message}>
            {(p) => (
              <Select {...p} {...form.register('usage')}>
                <option value="none">No reuse by the brand</option>
                <option value="days">Brand may reuse for a period</option>
                <option value="perpetual">Brand may reuse forever</option>
              </Select>
            )}
          </Field>
          {usage === 'days' ? (
            <Field label="Usage period (days)" error={errors.usage_days?.message}>
              {(p) => (
                <Input
                  {...p}
                  inputMode="numeric"
                  placeholder="90"
                  {...form.register('usage_days')}
                />
              )}
            </Field>
          ) : (
            <div className="hidden sm:block" />
          )}
          <Field
            label="Exclusivity (days)"
            hint="No competing brands for this long."
            error={errors.exclusivity_days?.message}
          >
            {(p) => (
              <Input
                {...p}
                inputMode="numeric"
                placeholder="0"
                {...form.register('exclusivity_days')}
              />
            )}
          </Field>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Deliverables</CardTitle>
        </CardHeader>
        <div className="flex flex-col gap-3 px-5 py-5">
          {rows.fields.map((row, i) => (
            <div
              key={row.id}
              className="flex items-start gap-2 rounded-md border border-line bg-surface p-3"
            >
              <div className="grid flex-1 gap-3 sm:grid-cols-[1fr_160px_160px]">
                <Field label="What" error={errors.deliverables?.[i]?.title?.message}>
                  {(p) => (
                    <Input
                      {...p}
                      placeholder="Instagram reel"
                      {...form.register(`deliverables.${i}.title`)}
                    />
                  )}
                </Field>
                <Field label="Platform" error={errors.deliverables?.[i]?.platform?.message}>
                  {(p) => (
                    <Select {...p} {...form.register(`deliverables.${i}.platform`)}>
                      {PLATFORMS.map(([v, label]) => (
                        <option key={v} value={v}>
                          {label}
                        </option>
                      ))}
                      <option value="other">Other</option>
                    </Select>
                  )}
                </Field>
                <Field label="Due" error={errors.deliverables?.[i]?.due_date?.message}>
                  {(p) => (
                    <Input {...p} type="date" {...form.register(`deliverables.${i}.due_date`)} />
                  )}
                </Field>
                {showDone && (
                  <label className="flex items-center gap-2 text-[13px] text-fg sm:col-span-3">
                    <input
                      type="checkbox"
                      className="size-4 accent-[var(--accent)]"
                      {...form.register(`deliverables.${i}.done`)}
                    />
                    Done
                  </label>
                )}
              </div>
              <button
                type="button"
                onClick={() => rows.remove(i)}
                aria-label="Remove deliverable"
                className="rounded-md p-1.5 text-subtle transition-colors duration-150 hover:bg-danger-subtle hover:text-danger"
              >
                <Trash2 className="size-4" aria-hidden />
              </button>
            </div>
          ))}
          {rows.fields.length < 20 && (
            <Button
              type="button"
              variant="secondary"
              size="sm"
              className="self-start"
              onClick={() =>
                rows.append({ title: '', platform: 'instagram', due_date: '', done: false })
              }
            >
              <Plus aria-hidden /> Add deliverable
            </Button>
          )}
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Notes</CardTitle>
        </CardHeader>
        <div className="px-5 py-5">
          <Field label="Private notes" hint="Only you can see these." error={errors.notes?.message}>
            {(p) => <Textarea {...p} rows={4} {...form.register('notes')} />}
          </Field>
        </div>
      </Card>

      <div className="flex items-center justify-end gap-3">
        {formError && <span className="mr-auto text-[13px] text-danger">{formError}</span>}
        {onCancel && (
          <Button type="button" variant="secondary" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button type="submit" loading={form.formState.isSubmitting}>
          {submitLabel}
        </Button>
      </div>
    </form>
  )
}
