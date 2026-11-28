import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  billingApi,
  dealApi,
  invoiceApi,
  profileApi,
  type DealStatus,
  type InvoiceFilter,
  type MyProfile,
} from './api'

export const keys = {
  myProfile: ['me', 'profile'] as const,
  publicProfile: (handle: string) => ['profiles', handle] as const,
  deals: ['deals'] as const,
  dealList: (f: { status?: DealStatus; q?: string }) => ['deals', 'list', f] as const,
  deal: (id: string) => ['deals', 'detail', id] as const,
  dealEvents: (id: string) => ['deals', 'events', id] as const,
  dealSummary: ['deals', 'summary'] as const,
}

export function useDeals(f: { status?: DealStatus; q?: string } = {}) {
  return useQuery({
    queryKey: keys.dealList(f),
    queryFn: () => dealApi.list(f),
    select: (r) => r.deals,
  })
}

export function useDeal(id: string) {
  return useQuery({
    queryKey: keys.deal(id),
    queryFn: () => dealApi.get(id),
    select: (r) => r.deal,
  })
}

export function useDealEvents(id: string) {
  return useQuery({
    queryKey: keys.dealEvents(id),
    queryFn: () => dealApi.events(id),
    select: (r) => r.events,
  })
}

export function useDealSummary() {
  return useQuery({
    queryKey: keys.dealSummary,
    queryFn: dealApi.summary,
    select: (r) => r.summary,
  })
}

/** Any deal write invalidates every deal query (lists, summary, events). */
export function useDealMutation<TInput, TResult>(fn: (input: TInput) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => qc.invalidateQueries({ queryKey: keys.deals }),
  })
}

export function useMyProfile() {
  return useQuery({ queryKey: keys.myProfile, queryFn: profileApi.mine })
}

/** A mutation that writes one part of the caller's profile and patches the cache. */
export function useProfileMutation<TInput, TResult>(
  fn: (input: TInput) => Promise<TResult>,
  apply: (prev: MyProfile, result: TResult) => MyProfile,
) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (result) => {
      qc.setQueryData<MyProfile>(keys.myProfile, (prev) =>
        apply(prev ?? { profile: null, socials: [], rates: [], samples: [] }, result),
      )
      qc.invalidateQueries({ queryKey: ['profiles'] })
    },
  })
}

export const invoiceKeys = {
  all: ['invoices'] as const,
  list: (f: InvoiceFilter) => ['invoices', 'list', f] as const,
  detail: (id: string) => ['invoices', 'detail', id] as const,
  events: (id: string) => ['invoices', 'events', id] as const,
  summary: ['invoices', 'summary'] as const,
  billing: ['billing'] as const,
}

export function useBilling(enabled = true) {
  return useQuery({
    queryKey: invoiceKeys.billing,
    queryFn: billingApi.get,
    select: (r) => r.billing,
    enabled,
  })
}

export function useInvoices(f: InvoiceFilter = '') {
  return useQuery({
    queryKey: invoiceKeys.list(f),
    queryFn: () => invoiceApi.list(f),
    select: (r) => r.invoices,
  })
}

export function useInvoice(id: string) {
  return useQuery({
    queryKey: invoiceKeys.detail(id),
    queryFn: () => invoiceApi.get(id),
    select: (r) => r.invoice,
  })
}

export function useInvoiceEvents(id: string) {
  return useQuery({
    queryKey: invoiceKeys.events(id),
    queryFn: () => invoiceApi.events(id),
    select: (r) => r.events,
  })
}

export function useInvoiceSummary(enabled = true) {
  return useQuery({
    queryKey: invoiceKeys.summary,
    queryFn: invoiceApi.summary,
    select: (r) => r.summary,
    enabled,
  })
}

/** Invoice writes can change deals too (paying an invoice closes its deal). */
export function useInvoiceMutation<TInput, TResult>(fn: (input: TInput) => Promise<TResult>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: invoiceKeys.all })
      qc.invalidateQueries({ queryKey: keys.deals })
    },
  })
}
