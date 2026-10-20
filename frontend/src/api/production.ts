import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface ProductionReceiptInput { warehouse_id: string; notes?: string; items: Array<{ item_id: string; quantity_produced: number; batch_number?: string; manufacture_date?: string; expiry_date?: string; unit_cost?: number; quality_status?: 'accepted' | 'rejected' }> }
export interface ProductionReceipt { id: string; receipt_number: string; warehouse_id: string; warehouse_name: string; receipt_date: string; recorded_by: string; items: Array<{ id: string; item_id: string; item_code: string; item_name: string; quantity_produced: number; batch_number?: string; expiry_date?: string; unit_cost?: number; quality_status: 'accepted' | 'rejected' }> }

export const listProductionReceipts = (page = 1, limit = 20) => apiRequest<ApiEnvelope<ProductionReceipt[]>>(`/production-receipts?page=${page}&limit=${limit}`)
export const getProductionReceipt = (id: string) => apiRequest<ApiEnvelope<ProductionReceipt>>(`/production-receipts/${id}`)
export const createProductionReceipt = (input: ProductionReceiptInput) => apiRequest<ApiEnvelope<ProductionReceipt>>('/production-receipts', { method: 'POST', body: JSON.stringify(input) })
