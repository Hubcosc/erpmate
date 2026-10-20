const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080/api/v1'

let bearerToken: string | null = null

export function setApiToken(token: string | null) {
  bearerToken = token
}

/** The active JWT, from memory or from the persisted session.
 *  Used by API clients that manage their own fetch (e.g. the inventory client). */
export function getApiToken(): string | null {
  if (bearerToken) return bearerToken
  try {
    const raw = window.localStorage.getItem('erpmate.session') ?? window.sessionStorage.getItem('erpmate.session')
    if (!raw) return null
    const session = JSON.parse(raw) as { token?: string }
    return session.token ?? null
  } catch {
    return null
  }
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  if (bearerToken) headers.set('Authorization', `Bearer ${bearerToken}`)

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  })

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    throw new Error(payload?.error?.message ?? `API request failed with status ${response.status}`)
  }

  return payload as T
}

export async function apiBlobRequest(path: string, init?: RequestInit): Promise<Blob> {
  const headers = new Headers(init?.headers)
  if (bearerToken) headers.set('Authorization', `Bearer ${bearerToken}`)
  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers })
  if (!response.ok) {
    const payload = await response.json().catch(() => null)
    throw new Error(payload?.error?.message ?? `API request failed with status ${response.status}`)
  }
  return response.blob()
}
