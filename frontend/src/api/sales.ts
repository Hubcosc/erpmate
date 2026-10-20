import { apiBlobRequest, apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface SalesOrderItemInput { item_id: string; quantity: number; unit_price: number }
export interface SalesOrderInput { customer_id: string; required_date?: string | null; notes?: string | null; items: SalesOrderItemInput[] }
export interface SalesOrder { id: string; order_number: string; customer_id: string; customer_name: string; status: string; total_amount: number; items: Array<SalesOrderItemInput & { id: string; reserved_quantity: number; delivered_quantity: number; available_quantity: number }> }
export interface DeliveryInput { recipient_name?: string | null; recipient_signature_reference?: string | null; notes?: string | null; items: Array<{ sales_order_item_id: string; quantity_delivered: number }> }
export interface DeliveryNote { id: string; delivery_number: string; sales_order_id: string; order_number: string; delivery_date: string; recipient_name?: string | null; items: Array<{ id: string; item_id: string; item_code: string; item_name: string; quantity_delivered: number; batch_id?: string | null }> }

export const listSalesOrders = (page = 1, limit = 20, status?: string) => apiRequest<ApiEnvelope<SalesOrder[]>>(`/sales-orders?page=${page}&limit=${limit}${status ? `&status=${encodeURIComponent(status)}` : ''}`)
export const getSalesOrder = (id: string) => apiRequest<ApiEnvelope<SalesOrder>>(`/sales-orders/${id}`)
export const createSalesOrder = (input: SalesOrderInput) => apiRequest<ApiEnvelope<SalesOrder>>('/sales-orders', { method: 'POST', body: JSON.stringify(input) })
export const createDeliveryNote = (orderId: string, input: DeliveryInput) => apiRequest<ApiEnvelope<DeliveryNote>>(`/sales-orders/${orderId}/deliveries`, { method: 'POST', body: JSON.stringify(input) })
export const listDeliveryNotes = (page = 1, limit = 20, orderId?: string) => apiRequest<ApiEnvelope<DeliveryNote[]>>(`/delivery-notes?page=${page}&limit=${limit}${orderId ? `&sales_order_id=${encodeURIComponent(orderId)}` : ''}`)
export const getDeliveryNote = (id: string) => apiRequest<ApiEnvelope<DeliveryNote>>(`/delivery-notes/${id}`)
export const downloadDeliveryNotePDF = (id: string) => apiBlobRequest(`/delivery-notes/${id}/pdf`)
