import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface MaterialRequisitionItem { id?: string; item_id: string; item_code?: string; item_name?: string; quantity: number; issued_qty?: number; available_qty?: number; reserved_qty?: number; stock_sufficient?: boolean; suggest_purchase_requisition?: boolean; notes?: string | null }
export interface MaterialRequisition { id: string; mr_number: string; department_id: string; requested_by: string; status: string; required_date?: string | null; purpose?: string | null; items?: MaterialRequisitionItem[] }
export interface CreateMRInput { department_id: string; required_date?: string | null; purpose?: string | null; items: MaterialRequisitionItem[] }
export interface CreateIssueInput { mr_id: string; issued_to?: string | null; notes?: string | null; items: Array<{ mr_item_id: string; item_id: string; batch_id?: string | null; quantity: number }> }
const base = '/material-requisitions'
const post = <T>(path: string, body?: unknown) => apiRequest<ApiEnvelope<T>>(path, { method: 'POST', ...(body === undefined ? {} : { body: JSON.stringify(body) }) })

export const listMaterialRequisitions = (page = 1, limit = 20, status?: string) => apiRequest<ApiEnvelope<MaterialRequisition[]>>(`${base}?page=${page}&limit=${limit}${status ? `&status=${encodeURIComponent(status)}` : ''}`)
export const getMaterialRequisition = (id: string) => apiRequest<ApiEnvelope<MaterialRequisition>>(`${base}/${id}`)
export const createMaterialRequisition = (input: CreateMRInput) => post<MaterialRequisition>(base, input)
export const submitMaterialRequisition = (id: string) => post<{ message: string }>(`${base}/${id}/submit`)
export const approveMaterialRequisitionPM = (id: string) => post<{ message: string }>(`${base}/${id}/pm-approve`)
export const rejectMaterialRequisitionPM = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/pm-reject`, { reason })
export const getMaterialRequisitionHistory = (id: string) => apiRequest<ApiEnvelope<unknown[]>>(`${base}/${id}/history`)
export const createMaterialIssue = (input: CreateIssueInput) => post(`${base}/issues`, input)
export const getMaterialIssue = (id: string) => apiRequest<ApiEnvelope<unknown>>(`${base}/issues/${id}`)
export const listIssuesForMaterialRequisition = (id: string) => apiRequest<ApiEnvelope<unknown[]>>(`${base}/${id}/issues`)
export const acknowledgeMaterialIssue = (id: string) => post<{ message: string }>(`${base}/issues/${id}/acknowledge`)
