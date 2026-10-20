import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface MaterialRequisitionItem { id?: string; item_id: string; item_code?: string; item_name?: string; quantity: number; issued_qty?: number; available_qty?: number; reserved_qty?: number; stock_sufficient?: boolean; suggest_purchase_requisition?: boolean; notes?: string | null }
export interface MaterialRequisition { id: string; mr_number: string; department_id: string; department_name?: string; requested_by: string; requester_name?: string; status: string; required_date?: string | null; purpose?: string | null; pm_approved_by?: string | null; pm_approved_at?: string | null; pm_rejection_reason?: string | null; created_at?: string; updated_at?: string; items?: MaterialRequisitionItem[] }
export interface CreateMRInput { department_id: string; required_date?: string | null; purpose?: string | null; items: Array<{ item_id: string; quantity: number; notes?: string | null }> }
export interface CreateIssueInput { mr_id: string; issued_to?: string | null; notes?: string | null; items: Array<{ mr_item_id: string; item_id: string; batch_id?: string | null; quantity: number }> }
export interface MaterialIssueItem { id: string; issue_id: string; mr_item_id: string; item_id: string; item_code?: string; item_name?: string; batch_id?: string | null; quantity: number }
export interface MaterialIssue { id: string; issue_number: string; mr_id: string; mr_number?: string; issued_by: string; issued_by_name?: string; issued_to?: string | null; issued_to_name?: string | null; issued_date: string; notes?: string | null; created_at?: string; acknowledged_by?: string | null; acknowledged_at?: string | null; items?: MaterialIssueItem[] }
export interface StatusHistory { id: string; entity_type: string; entity_id: string; old_status?: string | null; new_status: string; changed_by?: string | null; changed_by_name?: string; reason?: string | null; created_at: string }
const base = '/material-requisitions'
const post = <T>(path: string, body?: unknown) => apiRequest<ApiEnvelope<T>>(path, { method: 'POST', ...(body === undefined ? {} : { body: JSON.stringify(body) }) })

export const listMaterialRequisitions = (page = 1, limit = 20, status?: string) => apiRequest<ApiEnvelope<MaterialRequisition[]>>(`${base}?page=${page}&limit=${limit}${status ? `&status=${encodeURIComponent(status)}` : ''}`)
export const getMaterialRequisition = (id: string) => apiRequest<ApiEnvelope<MaterialRequisition>>(`${base}/${id}`)
export const createMaterialRequisition = (input: CreateMRInput) => post<MaterialRequisition>(base, input)
export const submitMaterialRequisition = (id: string) => post<{ message: string }>(`${base}/${id}/submit`)
export const approveMaterialRequisitionPM = (id: string) => post<{ message: string }>(`${base}/${id}/pm-approve`)
export const rejectMaterialRequisitionPM = (id: string, reason: string) => post<{ message: string }>(`${base}/${id}/pm-reject`, { reason })
export const getMaterialRequisitionHistory = (id: string) => apiRequest<ApiEnvelope<StatusHistory[]>>(`${base}/${id}/history`)
export const createMaterialIssue = (input: CreateIssueInput) => post<MaterialIssue>(`${base}/issues`, input)
export const getMaterialIssue = (id: string) => apiRequest<ApiEnvelope<MaterialIssue>>(`${base}/issues/${id}`)
export const listIssuesForMaterialRequisition = (id: string) => apiRequest<ApiEnvelope<MaterialIssue[]>>(`${base}/${id}/issues`)
export const acknowledgeMaterialIssue = (id: string) => post<{ message: string }>(`${base}/issues/${id}/acknowledge`)
