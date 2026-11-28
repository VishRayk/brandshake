import { Link } from '@tanstack/react-router'
import { Download, Plus, ReceiptIndianRupee, Settings2 } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { EmptyState } from '@/components/brandshake/empty-state'
import { MoneyAmount } from '@/components/brandshake/money'
import { PageHeader } from '@/components/brandshake/page-header'
import { StatTile } from '@/components/brandshake/stat-tile'
import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { invoiceApi, type InvoiceFilter } from '@/lib/api'
import { cn } from '@/lib/cn'
import { currentFY, invoiceStatus, longDate } from '@/lib/invoices'
import { useBilling, useInvoiceSummary, useInvoices } from '@/lib/queries'

const FILTERS: [InvoiceFilter, string][] = [
  ['', 'All'],
  ['issued', 'Unpaid'],
  ['overdue', 'Overdue'],
  ['paid', 'Paid'],
  ['void', 'Void'],
]

function ExportButton() {
  const fy = currentFY()
  const exp = useMutation({
    mutationFn: () => invoiceApi.exportCsv(fy),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob)
      Object.assign(document.createElement('a'), {
        href: url,
        download: `brandshake-invoices-${fy}.csv`,
      }).click()
      URL.revokeObjectURL(url)
    },
  })
  return (
    <Button
      variant="secondary"
      size="sm"
      loading={exp.isPending}
      onClick={() => exp.mutate()}
      title={`Export FY ${fy} as CSV`}
    >
      <Download aria-hidden /> Export {fy}
    </Button>
  )
}

export function InvoicesPage({ status }: { status: InvoiceFilter }) {
  const { data: billing, isPending: billingPending } = useBilling()
  const { data: summary } = useInvoiceSummary()
  const { data: invoices, isPending } = useInvoices(status)

  if (!billingPending && !billing) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader
          title="Invoices"
          description="GST-ready invoices with a UPI QR code, shared as a link."
        />
        <Card>
          <EmptyState
            icon={ReceiptIndianRupee}
            title="Set up invoicing"
            description="Add your name, state and UPI ID or bank account once. Every invoice uses them, and brands pay you directly."
            action={
              <Link to="/invoices/billing" className={buttonVariants({ size: 'sm' })}>
                Add billing details
              </Link>
            }
          />
        </Card>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Invoices"
        description="Brands pay you directly. Brandshake tracks what's paid and reminds them when it's late."
        actions={
          <>
            <Link
              to="/invoices/billing"
              className={buttonVariants({ variant: 'ghost', size: 'sm' })}
              aria-label="Billing details"
            >
              <Settings2 aria-hidden />
            </Link>
            <ExportButton />
            <Link to="/invoices/new" search={{}} className={buttonVariants({ size: 'sm' })}>
              <Plus aria-hidden /> New invoice
            </Link>
          </>
        }
      />

      {summary && (
        <section className="grid gap-4 sm:grid-cols-4" aria-label="Earnings">
          <StatTile
            label="Unpaid"
            value={<MoneyAmount amountMinor={summary.outstanding_minor} currency="INR" />}
            hint={`${summary.outstanding_count} invoice${summary.outstanding_count === 1 ? '' : 's'}`}
          />
          <StatTile
            label="Overdue"
            value={<MoneyAmount amountMinor={summary.overdue_minor} currency="INR" />}
            hint={
              summary.overdue_count
                ? `${summary.overdue_count} past due · reminders on`
                : 'Nothing overdue'
            }
          />
          <StatTile
            label="Paid this month"
            value={<MoneyAmount amountMinor={summary.paid_this_month_minor} currency="INR" />}
          />
          <StatTile
            label={`Paid in FY ${summary.fy}`}
            value={<MoneyAmount amountMinor={summary.paid_this_fy_minor} currency="INR" />}
          />
        </section>
      )}
      {summary && summary.claimed_paid_count > 0 && (
        <p className="rounded-md border border-info/30 bg-info-subtle px-4 py-2.5 text-[13px] text-fg">
          {summary.claimed_paid_count} brand
          {summary.claimed_paid_count === 1 ? ' says it has' : 's say they have'} paid. Check your
          bank or UPI app and confirm on the invoice.
        </p>
      )}

      <div className="flex flex-wrap gap-1" role="tablist" aria-label="Filter invoices">
        {FILTERS.map(([v, label]) => (
          <Link
            key={v || 'all'}
            to="/invoices"
            search={v ? { status: v } : {}}
            role="tab"
            aria-selected={status === v}
            className={cn(
              'h-8 rounded-md px-3 text-[13px] leading-8 font-medium transition-colors duration-150',
              status === v
                ? 'bg-fg text-background'
                : 'text-muted hover:bg-neutral-subtle hover:text-fg',
            )}
          >
            {label}
          </Link>
        ))}
      </div>

      {isPending && (
        <div className="h-48 animate-pulse rounded-md border border-line bg-raised" aria-busy />
      )}
      {invoices && invoices.length === 0 && (
        <Card>
          <EmptyState
            icon={ReceiptIndianRupee}
            title={status ? 'No invoices here' : 'No invoices yet'}
            description={
              status
                ? 'Try another filter.'
                : 'Create one from scratch, or from a deal with "Create invoice".'
            }
          />
        </Card>
      )}
      {invoices && invoices.length > 0 && (
        <Card className="overflow-x-auto">
          <table className="w-full min-w-[640px] text-[13px]">
            <thead>
              <tr className="border-b border-line text-left text-xs text-subtle">
                <th className="px-4 py-2.5 font-medium">Invoice</th>
                <th className="px-4 py-2.5 font-medium">Billed to</th>
                <th className="px-4 py-2.5 font-medium">Due</th>
                <th className="px-4 py-2.5 font-medium">Status</th>
                <th className="px-4 py-2.5 text-right font-medium">Total</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {invoices.map((inv) => {
                const st = invoiceStatus(inv)
                return (
                  <tr key={inv.id} className="transition-colors duration-150 hover:bg-surface">
                    <td className="px-4 py-3">
                      <Link
                        to="/invoices/$invoiceId"
                        params={{ invoiceId: inv.id }}
                        className="font-mono font-medium text-fg hover:underline"
                      >
                        {inv.number}
                      </Link>
                      <span className="block text-xs text-subtle">{longDate(inv.issue_date)}</span>
                    </td>
                    <td className="px-4 py-3 text-fg">{inv.buyer_name}</td>
                    <td
                      className={cn(
                        'px-4 py-3 text-xs',
                        inv.overdue ? 'font-medium text-danger' : 'text-muted',
                      )}
                    >
                      {longDate(inv.due_date)}
                    </td>
                    <td className="px-4 py-3">
                      <Badge tone={st.tone} dot>
                        {st.label}
                      </Badge>
                    </td>
                    <td className="px-4 py-3 text-right font-medium text-fg">
                      <MoneyAmount amountMinor={inv.total_minor} currency="INR" />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  )
}
