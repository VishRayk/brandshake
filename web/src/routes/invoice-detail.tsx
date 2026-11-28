import { Link } from '@tanstack/react-router'
import { ArrowLeft, Ban, Check, Copy, Printer, Send } from 'lucide-react'
import { useState } from 'react'
import { InvoiceDocument } from '@/components/invoices/invoice-document'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog } from '@/components/ui/controls'
import { Field, Input } from '@/components/ui/field'
import { ApiError, invoiceApi, type Invoice, type InvoiceEvent } from '@/lib/api'
import { invoiceStatus, longDate, todayIST } from '@/lib/invoices'
import { useInvoice, useInvoiceEvents, useInvoiceMutation } from '@/lib/queries'

function eventText(e: InvoiceEvent): string {
  switch (e.kind) {
    case 'issued':
      return 'Invoice created'
    case 'sent':
      return `Emailed to ${e.detail}`
    case 'viewed':
      return 'Opened by the brand'
    case 'payer_claimed_paid':
      return e.detail ? `Brand reported payment (ref ${e.detail})` : 'Brand reported payment'
    case 'paid':
      return e.detail ? `Marked paid (ref ${e.detail})` : 'Marked paid'
    case 'voided':
      return 'Voided'
    case 'reminder_sent':
      return `Reminder emailed (${e.detail})`
  }
}

function errorText(e: unknown) {
  return e instanceof ApiError
    ? (Object.values(e.fields)[0] ?? e.message)
    : e
      ? 'Something went wrong.'
      : undefined
}

function MarkPaid({ inv }: { inv: Invoice }) {
  const [open, setOpen] = useState(false)
  const [paidOn, setPaidOn] = useState(todayIST())
  const [reference, setReference] = useState(inv.payer_reference)
  const pay = useInvoiceMutation(() => invoiceApi.markPaid(inv.id, { paid_on: paidOn, reference }))
  return (
    <>
      <Button size="sm" onClick={() => setOpen(true)}>
        <Check aria-hidden /> Mark paid
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title={`Mark ${inv.number} as paid?`}>
        <form
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            pay.mutate(undefined, { onSuccess: () => setOpen(false) })
          }}
        >
          <p className="text-sm text-muted">
            Only do this once the money is in your account.
            {inv.deal_id ? ' The linked deal moves to Paid.' : ''}
          </p>
          <Field label="Received on">
            {(p) => (
              <Input
                {...p}
                type="date"
                max={todayIST()}
                value={paidOn}
                onChange={(e) => setPaidOn(e.target.value)}
              />
            )}
          </Field>
          <Field label="UPI or bank reference (optional)" error={errorText(pay.error)}>
            {(p) => (
              <Input
                {...p}
                value={reference}
                onChange={(e) => setReference(e.target.value)}
                maxLength={100}
              />
            )}
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" size="sm" onClick={() => setOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" size="sm" loading={pay.isPending}>
              Mark paid
            </Button>
          </div>
        </form>
      </Dialog>
    </>
  )
}

