import { apiRequest } from './client'

export interface ApiEnvelope<T> {
  success: boolean
  data: T
  meta?: { page: number; limit: number; total: number }
}

export interface User {
  id: string
  employee_id: string | null
  full_name: string
  email: string
  phone: string | null
  department_id: string | null
  department_name?: string
  role_id: string
  role_name: string
  is_active: boolean
  last_login_at: string | null
  created_at: string
  updated_at: string
}

export interface Role {
  id: string
  name: string
  description: string | null
}

export interface CreateUserInput {
  employee_id?: string
  full_name: string
  email: string
  phone?: string
  password: string
  department_id?: string
  role_id: string
}

export interface UpdateUserInput {
  employee_id?: string
  full_name?: string
  phone?: string
  department_id?: string
  role_id?: string
  is_active?: boolean
}

export interface ListUsersParams {
  search?: string
  page?: number
  limit?: number
  role_id?: string
}

export const listUsers = (params: ListUsersParams = {}) => {
  const query = new URLSearchParams()
  if (params.search) query.set('search', params.search)
  if (params.page) query.set('page', String(params.page))
  if (params.limit) query.set('limit', String(params.limit))
  if (params.role_id) query.set('role_id', params.role_id)
  const qs = query.toString()
  return apiRequest<ApiEnvelope<User[]>>(`/users${qs ? `?${qs}` : ''}`)
}

export const getUser = (id: string) =>
  apiRequest<User>(`/users/${id}`)

export const createUser = (data: CreateUserInput) =>
  apiRequest<User>('/users', {
    method: 'POST',
    body: JSON.stringify(data),
  })

export const updateUser = (id: string, data: UpdateUserInput) =>
  apiRequest<User>(`/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })

export const deleteUser = (id: string) =>
  apiRequest<{ message: string }>(`/users/${id}`, {
    method: 'DELETE',
  })

export const listRoles = () =>
  apiRequest<Role[] | ApiEnvelope<Role[]>>('/users/roles').then((res) =>
    Array.isArray(res) ? res : res.data
  )