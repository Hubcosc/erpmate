import { createMasterApi } from './master-factory'

export interface Warehouse {
  id: string
  code: string
  name: string
  warehouse_type: string
  location: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateWarehouseInput {
  code: string
  name: string
  warehouse_type: string
  location?: string
}

export interface UpdateWarehouseInput {
  name?: string
  warehouse_type?: string
  location?: string | null
  is_active?: boolean
}

export const warehousesApi = createMasterApi<
  Warehouse,
  CreateWarehouseInput,
  UpdateWarehouseInput
>('/master/warehouses')