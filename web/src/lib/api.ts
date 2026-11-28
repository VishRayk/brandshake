const BASE_URL = import.meta.env.VITE_API_URL ?? '/api'

export type Role = 'creator' | 'brand'

export interface User {
  id: string
  email: string
  role: Role
  display_name: string
  created_at: string
}

export interface Session {
  access_token: string
  expires_at: string
  user: User
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields: Record<string, string>

  constructor(status: number, code: string, message: string, fields: Record<string, string> = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

let accessToken: string | null = null

export function setAccessToken(token: string | null) {
  accessToken = token
}

export async function parseError(res: Response): Promise<ApiError> {
  try {
    const body = await res.json()
    const e = body?.error
    if (e && typeof e.code === 'string') {
      return new ApiError(res.status, e.code, e.message ?? 'Request failed.', e.fields ?? {})
    }
  } catch {
    // not JSON
  }
  return new ApiError(res.status, 'http_error', `Request failed (${res.status}).`)
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body !== undefined) headers.set('Content-Type', 'application/json')
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)

  let res: Response
  try {
    res = await fetch(`${BASE_URL}${path}`, { ...init, headers })
  } catch {
    throw new ApiError(
      0,
      'network',
      'Could not reach Brandshake. Check your connection and try again.',
    )
  }
  if (!res.ok) throw await parseError(res)
  return (await res.json()) as T
}

export const api = {
  signup: (body: { email: string; password: string; role: Role; display_name: string }) =>
    request<Session>('/v1/auth/signup', { method: 'POST', body: JSON.stringify(body) }),
  login: (body: { email: string; password: string }) =>
    request<Session>('/v1/auth/login', { method: 'POST', body: JSON.stringify(body) }),
  me: () => request<{ user: User }>('/v1/me'),
}

export interface Profile {
  handle: string
  headline: string
  bio: string
  city: string
  state: string
  website: string
  niches: string[]
  languages: string[]
  open_to_collabs: boolean
  industry: string
  updated_at: string
}

export interface SocialAccount {
  platform: string
  handle: string
  url: string
  followers: number
  verified?: boolean
}

export interface Rate {
  deliverable: string
  price_minor: number
  note: string
}

export interface Sample {
  title: string
  url: string
  platform?: string
  brand_name: string
}

export interface MyProfile {
  profile: Profile | null
  socials: SocialAccount[]
  rates: Rate[]
  samples: Sample[]
}

export interface PublicProfile extends Profile {
  role: Role
  display_name: string
  member_since: string
  socials: SocialAccount[]
  rates: Rate[]
  samples: Sample[]
}

export type ProfileInput = Omit<Profile, 'updated_at'>

const put = <T>(path: string, body: unknown) =>
  request<T>(path, { method: 'PUT', body: JSON.stringify(body) })

export const profileApi = {
  mine: () => request<MyProfile>('/v1/me/profile'),
  save: (p: ProfileInput) => put<{ profile: Profile }>('/v1/me/profile', p),
  saveSocials: (socials: SocialAccount[]) =>
    put<{ socials: SocialAccount[] }>('/v1/me/socials', { socials }),
  saveRates: (rates: Rate[]) => put<{ rates: Rate[] }>('/v1/me/rates', { rates }),
  saveSamples: (samples: Sample[]) => put<{ samples: Sample[] }>('/v1/me/samples', { samples }),
  checkHandle: (handle: string) =>
    request<{ handle: string; available: boolean; reason?: string }>(
      `/v1/handles/${encodeURIComponent(handle)}`,
    ),
  public: (handle: string) =>
    request<{ profile: PublicProfile }>(`/v1/profiles/${encodeURIComponent(handle)}`),
}

export const accountApi = {
  export: () => request<Record<string, unknown>>('/v1/me/export'),
  delete: (password: string) =>
    requestNoContent('/v1/me', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password }),
    }),
}

/* ---------- Deals ---------- */

export type DealStatus =
  'lead' | 'negotiating' | 'confirmed' | 'in_progress' | 'posted' | 'paid' | 'lost'

export interface DealDeliverable {
  title: string
  platform: string
  due_date: string | null
  done: boolean
  done_at: string | null
}

