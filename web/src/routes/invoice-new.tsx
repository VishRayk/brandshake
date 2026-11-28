import { zodResolver } from '@hookform/resolvers/zod'
import { Link, Navigate, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useFieldArray, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { MoneyAmount } from '@/components/brandshake/money'
import { PageHeader } from '@/components/brandshake/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, Textarea } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { invoiceApi, type Billing, type Deal, type InvoiceInput } from '@/lib/api'
import { paiseToRupeesText, rupeesToPaise } from '@/lib/format'
import { DEFAULT_SAC, computeTotals, gstMode, todayIST } from '@/lib/invoices'
import { useBilling, useDeal, useInvoiceMutation } from '@/lib/queries'
import { applyServerErrors } from '@/lib/server-errors'
import { STATES } from '@/lib/vocab'

const schema = z.object({
  buyer_name: z.string().trim().min(1, 'Who is this invoice for?').max(120),
  buyer_email: z.union([z.literal(''), z.email('Enter a valid email address.')]),
  buyer_address: z.string().trim().max(300),
  buyer_state: z.string(),
  buyer_gstin: z.string().transform((v) => v.trim().toUpperCase()),
  issue_date: z.string(),
  due_date: z.string(),
  notes: z.string().max(1000),
  items: z
    .array(
      z.object({
        description: z.string().trim().min(1, 'Describe the work.').max(200),
        sac_code: z.string().regex(/^[0-9]{4,8}$/, '4–8 digits.'),
        quantity: z.coerce.number<string | number>().int().min(1, 'At least 1.').max(1000),
        price: z.string().refine((v) => rupeesToPaise(v) !== null, 'Amount in rupees.'),
      }),
    )
    .min(1, 'Add at least one line item.')
    .max(20),
})
type In = z.input<typeof schema>
type Out = z.output<typeof schema>

function addDays(iso: string, days: number) {
  const d = new Date(`${iso}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}

function defaults(billing: Billing, deal?: Deal): In {
  const today = todayIST()
  return {
    buyer_name: deal?.counterparty.name ?? '',
    buyer_email: deal?.counterparty.email ?? '',
    buyer_address: '',
    buyer_state: '',
    buyer_gstin: '',
    issue_date: today,
    due_date: addDays(today, billing.payment_terms_days),
    notes: '',
    items: [
      {
        description: deal?.title ?? '',
        sac_code: DEFAULT_SAC,
        quantity: 1,
        price: deal?.amount_minor != null ? paiseToRupeesText(deal.amount_minor) : '',
      },
    ],
  }
}

function InvoiceForm({ billing, deal }: { billing: Billing; deal?: Deal }) {
  const navigate = useNavigate()
  const form = useForm<In, unknown, Out>({
    resolver: zodResolver(schema),
    defaultValues: defaults(billing, deal),
  })
  const rows = useFieldArray({ control: form.control, name: 'items' })
  const [formError, setFormError] = useState<string>()
  const create = useInvoiceMutation((i: InvoiceInput) => invoiceApi.create(i))
  const errors = form.formState.errors

  const items = useWatch({ control: form.control, name: 'items' })
  const buyerState = useWatch({ control: form.control, name: 'buyer_state' })
  const subtotal = items.reduce(
    (n, it) => n + (rupeesToPaise(String(it.price ?? '')) ?? 0) * (Number(it.quantity) || 0),
    0,
  )
  const mode = gstMode(billing, buyerState)
  const totals = computeTotals(subtotal, mode, mode === 'none' ? 0 : billing.gst_rate)

  const submit = form.handleSubmit(async (v) => {
    setFormError(undefined)
    try {
      const { invoice } = await create.mutateAsync({
        deal_id: deal?.id ?? '',
        buyer_name: v.buyer_name,
        buyer_email: v.buyer_email,
        buyer_address: v.buyer_address,
        buyer_state: v.buyer_state,
        buyer_gstin: v.buyer_gstin,
        issue_date: v.issue_date,
        due_date: v.due_date,
        notes: v.notes,
        items: v.items.map((it) => ({
          description: it.description,
          sac_code: it.sac_code,
          quantity: it.quantity,
          unit_price_minor: rupeesToPaise(it.price)!,
        })),
      })
      navigate({ to: '/invoices/$invoiceId', params: { invoiceId: invoice.id } })
    } catch (e) {
      setFormError(applyServerErrors(form, e, { unit_price_minor: 'price' }))
    }
  })

  return (
    <form onSubmit={submit} noValidate className="grid gap-6 lg:grid-cols-[1fr_300px]">
      <div className="flex flex-col gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Bill to</CardTitle>
          </CardHeader>
          <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
            <Field label="Brand or company name" error={errors.buyer_name?.message}>
              {(p) => <Input {...p} {...form.register('buyer_name')} />}
            </Field>
            <Field label="Email (to send the invoice)" error={errors.buyer_email?.message}>
              {(p) => <Input {...p} type="email" {...form.register('buyer_email')} />}
            </Field>
            <Field
              label={billing.gst_registered ? 'State (place of supply)' : 'State (optional)'}
              hint={
                billing.gst_registered
                  ? 'Same state as you: CGST + SGST. Different state: IGST.'
                  : undefined
              }
              error={errors.buyer_state?.message}
            >
              {(p) => (
                <Select {...p} {...form.register('buyer_state')}>
                  <option value="">Select a state</option>
                  {STATES.map((s) => (
                    <option key={s}>{s}</option>
                  ))}
                </Select>
              )}
            </Field>
            <Field label="Their GSTIN (optional)" error={errors.buyer_gstin?.message}>
              {(p) => (
                <Input
                  {...p}
                  className="uppercase"
                  autoComplete="off"
                  {...form.register('buyer_gstin')}
                />
              )}
            </Field>
            <div className="sm:col-span-2">
              <Field label="Address (optional)" error={errors.buyer_address?.message}>
                {(p) => <Textarea {...p} rows={2} {...form.register('buyer_address')} />}
              </Field>
            </div>
          </div>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Line items</CardTitle>
          </CardHeader>
          <div className="flex flex-col gap-3 px-5 py-5">
            {rows.fields.map((row, i) => (
              <div
                key={row.id}
                className="flex items-start gap-2 rounded-md border border-line bg-surface p-3"
              >
                <div className="grid flex-1 gap-3 sm:grid-cols-[1fr_96px_72px_120px]">
                  <Field label="Description" error={errors.items?.[i]?.description?.message}>
                    {(p) => (
                      <Input
                        {...p}
                        placeholder="Instagram Reel for protein launch"
                        {...form.register(`items.${i}.description`)}
                      />
                    )}
                  </Field>
                  <Field label="SAC" error={errors.items?.[i]?.sac_code?.message}>
                    {(p) => (
                      <Input {...p} inputMode="numeric" {...form.register(`items.${i}.sac_code`)} />
                    )}
                  </Field>
                  <Field label="Qty" error={errors.items?.[i]?.quantity?.message}>
                    {(p) => (
                      <Input {...p} inputMode="numeric" {...form.register(`items.${i}.quantity`)} />
                    )}
                  </Field>
                  <Field label="Rate (₹)" error={errors.items?.[i]?.price?.message}>
                    {(p) => (
                      <Input
                        {...p}
                        inputMode="decimal"
                        className="tabular-nums"
                        {...form.register(`items.${i}.price`)}
                      />
                    )}
                  </Field>
                </div>
                {rows.fields.length > 1 && (
                  <button
                    type="button"
                    onClick={() => rows.remove(i)}
                    aria-label="Remove line item"
                    className="mt-6 rounded-md p-1.5 text-subtle transition-colors duration-150 hover:bg-danger-subtle hover:text-danger"
                  >
                    <Trash2 className="size-4" aria-hidden />
                  </button>
                )}
              </div>
            ))}
            {errors.items?.message && <p className="text-xs text-danger">{errors.items.message}</p>}
            {rows.fields.length < 20 && (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                className="self-start"
                onClick={() =>
                  rows.append({ description: '', sac_code: DEFAULT_SAC, quantity: 1, price: '' })
                }
              >
                <Plus aria-hidden /> Add line
              </Button>
            )}
          </div>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Dates and notes</CardTitle>
          </CardHeader>
          <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
            <Field label="Invoice date" error={errors.issue_date?.message}>
              {(p) => (
                <Input {...p} type="date" max={todayIST()} {...form.register('issue_date')} />
              )}
            </Field>
            <Field label="Due date" error={errors.due_date?.message}>
              {(p) => <Input {...p} type="date" {...form.register('due_date')} />}
            </Field>
            <div className="sm:col-span-2">
              <Field
                label="Notes on the invoice (optional)"
                hint="e.g. a PO number or thank-you line."
                error={errors.notes?.message}
              >
                {(p) => <Textarea {...p} rows={2} {...form.register('notes')} />}
              </Field>
            </div>
          </div>
        </Card>
      </div>

      <aside className="flex flex-col gap-4 lg:sticky lg:top-6 lg:self-start">
        <Card>
          <CardHeader>
            <CardTitle>Total</CardTitle>
          </CardHeader>
          <dl className="flex flex-col gap-1.5 px-5 py-4 text-[13px]">
            <div className="flex justify-between text-muted">
              <dt>{mode === 'none' ? 'Subtotal' : 'Taxable value'}</dt>
              <dd className="text-fg">
                <MoneyAmount amountMinor={totals.subtotal} currency="INR" />
              </dd>
            </div>
            {mode === 'intra' && (
              <>
                <div className="flex justify-between text-muted">
                  <dt>CGST @ {billing.gst_rate / 2}%</dt>
                  <dd className="text-fg">
                    <MoneyAmount amountMinor={totals.cgst} currency="INR" />
                  </dd>
                </div>
                <div className="flex justify-between text-muted">
                  <dt>SGST @ {billing.gst_rate / 2}%</dt>
                  <dd className="text-fg">
                    <MoneyAmount amountMinor={totals.sgst} currency="INR" />
                  </dd>
                </div>
              </>
            )}
            {mode === 'inter' && (
              <div className="flex justify-between text-muted">
                <dt>IGST @ {billing.gst_rate}%</dt>
                <dd className="text-fg">
                  <MoneyAmount amountMinor={totals.igst} currency="INR" />
                </dd>
              </div>
            )}
            <div className="mt-1 flex justify-between border-t border-line pt-2 text-base font-semibold text-fg">
              <dt>Total</dt>
              <dd>
                <MoneyAmount amountMinor={totals.total} currency="INR" />
              </dd>
            </div>
          </dl>
          {billing.gst_registered && !buyerState && (
            <p className="border-t border-line px-5 py-2.5 text-xs text-warning">
              Choose the brand's state to calculate GST.
            </p>
          )}
        </Card>
        {formError && <p className="text-[13px] text-danger">{formError}</p>}
        <Button type="submit" size="lg" loading={form.formState.isSubmitting}>
          Create invoice
        </Button>
        <p className="text-xs text-subtle">
          The invoice number is assigned when you create it. Issued invoices can't be edited; void
          one and create a new one if needed.
        </p>
      </aside>
    </form>
  )
}

function FromDeal({ dealId, billing }: { dealId: string; billing: Billing }) {
  const { data: deal, isPending } = useDeal(dealId)
  if (isPending)
    return <div className="h-64 animate-pulse rounded-md border border-line bg-raised" aria-busy />
  return <InvoiceForm billing={billing} deal={deal} />
}

export function NewInvoicePage({ dealId }: { dealId?: string }) {
  const { data: billing, isPending } = useBilling()
  if (isPending)
    return <div className="h-64 animate-pulse rounded-md border border-line bg-raised" aria-busy />
  if (!billing) return <Navigate to="/invoices/billing" />
  return (
    <div className="flex flex-col gap-6">
      <Link
        to="/invoices"
        className="flex items-center gap-1.5 self-start text-[13px] text-muted hover:text-fg"
      >
        <ArrowLeft className="size-4" aria-hidden /> Invoices
      </Link>
      <PageHeader
        title="New invoice"
        description={`From ${billing.legal_name}. Brands pay you directly.`}
      />
      {dealId ? <FromDeal dealId={dealId} billing={billing} /> : <InvoiceForm billing={billing} />}
    </div>
  )
}
