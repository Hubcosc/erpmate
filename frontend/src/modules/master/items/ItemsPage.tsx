import { useEffect, useState } from 'react'
import MasterCrudPage, { type MasterCrudConfig } from '../MasterCrudPage'
import { categoriesApi, type Category } from '../../../api/categories'
import {
  itemsApi,
  getMasterLookups,
  type Item,
  type CreateItemInput,
  type UpdateItemInput,
} from '../../../api/items'

// Friendly labels for backend enum values
const itemTypeLabels: Record<string, string> = {
  raw_material: 'Raw Material',
  finished_good: 'Finished Good',
  consumable: 'Consumable',
  spare_part: 'Spare Part',
}

export default function ItemsPage() {
  const [categories, setCategories] = useState<Category[]>([])
  const [itemTypes, setItemTypes] = useState<string[]>([
    'raw_material',
    'finished_good',
    'consumable',
    'spare_part',
  ])

  // Load categories for the dropdown
  useEffect(() => {
    void categoriesApi
      .list({ page: 1, limit: 100 })
      .then((res) => setCategories(res.data ?? []))
      .catch(() => setCategories([]))
  }, [])

  // Load item type lookups from backend (single source of truth)
  useEffect(() => {
    void getMasterLookups()
      .then((res) => {
        if (res.item_types?.length) setItemTypes(res.item_types)
      })
      .catch(() => {
        // fall back to defaults
      })
  }, [])

  // Map for showing category name in the table
  const categoryNameById: Record<string, string> = {}
  categories.forEach((c) => {
    categoryNameById[c.id] = c.name
  })

  const config: MasterCrudConfig<Item, CreateItemInput, UpdateItemInput> = {
    title: 'Items',
    entityLabel: 'Item',
    api: itemsApi,
    searchPlaceholder: 'Search items by name or code…',

    getLabel: (row: Item) => row.name,

    columns: [
      { key: 'code', label: 'Code' },
      { key: 'name', label: 'Name' },
      {
        key: 'item_type',
        label: 'Type',
        render: (row: Item) => itemTypeLabels[row.item_type] ?? row.item_type,
      },
      {
        key: 'category_id',
        label: 'Category',
        render: (row: Item) =>
          row.category_name ?? (row.category_id ? categoryNameById[row.category_id] ?? '—' : '—'),
      },
      {
        key: 'unit_of_measure',
        label: 'UOM',
        render: (row: Item) => row.unit_of_measure,
      },
      {
        key: 'min_stock_level',
        label: 'Min Stock',
        align: 'right',
        render: (row: Item) => String(row.min_stock_level),
      },
      {
        key: 'is_active',
        label: 'Status',
        render: (row: Item) => (
          <span className={`mcp-pill ${row.is_active ? 'is-active' : 'is-inactive'}`}>
            {row.is_active ? 'Active' : 'Inactive'}
          </span>
        ),
      },
    ],

    fields: [
      {
        key: 'code',
        label: 'Code',
        type: 'text',
        required: true,
        placeholder: 'e.g. RM-001',
        createOnly: true,
        span: 1,
      },
      {
        key: 'name',
        label: 'Name',
        type: 'text',
        required: true,
        placeholder: 'e.g. Steel Sheet 2mm',
        span: 1,
      },
      {
        key: 'item_type',
        label: 'Item Type',
        type: 'select',
        required: true,
        placeholder: 'Select type…',
        options: itemTypes.map((t) => ({ value: t, label: itemTypeLabels[t] ?? t })),
        span: 1,
      },
      {
        key: 'unit_of_measure',
        label: 'Unit of Measure',
        type: 'text',
        required: true,
        placeholder: 'e.g. kg, pcs, liters',
        span: 1,
      },
      {
        key: 'category_id',
        label: 'Category',
        type: 'select',
        placeholder: 'No category',
        options: [
          { value: '', label: 'No category' },
          ...categories.map((c) => ({ value: c.id, label: c.name })),
        ],
        span: 2,
      },
      {
        key: 'description',
        label: 'Description',
        type: 'textarea',
        placeholder: 'Short description (optional)',
        span: 2,
      },
      {
        key: 'min_stock_level',
        label: 'Min Stock Level',
        type: 'number',
        required: true,
        placeholder: '0',
        span: 1,
      },
      {
        key: 'max_stock_level',
        label: 'Max Stock Level',
        type: 'number',
        placeholder: 'Optional',
        span: 1,
      },
      {
        key: 'reorder_level',
        label: 'Reorder Level',
        type: 'number',
        placeholder: 'Optional',
        span: 2,
      },
      {
        key: 'is_batch_tracked',
        label: 'Track batches (lot numbers)',
        type: 'checkbox',
        span: 2,
      },
      {
        key: 'is_expiry_tracked',
        label: 'Track expiry dates',
        type: 'checkbox',
        span: 2,
      },
      {
        key: 'is_active',
        label: 'Active',
        type: 'checkbox',
        editOnly: true,
        span: 2,
      },
    ],

    toCreatePayload: (v) => ({
      code: String(v.code ?? '').trim(),
      name: String(v.name ?? '').trim(),
      item_type: String(v.item_type ?? '').trim(),
      unit_of_measure: String(v.unit_of_measure ?? '').trim(),
      min_stock_level: Number(v.min_stock_level ?? 0),
      description: v.description ? String(v.description).trim() : undefined,
      category_id: v.category_id ? String(v.category_id) : undefined,
      max_stock_level: v.max_stock_level !== '' && v.max_stock_level != null
        ? Number(v.max_stock_level)
        : undefined,
      reorder_level: v.reorder_level !== '' && v.reorder_level != null
        ? Number(v.reorder_level)
        : undefined,
      is_batch_tracked: Boolean(v.is_batch_tracked),
      is_expiry_tracked: Boolean(v.is_expiry_tracked),
    }),

    toUpdatePayload: (v) => ({
      name: String(v.name ?? '').trim() || undefined,
      unit_of_measure: String(v.unit_of_measure ?? '').trim() || undefined,
      min_stock_level:
        v.min_stock_level !== '' && v.min_stock_level != null
          ? Number(v.min_stock_level)
          : undefined,
      description: v.description ? String(v.description).trim() : null,
      category_id: v.category_id ? String(v.category_id) : null,
      max_stock_level:
        v.max_stock_level !== '' && v.max_stock_level != null
          ? Number(v.max_stock_level)
          : null,
      reorder_level:
        v.reorder_level !== '' && v.reorder_level != null
          ? Number(v.reorder_level)
          : null,
      is_batch_tracked: typeof v.is_batch_tracked === 'boolean' ? v.is_batch_tracked : undefined,
      is_expiry_tracked: typeof v.is_expiry_tracked === 'boolean' ? v.is_expiry_tracked : undefined,
      is_active: typeof v.is_active === 'boolean' ? v.is_active : undefined,
    }),
  }

  return <MasterCrudPage config={config} />
}