export interface Deal {
  id: string
  title: string
  status: DealStatus
  counterparty: {
    name: string
    email: string
    user: { handle: string; display_name: string; role: Role } | null
  }
  amount_minor: number | null
  currency: 'INR'
  revisions: number | null
  usage_rights_days: number | null
  usage_perpetual: boolean
  exclusivity_days: number | null
  notes: string
  source: 'manual' | 'ai_draft'
  deliverables: DealDeliverable[]
  created_at: string
  updated_at: string
  status_changed_at: string
}

export interface DealInput {
  title: string
  status?: DealStatus
  counterparty_name: string
  counterparty_email: string
  counterparty_handle: string
  amount_minor: number | null
  revisions: number | null
  usage_rights_days: number | null
  usage_perpetual: boolean
  exclusivity_days: number | null
  notes: string
  source?: 'manual' | 'ai_draft'
  deliverables: { title: string; platform: string; due_date: string; done: boolean }[]
}

export interface DealEvent {
  id: number
  kind: 'created' | 'updated' | 'status_changed' | 'deliverable_done' | 'deliverable_reopened'
  from_status: DealStatus | null
  to_status: DealStatus | null
  detail: string
  created_at: string
}

export interface DealSummary {
  counts: Record<DealStatus, number>
  active_count: number
  active_value_minor: number
  awaiting_payment_minor: number
  paid_this_month_minor: number
  overdue_deliverables: number
  due_soon: {
    deal_id: string
    deal_title: string
    counterparty_name: string
    title: string
    platform: string
    due_date: string
    overdue: boolean
  }[]
}

export interface RiskFlag {
  severity: 'low' | 'medium' | 'high'
  term: string
  message: string
}

export interface DealDraftResponse {
  suggestion: DealInput & { status: DealStatus; source: 'ai_draft' }
  summary: string
  open_questions: string[]
  risk_flags: RiskFlag[]
}

async function requestNoContent(path: string, init: RequestInit) {
  const headers = new Headers(init.headers)
  if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
  let res: Response
  try {
    res = await fetch(`${BASE_URL}${path}`, { ...init, headers })
  } catch {
    throw new ApiError(
      0,
      'network',
      'Could not reach Brandshake. Check your connection and try again.',
    )
  }
  if (!res.ok) throw await parseError(res)
}