function VoidInvoice({ inv }: { inv: Invoice }) {
  const [open, setOpen] = useState(false)
  const v = useInvoiceMutation(() => invoiceApi.void(inv.id))
  return (
    <>
      <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
        <Ban aria-hidden /> Void
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title={`Void ${inv.number}?`}>
        <p className="text-sm text-muted">
          The invoice stays in your records marked void, and its number is not reused. The brand's
          link will show it as void.
        </p>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="secondary" size="sm" onClick={() => setOpen(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            size="sm"
            loading={v.isPending}
            onClick={() => v.mutate(undefined, { onSuccess: () => setOpen(false) })}
          >
            Void invoice
          </Button>
        </div>
      </Dialog>
    </>
  )
}

export function InvoiceDetailPage({ invoiceId }: { invoiceId: string }) {
  const { data: inv, isPending, error } = useInvoice(invoiceId)
  const { data: events } = useInvoiceEvents(invoiceId)
  const send = useInvoiceMutation(() => invoiceApi.send(invoiceId))
  const [copied, setCopied] = useState(false)

  const back = (
    <Link
      to="/invoices"
      className="flex items-center gap-1.5 self-start text-[13px] text-muted hover:text-fg print:hidden"
    >
      <ArrowLeft className="size-4" aria-hidden /> Invoices
    </Link>
  )
  if (isPending)
    return <div className="h-96 animate-pulse rounded-md border border-line bg-raised" aria-busy />
  if (error || !inv) {
    return (
      <div className="flex flex-col gap-4">
        {back}
        <p className="text-sm text-muted">
          {error instanceof ApiError && error.status === 404
            ? 'This invoice does not exist.'
            : 'This invoice could not be loaded.'}
        </p>
      </div>
    )
  }

  const st = invoiceStatus(inv)
  const open = inv.status === 'issued'
  const sendError = errorText(send.error)

  return (
    <div className="flex flex-col gap-6">
      {back}
      <div className="flex flex-wrap items-center justify-between gap-3 print:hidden">
        <div className="flex items-center gap-2">
          <h1 className="font-mono text-xl font-semibold text-fg">{inv.number}</h1>
          <Badge tone={st.tone} dot>
            {st.label}
          </Badge>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={async () => {
              await navigator.clipboard.writeText(inv.public_url ?? '')
              setCopied(true)
              setTimeout(() => setCopied(false), 2000)
            }}
          >
            {copied ? <Check aria-hidden /> : <Copy aria-hidden />}{' '}
            {copied ? 'Link copied' : 'Copy link'}
          </Button>
          {open && (
            <Button
              variant="secondary"
              size="sm"
              loading={send.isPending}
              disabled={!inv.buyer_email}
              onClick={() => send.mutate(undefined)}
              title={
                inv.buyer_email ? `Email to ${inv.buyer_email}` : 'Add the brand’s email to send it'
              }
            >
              <Send aria-hidden /> {inv.sent_at ? 'Send again' : 'Send by email'}
            </Button>
          )}
          <Button variant="secondary" size="sm" onClick={() => window.print()}>
            <Printer aria-hidden /> Print / PDF
          </Button>
          {open && <VoidInvoice inv={inv} />}
          {open && <MarkPaid inv={inv} />}
        </div>
      </div>
      {sendError && <p className="text-[13px] text-danger print:hidden">{sendError}</p>}
      {inv.payer_claimed_paid_at && open && (
        <p className="rounded-md border border-info/30 bg-info-subtle px-4 py-2.5 text-[13px] text-fg print:hidden">
          {inv.buyer_name} says they paid on {longDate(inv.payer_claimed_paid_at)}
          {inv.payer_reference && ` (ref ${inv.payer_reference})`}. Check your bank or UPI app, then
          mark it paid.
        </p>
      )}

      <div className="grid gap-6 lg:grid-cols-[1fr_280px]">
        <InvoiceDocument inv={inv} />
        <aside className="flex flex-col gap-6 print:hidden">
          <Card>
            <CardHeader>
              <CardTitle>Activity</CardTitle>
            </CardHeader>
            <ol className="flex flex-col gap-3 px-5 py-4">
              {(events ?? []).map((e) => (
                <li key={e.id} className="text-[13px]">
                  <p className="text-fg">{eventText(e)}</p>
                  <p className="text-xs text-subtle">
                    {new Date(e.created_at).toLocaleString('en-IN', {
                      day: 'numeric',
                      month: 'short',
                      hour: 'numeric',
                      minute: '2-digit',
                    })}
                  </p>
                </li>
              ))}
            </ol>
          </Card>
          {inv.deal_id && (
            <Link
              to="/deals/$dealId"
              params={{ dealId: inv.deal_id }}
              className="text-[13px] font-medium text-muted hover:text-fg"
            >
              View the linked deal →
            </Link>
          )}
          {open && (
            <p className="text-xs text-subtle">
              Overdue reminders go to the brand automatically 1, 7 and 14 days after the due date
              {inv.buyer_email ? '' : ' (needs their email)'}.
            </p>
          )}
        </aside>
      </div>
    </div>
  )
}
