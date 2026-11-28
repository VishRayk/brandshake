import { MoneyAmount } from '@/components/brandshake/money'
import type { Invoice } from '@/lib/api'
import { longDate } from '@/lib/invoices'
import { amountInWords } from '@/lib/words'

const STATE_CODES: Record<string, string> = {
  'Jammu and Kashmir': '01',
  'Himachal Pradesh': '02',
  Punjab: '03',
  Chandigarh: '04',
  Uttarakhand: '05',
  Haryana: '06',
  Delhi: '07',
  Rajasthan: '08',
  'Uttar Pradesh': '09',
  Bihar: '10',
  Sikkim: '11',
  'Arunachal Pradesh': '12',
  Nagaland: '13',
  Manipur: '14',
  Mizoram: '15',
  Tripura: '16',
  Meghalaya: '17',
  Assam: '18',
  'West Bengal': '19',
  Jharkhand: '20',
  Odisha: '21',
  Chhattisgarh: '22',
  'Madhya Pradesh': '23',
  Gujarat: '24',
  'Dadra and Nagar Haveli and Daman and Diu': '26',
  Maharashtra: '27',
  Karnataka: '29',
  Goa: '30',
  Lakshadweep: '31',
  Kerala: '32',
  'Tamil Nadu': '33',
  Puducherry: '34',
  'Andaman and Nicobar Islands': '35',
  Telangana: '36',
  'Andhra Pradesh': '37',
  Ladakh: '38',
}

const withCode = (state: string) =>
  state ? `${state}${STATE_CODES[state] ? ` (${STATE_CODES[state]})` : ''}` : ''

function Party({ label, lines }: { label: string; lines: (string | false | undefined)[] }) {
  return (
    <div>
      <p className="text-[11px] font-medium tracking-wide text-subtle uppercase">{label}</p>
      <div className="mt-1.5 space-y-0.5 text-[13px] leading-5 text-fg">
        {lines.filter(Boolean).map((l, i) => (
          <p key={i} className={i === 0 ? 'font-semibold' : 'whitespace-pre-line text-muted'}>
            {l}
          </p>
        ))}
      </div>
    </div>
  )
}

function Row({ label, value, strong }: { label: string; value: number; strong?: boolean }) {
  return (
    <div
      className={
        strong
          ? 'flex justify-between border-t border-line pt-2 text-[15px] font-semibold text-fg'
          : 'flex justify-between text-[13px] text-muted'
      }
    >
      <span>{label}</span>
      <MoneyAmount amountMinor={value} currency="INR" className={strong ? '' : 'text-fg'} />
    </div>
  )
}

