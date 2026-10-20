import { createMasterApi } from './master-factory'
import { apiRequest } from './client'

export interface Item {
  id: string
  code: string
  name: string
  description: string | null
  category_id: string | null
  category_name?: string
  item_type: string
  unit_of_measure: string
  min_stock_level: number
  max_stock_level: number | null
  reorder_level: number | null
  is_batch_tracked: boolean
  is_expiry_tracked: boolean
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateItemInput {
  code: string
  name: string
  description?: string
  category_id?: string
  item_type: string
  unit_of_measure: string
  min_stock_level: number
  max_stock_level?: number
  reorder_level?: number
  is_batch_tracked: boolean
  is_expiry_tracked: boolean
}

export interface UpdateItemInput {
  name?: string
  description?: string | null
  category_id?: string | null
  unit_of_measure?: string
  min_stock_level?: number
  max_stock_level?: number | null
  reorder_level?: number | null
  is_batch_tracked?: boolean
  is_expiry_tracked?: boolean
  is_active?: boolean
}

export const itemsApi = createMasterApi<Item, CreateItemInput, UpdateItemInput>('/master/items')

/** Lookups — returns allowed item types from backend */
export const getMasterLookups = () =>
  apiRequest<{ item_types: string[]; warehouse_types: string[] }>('/master/lookups')