import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Check, Copy, FileX, Printer } from 'lucide-react'
import { useEffect, useState } from 'react'
import { EmptyState } from '@/components/brandshake/empty-state'
import { Logo } from '@/components/brandshake/logo'
import { MoneyAmount } from '@/components/brandshake/money'
import { InvoiceDocument } from '@/components/invoices/invoice-document'
import { UpiQr } from '@/components/invoices/upi-qr'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Dialog } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { ApiError, publicInvoiceApi, type Invoice } from '@/lib/api'
import { longDate } from '@/lib/invoices'

function CopyText({ text, label }: { text: string; label: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <button
      type="button"
      onClick={async () => {
        await navigator.clipboard.writeText(text)
        setCopied(true)
        setTimeout(() => setCopied(false), 2000)
      }}
      className="inline-flex items-center gap-1 text-xs font-medium text-muted hover:text-fg"
      aria-label={`Copy ${label}`}
    >
      {copied ? (
        <Check className="size-3.5" aria-hidden />
      ) : (
        <Copy className="size-3.5" aria-hidden />
      )}
      {copied ? 'Copied' : 'Copy'}
    </button>
  )
}

function ReportPaid({ token, inv }: { token: string; inv: Invoice }) {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [reference, setReference] = useState('')
  const claim = useMutation({
    mutationFn: () => publicInvoiceApi.claimPaid(token, reference),
    onSuccess: (r) => {
      qc.setQueryData(['public-invoice', token], r)
      setOpen(false)
    },
  })
  const err =
    claim.error instanceof ApiError
      ? claim.error.message
      : claim.error
        ? 'Something went wrong.'
        : undefined
  return (
    <>
      <Button variant="secondary" className="w-full" onClick={() => setOpen(true)}>
        I've paid
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title="Let them know you've paid">
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            claim.mutate()
          }}
        >
          <p className="text-sm text-muted">
            {inv.seller.legal_name} gets an email and confirms once the money arrives. This doesn't
            move any money.
          </p>
          <Field
            label="UPI or bank reference (optional)"
            hint="Helps them find your payment."
            error={err}
          >
            {(p) => (
              <Input
                {...p}
                value={reference}
                onChange={(e) => setReference(e.target.value)}
                maxLength={100}
                autoFocus
              />
            )}
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" size="sm" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" size="sm" loading={claim.isPending}>
              Send
            </Button>
          </div>
        </form>
      </Dialog>
    </>
  )
}

function PayPanel({ token, inv }: { token: string; inv: Invoice }) {
  if (inv.status === 'paid') {
    return (
      <Card className="border-success/30 bg-success-subtle px-5 py-4">
        <p className="flex items-center gap-2 text-[15px] font-semibold text-fg">
          <Check className="size-4 text-success" aria-hidden /> Paid
        </p>
        <p className="mt-1 text-[13px] text-muted">
          Confirmed by {inv.seller.legal_name}
          {inv.paid_at ? ` on ${longDate(inv.paid_at)}` : ''}. Thank you!
        </p>
      </Card>
    )
  }
  if (inv.status === 'void') {
    return (
      <Card className="px-5 py-4">
        <p className="text-[15px] font-semibold text-fg">This invoice was voided</p>
        <p className="mt-1 text-[13px] text-muted">
          No payment is due. Contact {inv.seller.legal_name} if you have questions.
        </p>
      </Card>
    )
  }
  return (
    <Card className="px-5 py-5">
      <p className="text-xs text-muted">Amount due{inv.overdue ? ' (overdue)' : ''}</p>
      <MoneyAmount
        amountMinor={inv.total_minor}
        currency="INR"
        className="text-2xl font-semibold text-fg"
      />
      <p
        className={
          inv.overdue ? 'mt-0.5 text-xs font-medium text-danger' : 'mt-0.5 text-xs text-muted'
        }
      >
        Due {longDate(inv.due_date)}
      </p>

      {inv.upi_uri && (
        <div className="mt-5 flex flex-col items-center gap-3 border-t border-line pt-5">
          <UpiQr uri={inv.upi_uri} />
          <p className="text-center text-xs text-muted">
            Scan with any UPI app: GPay, PhonePe, Paytm or your bank's app.
          </p>
          <a href={inv.upi_uri} className={buttonVariants({ className: 'w-full sm:hidden' })}>
            Pay with UPI app
          </a>
          <p className="flex items-center gap-2 font-mono text-[13px] text-fg">
            {inv.seller.upi_id} <CopyText text={inv.seller.upi_id} label="UPI ID" />
          </p>
        </div>
      )}
      {inv.seller.bank_account_number && (
        <div className="mt-4 border-t border-line pt-4 text-[13px]">
          <p className="text-xs font-medium text-subtle">Or bank transfer</p>
          <p className="mt-1 text-fg">{inv.seller.bank_account_name}</p>
          <p className="flex items-center gap-2 font-mono text-fg">
            {inv.seller.bank_account_number}{' '}
            <CopyText text={inv.seller.bank_account_number} label="account number" />
          </p>
          <p className="font-mono text-muted">IFSC {inv.seller.bank_ifsc}</p>
        </div>
      )}

      <div className="mt-5 border-t border-line pt-5">
        {inv.payer_claimed_paid_at ? (
          <p className="text-[13px] text-muted">
            You reported this payment on {longDate(inv.payer_claimed_paid_at)}.{' '}
            {inv.seller.legal_name} will confirm once it arrives.
          </p>
        ) : (
          <ReportPaid token={token} inv={inv} />
        )}
      </div>
      <p className="mt-4 text-[11px] leading-4 text-subtle">
        You pay {inv.seller.legal_name} directly. Brandshake never handles the money.
      </p>
    </Card>
  )
}

export function PublicInvoicePage({ token }: { token: string }) {
  const { data, isPending, error } = useQuery({
    queryKey: ['public-invoice', token],
    queryFn: () => publicInvoiceApi.get(token),
    retry: (n, e) => !(e instanceof ApiError && e.status === 404) && n < 2,
  })
  const inv = data?.invoice
  useEffect(() => {
    if (!inv) return
    document.title = `Invoice ${inv.number} from ${inv.seller.legal_name}`
    return () => {
      document.title = 'Brandshake: where creators and brands do business'
    }
  }, [inv])

  return (
    <div className="min-h-dvh bg-surface print:bg-white">
      <header className="border-b border-line bg-background print:hidden">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4 sm:px-6">
          <Link to="/" aria-label="Brandshake home">
            <Logo />
          </Link>
          {inv && (
            <Button variant="secondary" size="sm" onClick={() => window.print()}>
              <Printer aria-hidden /> Print / PDF
            </Button>
          )}
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6 print:p-0">
        {isPending && (
          <div className="h-96 animate-pulse rounded-md border border-line bg-raised" aria-busy />
        )}
        {error && (
          <Card>
            <EmptyState
              icon={FileX}
              title={
                error instanceof ApiError && error.status === 404
                  ? 'Invoice not found'
                  : 'Something went wrong'
              }
              description="Check that you opened the full link from the email or message you received."
            />
          </Card>
        )}
        {inv && (
          <div className="grid gap-6 lg:grid-cols-[1fr_300px]">
            <InvoiceDocument inv={inv} />
            <aside className="flex flex-col gap-4 print:hidden lg:sticky lg:top-6 lg:self-start">
              <PayPanel token={token} inv={inv} />
            </aside>
          </div>
        )}
      </main>
    </div>
  )
}