/** A printable invoice. Used by the creator's view, the public link and print/PDF. */
export function InvoiceDocument({ inv }: { inv: Invoice }) {
  const s = inv.seller
  const taxInvoice = inv.gst_mode !== 'none'
  const half = inv.gst_rate / 2
  return (
    <article className="invoice-doc rounded-md border border-line bg-raised p-6 sm:p-8 print:border-0 print:p-0">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-line pb-5">
        <div>
          <h2 className="text-xl font-semibold text-fg">
            {taxInvoice ? 'Tax Invoice' : 'Invoice'}
          </h2>
          <p className="mt-1 font-mono text-[13px] text-muted">{inv.number}</p>
        </div>
        <dl className="grid grid-cols-[auto_auto] gap-x-4 gap-y-1 text-[13px]">
          <dt className="text-subtle">Invoice date</dt>
          <dd className="text-right text-fg">{longDate(inv.issue_date)}</dd>
          <dt className="text-subtle">Due date</dt>
          <dd className="text-right font-medium text-fg">{longDate(inv.due_date)}</dd>
          {inv.buyer_state && (
            <>
              <dt className="text-subtle">Place of supply</dt>
              <dd className="text-right text-fg">{withCode(inv.buyer_state)}</dd>
            </>
          )}
        </dl>
      </header>

      <div className="grid gap-6 border-b border-line py-5 sm:grid-cols-2">
        <Party
          label="From"
          lines={[
            s.legal_name,
            s.address,
            withCode(s.state),
            s.gstin && `GSTIN ${s.gstin}`,
            s.pan && `PAN ${s.pan}`,
          ]}
        />
        <Party
          label="Bill to"
          lines={[
            inv.buyer_name,
            inv.buyer_address,
            withCode(inv.buyer_state),
            inv.buyer_gstin && `GSTIN ${inv.buyer_gstin}`,
            inv.buyer_email,
          ]}
        />
      </div>

      <div className="overflow-x-auto">
        <table className="w-full min-w-[520px] text-[13px]">
          <thead>
            <tr className="border-b border-line text-left text-xs text-subtle">
              <th className="py-2.5 pr-2 font-medium">#</th>
              <th className="py-2.5 pr-2 font-medium">Description</th>
              <th className="py-2.5 pr-2 font-medium">SAC</th>
              <th className="py-2.5 pr-2 text-right font-medium">Qty</th>
              <th className="py-2.5 pr-2 text-right font-medium">Rate</th>
              <th className="py-2.5 text-right font-medium">Amount</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {inv.items.map((it, i) => (
              <tr key={i}>
                <td className="py-2.5 pr-2 text-subtle tabular-nums">{i + 1}</td>
                <td className="py-2.5 pr-2 text-fg">{it.description}</td>
                <td className="py-2.5 pr-2 font-mono text-xs text-muted">{it.sac_code}</td>
                <td className="py-2.5 pr-2 text-right tabular-nums">{it.quantity}</td>
                <td className="py-2.5 pr-2 text-right">
                  <MoneyAmount amountMinor={it.unit_price_minor} currency="INR" />
                </td>
                <td className="py-2.5 text-right text-fg">
                  <MoneyAmount amountMinor={it.amount_minor} currency="INR" />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="mt-4 flex flex-col items-end gap-1.5">
        <div className="flex w-full max-w-xs flex-col gap-1.5">
          <Row label={taxInvoice ? 'Taxable value' : 'Subtotal'} value={inv.subtotal_minor} />
          {inv.gst_mode === 'intra' && (
            <>
              <Row label={`CGST @ ${half}%`} value={inv.cgst_minor} />
              <Row label={`SGST @ ${half}%`} value={inv.sgst_minor} />
            </>
          )}
          {inv.gst_mode === 'inter' && (
            <Row label={`IGST @ ${inv.gst_rate}%`} value={inv.igst_minor} />
          )}
          <Row label="Total" value={inv.total_minor} strong />
        </div>
        <p className="mt-1 max-w-sm text-right text-xs text-muted">
          {amountInWords(inv.total_minor)}
        </p>
      </div>

      {(s.upi_id || s.bank_account_number) && (
        <section className="mt-6 rounded-md border border-line bg-surface px-4 py-3 text-[13px] print:bg-transparent">
          <p className="text-[11px] font-medium tracking-wide text-subtle uppercase">Pay to</p>
          <div className="mt-1.5 grid gap-1 sm:grid-cols-2">
            {s.upi_id && (
              <p>
                <span className="text-muted">UPI ID</span>{' '}
                <span className="font-mono text-fg">{s.upi_id}</span>
              </p>
            )}
            {s.bank_account_number && (
              <p className="text-fg">
                <span className="text-muted">Bank</span> {s.bank_account_name} · A/c{' '}
                <span className="font-mono">{s.bank_account_number}</span> · IFSC{' '}
                <span className="font-mono">{s.bank_ifsc}</span>
                {s.bank_name && ` · ${s.bank_name}`}
              </p>
            )}
          </div>
        </section>
      )}

      {inv.notes && <p className="mt-4 text-[13px] whitespace-pre-line text-muted">{inv.notes}</p>}

      <footer className="mt-6 border-t border-line pt-3 text-[11px] leading-4 text-subtle">
        {!taxInvoice && <p>Supplier is not registered under GST; no GST is charged.</p>}
        <p>
          Payment is made directly to {s.legal_name}. Created with Brandshake, which does not handle
          payments.
        </p>
      </footer>
    </article>
  )
}
