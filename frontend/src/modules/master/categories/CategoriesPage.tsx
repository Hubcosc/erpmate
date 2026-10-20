import { useEffect, useState } from 'react'
import MasterCrudPage, { type MasterCrudConfig } from '../MasterCrudPage'
import {
  categoriesApi,
  type Category,
  type CreateCategoryInput,
  type UpdateCategoryInput,
} from '../../../api/categories'

function CategoriesPageInner() {
  const [categories, setCategories] = useState<Category[]>([])

  // Load all categories once so we can show parent names + build dropdown options
  useEffect(() => {
    void categoriesApi
      .list({ page: 1, limit: 100 })
      .then((res) => setCategories(res.data ?? []))
      .catch(() => setCategories([]))
  }, [])

  // Build a map id → name for quick parent name lookups
  const nameById: Record<string, string> = {}
  categories.forEach((c) => {
    nameById[c.id] = c.name
  })

  const config: MasterCrudConfig<Category, CreateCategoryInput, UpdateCategoryInput> = {
    title: 'Item Categories',
    entityLabel: 'Category',
    api: categoriesApi,
    searchPlaceholder: 'Search categories by name…',

    getLabel: (row: Category) => row.name,

    columns: [
      { key: 'name', label: 'Name' },
      {
        key: 'parent_id',
        label: 'Parent',
        render: (row: Category) => (row.parent_id ? nameById[row.parent_id] ?? '—' : 'Top level'),
      },
      {
        key: 'created_at',
        label: 'Created',
        render: (row: Category) => new Date(row.created_at).toLocaleDateString(),
      },
    ],

    fields: [
      {
        key: 'name',
        label: 'Name',
        type: 'text',
        required: true,
        placeholder: 'e.g. Raw Materials',
        span: 2,
      },
        {
        key: 'parent_id',
        label: 'Parent Category',
        type: 'select',
        placeholder: 'Top level (no parent)',
        hint: 'Tip: reload the page after adding a new category to refresh this list',
        options: [
            { value: '', label: 'Top level (no parent)' },
            ...categories.map((c) => ({ value: c.id, label: c.name })),
        ],
        span: 2,
        },
    ],

    toCreatePayload: (v) => ({
      name: String(v.name ?? '').trim(),
      parent_id: v.parent_id ? String(v.parent_id) : undefined,
    }),

    toUpdatePayload: (v) => ({
      name: String(v.name ?? '').trim() || undefined,
      parent_id: v.parent_id ? String(v.parent_id) : null,
    }),
  }

  return <MasterCrudPage config={config} />
}

export default CategoriesPageInner