import { createMasterApi } from './master-factory'

export interface Department {
  id: string
  name: string
  code: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateDepartmentInput {
  name: string
  code?: string
}

export interface UpdateDepartmentInput {
  name?: string
  code?: string | null
  is_active?: boolean
}

export const departmentsApi = createMasterApi<Department, CreateDepartmentInput, UpdateDepartmentInput>(
  '/master/departments',
)