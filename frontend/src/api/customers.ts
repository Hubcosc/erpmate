import { createMasterApi } from './master-factory'

export interface Customer {
  id: string
  code: string | null
  name: string
  customer_type: string
  contact_person: string | null
  phone: string | null
  email: string | null
  address: string | null
  tin_number: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateCustomerInput {
  code?: string
  name: string
  customer_type: string
  contact_person?: string
  phone?: string
  email?: string
  address?: string
  tin_number?: string
}

export interface UpdateCustomerInput {
  code?: string | null
  name?: string
  customer_type?: string
  contact_person?: string | null
  phone?: string | null
  email?: string | null
  address?: string | null
  tin_number?: string | null
  is_active?: boolean
}

export const customersApi = createMasterApi<
  Customer,
  CreateCustomerInput,
  UpdateCustomerInput
>('/master/customers')