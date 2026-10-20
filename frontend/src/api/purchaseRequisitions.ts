import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

// Statuses
export type PRStatus =
  | 'draft'
  | 'pending_pm'
  | 'pending_finance'
  | 'approved'
  | 'rejected'
  | 'suspended'
  | 'returned_for_correction'

// One line item
export interface PRItem {
  id: string
  pr_id: string
  item_id: string | null
  item_code?: string
  item_name?: string
  description: string | null
  quantity: number
  unit_of_measure: string | null
  estimated_unit_price: number | null
  estimated_total: number | null
  notes: string | null
}

// Stock check alert (backend generates this)
export interface PRStockCheck {
  item_id: string
  item_code: string
  item_name: string
  requested_quantity: number
  available_quantity: number
  shortfall_quantity: number
  sufficient: boolean
}

// The full PR (returned by GET /{id})
export interface PurchaseRequisition {
  id: string
  pr_number: string
  department_id: string
  department_name: string
  requested_by: string
  requester_name: string
  required_date: string | null
  justification: string | null
  status: PRStatus
  preferred_supplier_id: string | null
  total_estimated: number
  item_count: number
  available_actions: string[]
  stock_check: PRStockCheck[]
  suggest_material_requisition: boolean
  created_at: string
  updated_at: string
  items: PRItem[]
}

// The lightweight row (returned by GET /)
export interface PRSummary {
  id: string
  pr_number: string
  department_id: string
  department_name: string
  requested_by: string
  requester_name: string
  required_date: string | null
  justification: string | null
  status: PRStatus
  total_estimated: number
  item_count: number
  available_actions: string[]
  created_at: string
  updated_at: string
}

// Summary counts for tabs
export interface PRStatusSummary {
  total: number
  counts: Record<PRStatus, number>
  action_required: number
}

// History
export interface PRHistory {
  id: string
  old_status: string | null
  new_status: string
  changed_by_name: string
  reason: string | null
  created_at: string
}

// Payloads
export interface CreatePRItemInput {
  item_id?: string
  description?: string
  quantity: number
  unit_of_measure?: string
  estimated_unit_price?: number
  notes?: string
}

export interface CreatePRInput {
  department_id: string
  preferred_supplier_id?: string
  required_date?: string
  justification?: string
  items: CreatePRItemInput[]
}

// API Functions
export const listPRs = (params: {
  status?: string
  q?: string
  department_id?: string
  mine?: boolean
  page?: number
  limit?: number
}) => {
  const q = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '') q.set(k, String(v))
  })
  return apiRequest<ApiEnvelope<PRSummary[]>>(`/purchase-requisitions?${q.toString()}`)
}
export interface PurchaseRequisition {
  pm_rejection_reason: string | null
  finance_rejection_reason: string | null
  suspension_reason: string | null
  return_reason: string | null

}
export const getPRSummary = () =>
  apiRequest<PRStatusSummary>('/purchase-requisitions/summary')

export const getPR = (id: string) =>
  apiRequest<PurchaseRequisition>(`/purchase-requisitions/${id}`)

export const getPRHistory = (id: string) =>
  apiRequest<PRHistory[]>(`/purchase-requisitions/${id}/history`)

export const createPR = (data: CreatePRInput) =>
  apiRequest<PurchaseRequisition>('/purchase-requisitions', {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const updatePR = (id: string, data: CreatePRInput) =>
  apiRequest<PurchaseRequisition>(`/purchase-requisitions/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })

export const deletePR = (id: string) =>
  apiRequest<{ message: string }>(`/purchase-requisitions/${id}`, {
    method: 'DELETE',
  })

export const prAction = (
  id: string,
  action: 'submit' | 'pm-approve' | 'pm-reject' | 'pm-return' | 'finance-approve' | 'finance-suspend' | 'finance-reject',
  reason?: string
) =>
  apiRequest<PurchaseRequisition>(`/purchase-requisitions/${id}/${action}`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })