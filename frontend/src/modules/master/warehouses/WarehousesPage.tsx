import MasterCrudPage, { type MasterCrudConfig } from '../MasterCrudPage'
import {
  warehousesApi,
  type Warehouse,
  type CreateWarehouseInput,
  type UpdateWarehouseInput,
} from '../../../api/warehouses'

const warehouseTypeOptions = [
  { value: 'raw_material', label: 'Raw Material' },
  { value: 'finished_goods', label: 'Finished Goods' },
  { value: 'quarantine', label: 'Quarantine' },
]

const config: MasterCrudConfig<Warehouse, CreateWarehouseInput, UpdateWarehouseInput> = {
  title: 'Warehouses',
  entityLabel: 'Warehouse',
  api: warehousesApi,
  searchPlaceholder: 'Search warehouses by name or code…',

  getLabel: (row: Warehouse) => row.name,

  columns: [
    { key: 'code', label: 'Code' },
    { key: 'name', label: 'Name' },
    {
      key: 'warehouse_type',
      label: 'Type',
      render: (row: Warehouse) =>
        warehouseTypeOptions.find((o) => o.value === row.warehouse_type)?.label ?? row.warehouse_type,
    },
    {
      key: 'location',
      label: 'Location',
      render: (row: Warehouse) => row.location ?? '—',
    },
    {
      key: 'is_active',
      label: 'Status',
      render: (row: Warehouse) => (
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
      placeholder: 'e.g. WH-01',
      createOnly: true,
      span: 1,
    },
    {
      key: 'name',
      label: 'Name',
      type: 'text',
      required: true,
      placeholder: 'e.g. Main Store',
      span: 1,
    },
    {
      key: 'warehouse_type',
      label: 'Type',
      type: 'select',
      required: true,
      options: warehouseTypeOptions,
      placeholder: 'Select type…',
      span: 2,
    },
    {
      key: 'location',
      label: 'Location',
      type: 'text',
      placeholder: 'e.g. Building A, Floor 1',
      hint: 'Physical address or description',
      span: 2,
    },
    {
      key: 'is_active',
      label: 'Active',
      type: 'checkbox',
      editOnly: true,
    },
  ],

  toCreatePayload: (v) => ({
    code: String(v.code ?? '').trim(),
    name: String(v.name ?? '').trim(),
    warehouse_type: String(v.warehouse_type ?? '').trim(),
    location: v.location ? String(v.location).trim() : undefined,
  }),

  toUpdatePayload: (v) => ({
    name: String(v.name ?? '').trim() || undefined,
    warehouse_type: String(v.warehouse_type ?? '').trim() || undefined,
    location: v.location ? String(v.location).trim() : null,
    is_active: typeof v.is_active === 'boolean' ? v.is_active : undefined,
  }),
}

export default function WarehousesPage() {
  return <MasterCrudPage config={config} />
}