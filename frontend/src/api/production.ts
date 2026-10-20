import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface ProductionReceiptItem { id: string; production_receipt_id: string; item_id: string; item_code?: string; item_name?: string; batch_id?: string | null; quantity_produced: number; batch_number?: string | null; manufacture_date?: string | null; expiry_date?: string | null; unit_cost?: number | null; quality_status: 'accepted' | 'rejected' }
export interface ProductionReceipt { id: string; receipt_number: string; warehouse_id: string; warehouse_name?: string; recorded_by: string; recorded_by_name?: string; receipt_date: string; notes?: string | null; created_at?: string; items?: ProductionReceiptItem[] }
export interface ProductionReceiptInput {
  warehouse_id: string
  notes?: string | null
  items: Array<{
    item_id: string
    quantity_produced: number
    batch_number?: string | null
    manufacture_date?: string | null
    expiry_date?: string | null
    unit_cost?: number | null
    quality_status?: 'accepted' | 'rejected'
  }>
}

export const listProductionReceipts = (page = 1, limit = 20) => apiRequest<ApiEnvelope<ProductionReceipt[]>>(`/production-receipts?page=${page}&limit=${limit}`)
export const getProductionReceipt = (id: string) => apiRequest<ApiEnvelope<ProductionReceipt>>(`/production-receipts/${id}`)
export const createProductionReceipt = (input: ProductionReceiptInput) => apiRequest<ApiEnvelope<ProductionReceipt>>('/production-receipts', { method: 'POST', body: JSON.stringify(input) })
