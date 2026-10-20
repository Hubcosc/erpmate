import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

export interface GRNItem {
  id: string
  goods_receipt_note_id: string
  purchase_order_item_id: string
  item_id: string
  item_code: string
  item_name: string
  quantity_received: number
  batch_number: string | null
  unit_cost: number
}

export interface GoodsReceipt {
  id: string
  grn_number: string
  purchase_order_id: string
  po_number: string
  warehouse_id: string
  warehouse_name: string
  received_by: string
  received_by_name: string
  receipt_date: string
  supplier_invoice_reference: string | null
  notes: string | null
  created_at: string
  items: GRNItem[]
}

export const listGRNs = (params: { purchase_order_id?: string; page?: number; limit?: number }) => {
  const q = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v !== undefined && v !== '') q.set(k, String(v)) })
  return apiRequest<ApiEnvelope<GoodsReceipt[]>>(`/goods-receipt-notes?${q.toString()}`) 
}

export const getGRN = (id: string) =>
  apiRequest<GoodsReceipt>(`/goods-receipt-notes/${id}`)

export const createGRN = (data: any) =>
  apiRequest<GoodsReceipt>('/goods-receipt-notes', { 
    method: 'POST',
    body: JSON.stringify(data),
  })