import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface PurchaseOrderItem { id: string; item_id?: string | null; item_code?: string | null; item_name?: string | null; description?: string | null; quantity: number; received_quantity: number; unit_of_measure?: string | null; unit_price?: number | null; line_total: number }
export interface PurchaseOrder { id: string; po_number: string; purchase_requisition_id: string; pr_number: string; supplier_id?: string | null; supplier_name?: string | null; status: string; order_date: string; expected_delivery_date?: string | null; payment_terms?: string | null; delivery_terms?: string | null; total_amount: number; items?: PurchaseOrderItem[] }
export interface PurchaseOrderUpdate { supplier_id?: string; expected_delivery_date?: string; payment_terms?: string; delivery_terms?: string; notes?: string; items?: Array<{ id: string; unit_price: number }> }
export interface GRNInput { purchase_order_id: string; warehouse_id: string; supplier_invoice_reference?: string; notes?: string; items: Array<{ purchase_order_item_id: string; quantity_received: number; batch_number?: string; manufacture_date?: string; expiry_date?: string; unit_cost: number; location?: string }> }
export interface GoodsReceiptNote { id: string; grn_number: string; purchase_order_id: string; po_number: string; warehouse_id: string; warehouse_name: string; receipt_date: string; supplier_invoice_reference?: string; items: Array<{ id: string; item_id: string; item_code: string; item_name: string; quantity_received: number; batch_number?: string; expiry_date?: string; unit_cost: number; location?: string }> }

export const listPurchaseOrders = (page = 1, limit = 20, status?: string) => apiRequest<ApiEnvelope<PurchaseOrder[]>>(`/purchase-orders?page=${page}&limit=${limit}${status ? `&status=${encodeURIComponent(status)}` : ''}`)
export const createPurchaseOrderFromPR = (prId: string) => apiRequest<ApiEnvelope<PurchaseOrder>>(`/purchase-orders/from-requisition/${prId}`, { method: 'POST' })
export const getPurchaseOrder = (id: string) => apiRequest<ApiEnvelope<PurchaseOrder>>(`/purchase-orders/${id}`)
export const updatePurchaseOrder = (id: string, input: PurchaseOrderUpdate) => apiRequest<ApiEnvelope<PurchaseOrder>>(`/purchase-orders/${id}`, { method: 'PUT', body: JSON.stringify(input) })
export const issuePurchaseOrder = (id: string) => apiRequest<ApiEnvelope<PurchaseOrder>>(`/purchase-orders/${id}/issue`, { method: 'POST' })
export const createGoodsReceiptNote = (input: GRNInput) => apiRequest<ApiEnvelope<GoodsReceiptNote>>('/goods-receipt-notes', { method: 'POST', body: JSON.stringify(input) })
export const listGoodsReceiptNotes = (page = 1, limit = 20, purchaseOrderId?: string) => apiRequest<ApiEnvelope<GoodsReceiptNote[]>>(`/goods-receipt-notes?page=${page}&limit=${limit}${purchaseOrderId ? `&purchase_order_id=${encodeURIComponent(purchaseOrderId)}` : ''}`)
export const getGoodsReceiptNote = (id: string) => apiRequest<ApiEnvelope<GoodsReceiptNote>>(`/goods-receipt-notes/${id}`)
