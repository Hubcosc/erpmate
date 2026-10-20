/** Shared formatting + API helpers for ErpMate frontend pages. */

/** Unwrap the standard backend envelope `{ success, data, meta, error }`. */
export function unwrap<T>(res: unknown): T {
  if (res && typeof res === 'object' && 'data' in (res as Record<string, unknown>)) {
    return (res as { data: T }).data
  }
  return res as T
}

/** Unwrap the envelope and return both the data and the pagination meta. */
export function unwrapPaged<T>(res: unknown): { items: T[]; total: number; page: number; limit: number } {
  const env = res as { data?: T[]; meta?: { page: number; limit: number; total: number } } | null
  const items = Array.isArray(env?.data) ? env.data : []
  return {
    items,
    total: env?.meta?.total ?? items.length,
    page: env?.meta?.page ?? 1,
    limit: env?.meta?.limit ?? items.length,
  }
}

/** Quantity with up to 4 decimals, trailing zeros trimmed (backend uses NUMERIC(18,4)). */
export function fmtQty(value: number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 }).format(value)
}

/** Money with 2 decimals. */
export function fmtMoney(value: number | null | undefined): string {
  if (value === null || value === undefined) return '—'
  return new Intl.NumberFormat(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(value)
}

/** ISO date -> "12 Mar 2026". Empty string for missing values. */
export function fmtDate(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })
}

/** ISO date -> "12 Mar 2026, 14:30". */
export function fmtDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return date.toLocaleString(undefined, {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Today's date as YYYY-MM-DD (for date inputs and report periods). */
export function todayISO(): string {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
}

/** First day of the current month as YYYY-MM-DD. */
export function monthStartISO(): string {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  return `${now.getFullYear()}-${month}-01`
}
