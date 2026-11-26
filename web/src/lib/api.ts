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
  delete: async (password: string) => {
    const headers = new Headers({ 'Content-Type': 'application/json' })
    if (accessToken) headers.set('Authorization', `Bearer ${accessToken}`)
    const res = await fetch(`${BASE_URL}/v1/me`, {
      method: 'DELETE',
      headers,
      body: JSON.stringify({ password }),
    })
    if (!res.ok) throw await parseError(res)
  },
}
