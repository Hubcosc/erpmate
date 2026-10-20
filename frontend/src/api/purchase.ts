import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface PRItem { id?: string; item_id?: string | null; description?: string | null; quantity: number; unit_of_measure?: string | null; estimated_unit_price?: number | null; estimated_total?: number | null; notes?: string | null }
export interface PRStockCheck { item_id: string; item_code: string; item_name: string; requested_quantity: number; available_quantity: number; sufficient: boolean }
export interface PurchaseRequisition { id: string; pr_number: string; department_id: string; department_name?: string; requested_by: string; requester_name?: string; status: string; required_date?: string | null; justification?: string | null; preferred_supplier_id?: string | null; return_reason?: string | null; stock_check?: PRStockCheck[]; suggest_material_requisition?: boolean; items?: PRItem[] }
export interface CreatePRInput { department_id: string; preferred_supplier_id?: string | null; required_date?: string | null; justification?: string | null; items: PRItem[] }
const base = '/purchase-requisitions'
const post = <T>(path: string, body?: unknown) => apiRequest<ApiEnvelope<T>>(path, { method: 'POST', ...(body === undefined ? {} : { body: JSON.stringify(body) }) })

export const listPurchaseRequisitions = (page = 1, limit = 20, status?: string) => apiRequest<ApiEnvelope<PurchaseRequisition[]>>(`${base}?page=${page}&limit=${limit}${status ? `&status=${encodeURIComponent(status)}` : ''}`)
export const getPurchaseRequisition = (id: string) => apiRequest<ApiEnvelope<PurchaseRequisition>>(`${base}/${id}`)
export const createPurchaseRequisition = (input: CreatePRInput) => post<PurchaseRequisition>(base, input)
export const updatePurchaseRequisition = (id: string, input: CreatePRInput) => apiRequest<ApiEnvelope<PurchaseRequisition>>(`${base}/${id}`, { method: 'PUT', body: JSON.stringify(input) })
export const submitPurchaseRequisition = (id: string) => post<{ message: string }>(`${base}/${id}/submit`)
export const approvePurchaseRequisitionPM = (id: string) => post<{ message: string }>(`${base}/${id}/pm-approve`)
export const rejectPurchaseRequisitionPM = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/pm-reject`, { reason })
export const returnPurchaseRequisitionPM = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/pm-return`, { reason })
export const approvePurchaseRequisitionFinance = (id: string) => post<{ message: string }>(`${base}/${id}/finance-approve`)
export const suspendPurchaseRequisitionFinance = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/finance-suspend`, { reason })
export const rejectPurchaseRequisitionFinance = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/finance-reject`, { reason })
export const getPurchaseRequisitionHistory = (id: string) => apiRequest<ApiEnvelope<unknown[]>>(`${base}/${id}/history`)
