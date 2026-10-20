import { apiRequest } from './client'

export interface ApiEnvelope<T> { success: boolean; data: T; meta?: { page: number; limit: number; total: number } }
export interface Department { id: string; name: string; code?: string | null; is_active: boolean }
export interface CatalogItem { id: string; code: string; name: string; description?: string | null; item_type: string; unit_of_measure: string; min_stock_level: number; reorder_level?: number | null; is_batch_tracked: boolean; is_expiry_tracked: boolean; is_active: boolean }
export interface Supplier { id: string; code?: string | null; name: string; is_active: boolean }
export interface Customer { id: string; code?: string | null; name: string; customer_type: 'local' | 'international'; is_active: boolean }
export interface Warehouse { id: string; code: string; name: string; warehouse_type: string; is_active: boolean }

export const listDepartments = () => apiRequest<Department[]>('/master/departments')
export const listItems = (itemType?: string, page = 1, limit = 100) => apiRequest<ApiEnvelope<CatalogItem[]>>(`/master/items?page=${page}&limit=${limit}${itemType ? `&item_type=${encodeURIComponent(itemType)}` : ''}`)
export const listSuppliers = (page = 1, limit = 100) => apiRequest<ApiEnvelope<Supplier[]>>(`/master/suppliers?page=${page}&limit=${limit}`)
export const listCustomers = (page = 1, limit = 100) => apiRequest<ApiEnvelope<Customer[]>>(`/master/customers?page=${page}&limit=${limit}`)
export const listWarehouses = () => apiRequest<Warehouse[]>('/master/warehouses')
