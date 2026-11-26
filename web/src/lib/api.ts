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
    throw new ApiError(0, 'network', 'Could not reach Brandshake. Check your connection and try again.')
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
