import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { useState } from 'react'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { PageHeader } from '@/components/brandshake/page-header'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { SaveStatus, Select, Switch, Textarea } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { billingApi, type Billing } from '@/lib/api'
import { GST_RATES } from '@/lib/invoices'
import { invoiceKeys, useBilling } from '@/lib/queries'
import { applyServerErrors } from '@/lib/server-errors'
import { STATES } from '@/lib/vocab'
import { useMutation, useQueryClient } from '@tanstack/react-query'

const upper = (v: string) => v.trim().toUpperCase()
const schema = z
  .object({
    legal_name: z.string().trim().min(1, 'Enter the name to show on invoices.').max(120),
    address: z.string().trim().max(300),
    state: z.string().min(1, 'Choose your state.'),
    pan: z
      .string()
      .transform(upper)
      .refine(
        (v) => v === '' || /^[A-Z]{5}[0-9]{4}[A-Z]$/.test(v),
        'Enter a valid PAN, e.g. ABCDE1234F.',
      ),
    gst_registered: z.boolean(),
    gstin: z.string().transform(upper),
    gst_rate: z.coerce.number<string | number>(),
    upi_id: z
      .string()
      .trim()
      .refine(
        (v) => v === '' || /^[a-zA-Z0-9._-]{2,256}@[a-zA-Z][a-zA-Z0-9.-]{1,63}$/.test(v),
        'Enter a valid UPI ID, e.g. yourname@okhdfcbank.',
      ),
    bank_account_name: z.string().trim().max(120),
    bank_account_number: z
      .string()
      .transform((v) => v.replace(/\s/g, ''))
      .refine((v) => v === '' || /^[0-9]{9,18}$/.test(v), 'Enter a 9–18 digit account number.'),
    bank_ifsc: z
      .string()
      .transform(upper)
      .refine(
        (v) => v === '' || /^[A-Z]{4}0[A-Z0-9]{6}$/.test(v),
        'Enter a valid IFSC, e.g. HDFC0001234.',
      ),
    bank_name: z.string().trim().max(80),
    invoice_prefix: z
      .string()
      .transform(upper)
      .refine((v) => /^[A-Z0-9]{1,5}$/.test(v), 'Use 1–5 capital letters or digits.'),
    payment_terms_days: z.coerce.number<string | number>().int().min(0).max(120, 'Up to 120 days.'),
  })
  .refine(
    (v) => !v.gst_registered || /^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$/.test(v.gstin),
    {
      path: ['gstin'],
      message: 'Enter your 15-character GSTIN.',
    },
  )
  .refine((v) => v.upi_id !== '' || v.bank_account_number !== '', {
    path: ['upi_id'],
    message: 'Add a UPI ID or a bank account so brands can pay you.',
  })
type In = z.input<typeof schema>
type Out = z.output<typeof schema>

const empty: In = {
  legal_name: '',
  address: '',
  state: '',
  pan: '',
  gst_registered: false,
  gstin: '',
  gst_rate: 18,
  upi_id: '',
  bank_account_name: '',
  bank_account_number: '',
  bank_ifsc: '',
  bank_name: '',
  invoice_prefix: 'INV',
  payment_terms_days: 15,
}

