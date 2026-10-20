import { createMasterApi } from './master-factory'

export interface Category {
  id: string
  name: string
  parent_id: string | null
  created_at: string
  updated_at: string
}

export interface CreateCategoryInput {
  name: string
  parent_id?: string
}

export interface UpdateCategoryInput {
  name?: string
  parent_id?: string | null
}

export const categoriesApi = createMasterApi<
  Category,
  CreateCategoryInput,
  UpdateCategoryInput
>('/master/categories')