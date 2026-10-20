import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

export interface ListParams {
  search?: string
  page?: number
  limit?: number
  is_active?: boolean
  [key: string]: string | number | boolean | undefined
}

export interface CrudApi<T, CreateInput, UpdateInput> {
  list: (params?: ListParams) => Promise<ApiEnvelope<T[]>>
  get: (id: string) => Promise<T>
  create: (data: CreateInput) => Promise<T>
  update: (id: string, data: UpdateInput) => Promise<T>
  remove: (id: string) => Promise<void>
}

/**
 * Builds a standard CRUD API for a master entity.
 * @param path - e.g. '/master/departments'
 */
export function createMasterApi<T, CreateInput, UpdateInput = Partial<CreateInput>>(
  path: string,
): CrudApi<T, CreateInput, UpdateInput> {
  const buildQuery = (params?: ListParams) => {
    if (!params) return ''
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v === undefined || v === null || v === '') return
      q.set(k, String(v))
    })
    const s = q.toString()
    return s ? `?${s}` : ''
  }

  return {
    list: (params) =>
      apiRequest<ApiEnvelope<T[]>>(`${path}${buildQuery(params)}`),

    get: (id) =>
      apiRequest<T>(`${path}/${id}`),

    create: (data) =>
      apiRequest<T>(path, { method: 'POST', body: JSON.stringify(data) }),

    update: (id, data) =>
      apiRequest<T>(`${path}/${id}`, { method: 'PUT', body: JSON.stringify(data) }),

    remove: async (id) => {
      await apiRequest<void>(`${path}/${id}`, { method: 'DELETE' })
    },
  }
}