import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { setAccessToken, type Session, type User } from './api'

const STORAGE_KEY = 'brandshake.session'

export interface AuthState {
  user: User | null
  signIn: (session: Session) => void
  signOut: () => void
}

const AuthContext = createContext<AuthState | null>(null)

function loadSession(): Session | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const session = JSON.parse(raw) as Session
    if (new Date(session.expires_at).getTime() <= Date.now()) {
      localStorage.removeItem(STORAGE_KEY)
      return null
    }
    setAccessToken(session.access_token)
    return session
  } catch {
    return null
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(loadSession)

  const signIn = useCallback((s: Session) => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(s))
    setAccessToken(s.access_token)
    setSession(s)
  }, [])

  const signOut = useCallback(() => {
    localStorage.removeItem(STORAGE_KEY)
    setAccessToken(null)
    setSession(null)
  }, [])

  const value = useMemo(
    () => ({ user: session?.user ?? null, signIn, signOut }),
    [session, signIn, signOut],
  )
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside <AuthProvider>')
  return ctx
}