function BillingForm({ initial }: { initial: Billing | null }) {
  const navigate = useNavigate()
  const qc = useQueryClient()
  const form = useForm<In, unknown, Out>({
    resolver: zodResolver(schema),
    defaultValues: initial ?? empty,
  })
  const registered = useWatch({ control: form.control, name: 'gst_registered' })
  const [formError, setFormError] = useState<string>()
  const save = useMutation({
    mutationFn: billingApi.save,
    onSuccess: (r) => {
      qc.setQueryData(invoiceKeys.billing, r)
      form.reset(r.billing)
    },
  })
  const errors = form.formState.errors
  const first = !initial

  const onSubmit = form.handleSubmit((v) => {
    setFormError(undefined)
    save.mutate(v as Billing, {
      onSuccess: () => first && navigate({ to: '/invoices/new', search: {} }),
      onError: (e) => setFormError(applyServerErrors(form, e)),
    })
  })

  return (
    <form onSubmit={onSubmit} noValidate className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <div>
            <CardTitle>Business details</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              Shown as the "From" section on every invoice.
            </p>
          </div>
        </CardHeader>
        <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <Field
              label="Name on invoices"
              hint="Your full name, or your business name if you have one."
              error={errors.legal_name?.message}
            >
              {(p) => <Input {...p} {...form.register('legal_name')} />}
            </Field>
          </div>
          <div className="sm:col-span-2">
            <Field label="Address (optional)" error={errors.address?.message}>
              {(p) => <Textarea {...p} rows={2} {...form.register('address')} />}
            </Field>
          </div>
          <Field label="State" error={errors.state?.message}>
            {(p) => (
              <Select {...p} {...form.register('state')}>
                <option value="">Select a state</option>
                {STATES.map((s) => (
                  <option key={s}>{s}</option>
                ))}
              </Select>
            )}
          </Field>
          <Field
            label="PAN (recommended)"
            hint="Brands need it to deduct TDS."
            error={errors.pan?.message}
          >
            {(p) => (
              <Input {...p} autoComplete="off" className="uppercase" {...form.register('pan')} />
            )}
          </Field>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>GST</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              Registration is usually needed only above ₹20 lakh a year in services. Confirm with
              your CA; Brandshake doesn't give tax advice.
            </p>
          </div>
        </CardHeader>
        <div className="flex flex-col gap-4 px-5 py-5">
          <Controller
            control={form.control}
            name="gst_registered"
            render={({ field }) => (
              <Switch
                checked={field.value}
                onChange={field.onChange}
                label="I'm registered under GST"
                description="Adds CGST + SGST (same state) or IGST (other state) to your invoices."
              />
            )}
          />
          {registered && (
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="GSTIN" error={errors.gstin?.message}>
                {(p) => (
                  <Input
                    {...p}
                    autoComplete="off"
                    className="uppercase"
                    {...form.register('gstin')}
                  />
                )}
              </Field>
              <Field
                label="GST rate"
                hint="18% for most creator services (SAC 998361)."
                error={errors.gst_rate?.message}
              >
                {(p) => (
                  <Select {...p} {...form.register('gst_rate')}>
                    {GST_RATES.map((r) => (
                      <option key={r} value={r}>
                        {r}%
                      </option>
                    ))}
                  </Select>
                )}
              </Field>
            </div>
          )}
        </div>
      </Card>

      <Card>
        <CardHeader>
          <div>
            <CardTitle>How brands pay you</CardTitle>
            <p className="mt-0.5 text-xs text-muted">
              Printed on invoices. Brands pay you directly; the money never goes through Brandshake.
            </p>
          </div>
        </CardHeader>
        <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <Field
              label="UPI ID"
              hint="Invoices get a scannable UPI QR code."
              error={errors.upi_id?.message}
            >
              {(p) => (
                <Input
                  {...p}
                  placeholder="yourname@okhdfcbank"
                  autoComplete="off"
                  {...form.register('upi_id')}
                />
              )}
            </Field>
          </div>
          <Field label="Account holder name" error={errors.bank_account_name?.message}>
            {(p) => <Input {...p} {...form.register('bank_account_name')} />}
          </Field>
          <Field label="Account number" error={errors.bank_account_number?.message}>
            {(p) => (
              <Input
                {...p}
                inputMode="numeric"
                autoComplete="off"
                {...form.register('bank_account_number')}
              />
            )}
          </Field>
          <Field label="IFSC" error={errors.bank_ifsc?.message}>
            {(p) => (
              <Input
                {...p}
                autoComplete="off"
                className="uppercase"
                {...form.register('bank_ifsc')}
              />
            )}
          </Field>
          <Field label="Bank name (optional)" error={errors.bank_name?.message}>
            {(p) => <Input {...p} {...form.register('bank_name')} />}
          </Field>
        </div>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Invoice defaults</CardTitle>
        </CardHeader>
        <div className="grid gap-4 px-5 py-5 sm:grid-cols-2">
          <Field
            label="Invoice number prefix"
            hint="Numbers look like INV/26-27/0001 and restart each financial year."
            error={errors.invoice_prefix?.message}
          >
            {(p) => (
              <Input
                {...p}
                className="uppercase"
                maxLength={5}
                {...form.register('invoice_prefix')}
              />
            )}
          </Field>
          <Field label="Payment due after (days)" error={errors.payment_terms_days?.message}>
            {(p) => <Input {...p} inputMode="numeric" {...form.register('payment_terms_days')} />}
          </Field>
        </div>
      </Card>

      <div className="flex items-center justify-end gap-3">
        {formError && <span className="mr-auto text-[13px] text-danger">{formError}</span>}
        <SaveStatus
          state={
            save.isPending
              ? 'saving'
              : save.isSuccess && !form.formState.isDirty
                ? 'saved'
                : save.isError
                  ? 'error'
                  : 'idle'
          }
        />
        <Button type="submit" loading={save.isPending}>
          {first ? 'Save and create invoice' : 'Save'}
        </Button>
      </div>
    </form>
  )
}

export function BillingPage() {
  const { data, isPending } = useBilling()
  return (
    <div className="flex flex-col gap-6">
      <Link
        to="/invoices"
        className="flex items-center gap-1.5 self-start text-[13px] text-muted hover:text-fg"
      >
        <ArrowLeft className="size-4" aria-hidden /> Invoices
      </Link>
      <PageHeader
        title="Billing details"
        description="What appears on your invoices and how brands pay you."
      />
      {isPending ? (
        <div className="h-64 animate-pulse rounded-md border border-line bg-raised" aria-busy />
      ) : (
        <BillingForm initial={data ?? null} />
      )}
    </div>
  )
}