export const dealApi = {
  list: (f: { status?: DealStatus; q?: string } = {}) => {
    const qs = new URLSearchParams()
    if (f.status) qs.set('status', f.status)
    if (f.q) qs.set('q', f.q)
    const s = qs.toString()
    return request<{ deals: Deal[] }>(`/v1/deals${s ? `?${s}` : ''}`)
  },
  get: (id: string) => request<{ deal: Deal }>(`/v1/deals/${encodeURIComponent(id)}`),
  create: (d: DealInput) =>
    request<{ deal: Deal }>('/v1/deals', { method: 'POST', body: JSON.stringify(d) }),
  update: (id: string, d: DealInput) =>
    put<{ deal: Deal }>(`/v1/deals/${encodeURIComponent(id)}`, d),
  setStatus: (id: string, status: DealStatus) =>
    put<{ deal: Deal }>(`/v1/deals/${encodeURIComponent(id)}/status`, { status }),
  setDone: (id: string, position: number, done: boolean) =>
    put<{ deal: Deal }>(`/v1/deals/${encodeURIComponent(id)}/deliverables/${position}`, { done }),
  remove: (id: string) =>
    requestNoContent(`/v1/deals/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  events: (id: string) =>
    request<{ events: DealEvent[] }>(`/v1/deals/${encodeURIComponent(id)}/events`),
  summary: () => request<{ summary: DealSummary }>('/v1/deals/summary'),
  draft: (text: string) =>
    request<DealDraftResponse>('/v1/deals/draft', {
      method: 'POST',
      body: JSON.stringify({ text }),
    }),
}

/* ---------- Invoices ---------- */

export interface Billing {
  legal_name: string
  address: string
  state: string
  pan: string
  gst_registered: boolean
  gstin: string
  gst_rate: number
  upi_id: string
  bank_account_name: string
  bank_account_number: string
  bank_ifsc: string
  bank_name: string
  invoice_prefix: string
  payment_terms_days: number
  updated_at?: string
}

export interface InvoiceItem {
  description: string
  sac_code: string
  quantity: number
  unit_price_minor: number
  amount_minor: number
}

export type InvoiceStatus = 'issued' | 'paid' | 'void'

export interface Invoice {
  id: string
  number: string
  fy: string
  status: InvoiceStatus
  overdue: boolean
  issue_date: string
  due_date: string
  seller: {
    legal_name: string
    address: string
    state: string
    pan: string
    gstin: string
    upi_id: string
    bank_account_name: string
    bank_account_number: string
    bank_ifsc: string
    bank_name: string
  }
  buyer_name: string
  buyer_email: string
  buyer_address: string
  buyer_state: string
  buyer_gstin: string
  gst_mode: 'none' | 'intra' | 'inter'
  gst_rate: number
  items: InvoiceItem[]
  subtotal_minor: number
  cgst_minor: number
  sgst_minor: number
  igst_minor: number
  total_minor: number
  notes: string
  deal_id?: string | null
  public_url?: string
  upi_uri?: string
  sent_at: string | null
  payer_claimed_paid_at: string | null
  payer_reference: string
  paid_at: string | null
  paid_reference: string
  voided_at: string | null
  reminders_sent: number
  created_at: string
}

export interface InvoiceInput {
  deal_id: string
  buyer_name: string
  buyer_email: string
  buyer_address: string
  buyer_state: string
  buyer_gstin: string
  issue_date: string
  due_date: string
  items: { description: string; sac_code: string; quantity: number; unit_price_minor: number }[]
  notes: string
}

export interface InvoiceSummary {
  outstanding_minor: number
  outstanding_count: number
  overdue_minor: number
  overdue_count: number
  paid_this_month_minor: number
  paid_this_fy_minor: number
  fy: string
  claimed_paid_count: number
}

export interface InvoiceEvent {
  id: number
  kind: 'issued' | 'sent' | 'viewed' | 'payer_claimed_paid' | 'paid' | 'voided' | 'reminder_sent'
  detail: string
  created_at: string
}

export type InvoiceFilter = '' | 'issued' | 'overdue' | 'paid' | 'void'

export const billingApi = {
  get: () => request<{ billing: Billing | null }>('/v1/me/billing'),
  save: (b: Billing) => put<{ billing: Billing }>('/v1/me/billing', b),
}

const post = <T>(path: string, body?: unknown) =>
  request<T>(path, { method: 'POST', body: JSON.stringify(body ?? {}) })

export const invoiceApi = {
  list: (status: InvoiceFilter = '') =>
    request<{ invoices: Invoice[] }>(`/v1/invoices${status ? `?status=${status}` : ''}`),
  get: (id: string) => request<{ invoice: Invoice }>(`/v1/invoices/${encodeURIComponent(id)}`),
  create: (i: InvoiceInput) => post<{ invoice: Invoice }>('/v1/invoices', i),
  summary: () => request<{ summary: InvoiceSummary }>('/v1/invoices/summary'),
  events: (id: string) =>
    request<{ events: InvoiceEvent[] }>(`/v1/invoices/${encodeURIComponent(id)}/events`),
  send: (id: string) => post<{ invoice: Invoice }>(`/v1/invoices/${encodeURIComponent(id)}/send`),
  markPaid: (id: string, body: { paid_on: string; reference: string }) =>
    post<{ invoice: Invoice }>(`/v1/invoices/${encodeURIComponent(id)}/paid`, body),
  void: (id: string) => post<{ invoice: Invoice }>(`/v1/invoices/${encodeURIComponent(id)}/void`),
  exportCsv: async (fy: string) => {
    const headers = new Headers()
    if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
    const res = await fetch(`${BASE_URL}/v1/invoices/export.csv?fy=${encodeURIComponent(fy)}`, {
      headers,
    })
    if (!res.ok) throw await parseError(res)
    return res.blob()
  },
}

export const publicInvoiceApi = {
  get: (token: string) =>
    request<{ invoice: Invoice }>(`/v1/public/invoices/${encodeURIComponent(token)}`),
  claimPaid: (token: string, reference: string) =>
    post<{ invoice: Invoice }>(`/v1/public/invoices/${encodeURIComponent(token)}/claim-paid`, {
      reference,
    }),
}
