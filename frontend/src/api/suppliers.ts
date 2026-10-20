import { createMasterApi } from './master-factory'

export interface Supplier {
  id: string
  code: string | null
  name: string
  contact_person: string | null
  phone: string | null
  email: string | null
  address: string | null
  tin_number: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateSupplierInput {
  code?: string
  name: string
  contact_person?: string
  phone?: string
  email?: string
  address?: string
  tin_number?: string
}

export interface UpdateSupplierInput {
  code?: string | null
  name?: string
  contact_person?: string | null
  phone?: string | null
  email?: string | null
  address?: string | null
  tin_number?: string | null
  is_active?: boolean
}

export const suppliersApi = createMasterApi<
  Supplier,
  CreateSupplierInput,
  UpdateSupplierInput
>('/master/suppliers')