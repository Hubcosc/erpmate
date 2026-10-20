import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

export type POStatus = 'draft' | 'issued' | 'partially_received' | 'received'

export interface POItem {
  id: string
  purchase_order_id: string
  item_id: string | null
  item_code?: string
  item_name?: string
  description: string | null
  quantity: number
  received_quantity: number
  remaining_quantity: number
  unit_of_measure: string | null
  unit_price: number
  line_total: number
  is_batch_tracked: boolean
}

export interface PurchaseOrder {
  id: string
  po_number: string
  purchase_requisition_id: string | null
  supplier_id: string | null
  supplier_name: string | null
  status: POStatus
  expected_delivery_date: string | null
  payment_terms: string | null
  delivery_terms: string | null
  notes: string | null
  total_amount: number
  item_count: number
  created_at: string
  updated_at: string
  items: POItem[]
}

export interface POSummary {
  id: string
  po_number: string
  supplier_name: string | null
  status: POStatus
  total_amount: number
  item_count: number
  expected_delivery_date: string | null
  created_at: string
}

export interface UpdatePOInput {
  supplier_id?: string
  expected_delivery_date?: string | null
  payment_terms?: string | null
  delivery_terms?: string | null
  notes?: string | null
  items?: { id: string; unit_price: number }[]
}

// ---- API Functions ----

export const listPOs = (params: { status?: string; search?: string; page?: number; limit?: number }) => {
  const q = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '') q.set(k, String(v)) })
  return apiRequest<ApiEnvelope<POSummary[]>>(`/purchase-orders?${q.toString()}`)
}

export const getPO = (id: string) =>
  apiRequest<PurchaseOrder>(`/purchase-orders/${id}`)

export const updatePO = (id: string, data: UpdatePOInput) =>
  apiRequest<PurchaseOrder>(`/purchase-orders/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })

export const issuePO = (id: string) =>
  apiRequest<PurchaseOrder>(`/purchase-orders/${id}/issue`, {
    method: 'POST',
  })