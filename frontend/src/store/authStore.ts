export interface AuthUser {
  id: string
  full_name: string
  email: string
  role_name: string
  department_id?: string | null
  department_name?: string | null
}

export interface AuthSession {
  token: string
  user: AuthUser
}

const STORAGE_KEY = 'erpmate.session'

export function loadSession(): AuthSession | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY) ?? window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const session = JSON.parse(raw) as AuthSession
    if (!session.token || !session.user?.id || !session.user?.role_name) return null
    return session
  } catch {
    return null
  }
}

export function saveSession(session: AuthSession, remember: boolean): void {
  clearSession()
  const storage = remember ? window.localStorage : window.sessionStorage
  storage.setItem(STORAGE_KEY, JSON.stringify(session))
}

export function clearSession(): void {
  window.localStorage.removeItem(STORAGE_KEY)
  window.sessionStorage.removeItem(STORAGE_KEY)
}
