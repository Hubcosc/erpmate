import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

export type AdminFieldType =
  | 'text'
  | 'textarea'
  | 'number'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'select'
  | 'password'
  | 'readonly'

export interface AdminOption {
  value: string
  label: string
}

export interface AdminField {
  name: string
  label: string
  type: AdminFieldType
  required: boolean
  read_only: boolean
  options?: AdminOption[]
  help?: string
  min?: number
  step?: number
  lookup?: string
}

export interface AdminColumn {
  key: string
  label: string
  align?: 'left' | 'right'
  type?: 'text' | 'number' | 'date' | 'datetime' | 'boolean' | 'badge'
}

export interface AdminModel {
  key: string
  label: string
  label_plural: string
  group: string
  description: string
  icon: string
  columns: AdminColumn[]
  fields: AdminField[]
  search?: string[]
  editable: boolean
}

export interface AdminModelsResponse {
  models: AdminModel[]
  groups: string[]
}

export interface AdminLookups {
  roles: AdminOption[]
  departments: AdminOption[]
  item_categories: AdminOption[]
  items: AdminOption[]
  warehouses: AdminOption[]
  suppliers: AdminOption[]
  customers: AdminOption[]
}

export interface AdminCount {
  key: string
  label: string
  group: string
  icon: string
  count: number
}

export type AdminRow = Record<string, unknown>

export interface ListParams {
  page?: number
  limit?: number
  search?: string
}

export const getAdminModels = () => apiRequest<ApiEnvelope<AdminModelsResponse>>('/admin/models')

export const getAdminLookups = () => apiRequest<ApiEnvelope<AdminLookups>>('/admin/lookups')

export const getAdminOverview = () => apiRequest<ApiEnvelope<AdminCount[]>>('/admin/overview')

export const listAdminRecords = (model: string, params: ListParams = {}) => {
  const query = new URLSearchParams()
  if (params.page) query.set('page', String(params.page))
  if (params.limit) query.set('limit', String(params.limit))
  if (params.search) query.set('search', params.search)
  const qs = query.toString()
  return apiRequest<ApiEnvelope<AdminRow[]>>(`/admin/${model}${qs ? `?${qs}` : ''}`)
}

export const getAdminRecord = (model: string, id: string) =>
  apiRequest<ApiEnvelope<AdminRow>>(`/admin/${model}/${id}`)

export const createAdminRecord = (model: string, values: Record<string, unknown>) =>
  apiRequest<ApiEnvelope<AdminRow>>(`/admin/${model}`, {
    method: 'POST',
    body: JSON.stringify(values),
  })

export const updateAdminRecord = (model: string, id: string, values: Record<string, unknown>) =>
  apiRequest<ApiEnvelope<AdminRow>>(`/admin/${model}/${id}`, {
    method: 'PUT',
    body: JSON.stringify(values),
  })

export const deleteAdminRecord = (model: string, id: string) =>
  apiRequest<ApiEnvelope<{ message: string }>>(`/admin/${model}/${id}`, { method: 'DELETE' })
