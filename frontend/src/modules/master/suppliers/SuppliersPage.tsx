import MasterCrudPage, { type MasterCrudConfig } from '../MasterCrudPage'
import {
  suppliersApi,
  type Supplier,
  type CreateSupplierInput,
  type UpdateSupplierInput,
} from '../../../api/suppliers'

const config: MasterCrudConfig<Supplier, CreateSupplierInput, UpdateSupplierInput> = {
  title: 'Suppliers',
  entityLabel: 'Supplier',
  api: suppliersApi,
  searchPlaceholder: 'Search suppliers by name, code, or contact…',

  getLabel: (row: Supplier) => row.name,

  columns: [
    {
      key: 'code',
      label: 'Code',
      render: (row: Supplier) => row.code ?? '—',
    },
    { key: 'name', label: 'Name' },
    {
      key: 'contact_person',
      label: 'Contact',
      render: (row: Supplier) => row.contact_person ?? '—',
    },
    {
      key: 'phone',
      label: 'Phone',
      render: (row: Supplier) => row.phone ?? '—',
    },
    {
      key: 'email',
      label: 'Email',
      render: (row: Supplier) => row.email ?? '—',
    },
    {
      key: 'is_active',
      label: 'Status',
      render: (row: Supplier) => (
        <span className={`mcp-pill ${row.is_active ? 'is-active' : 'is-inactive'}`}>
          {row.is_active ? 'Active' : 'Inactive'}
        </span>
      ),
    },
  ],

  fields: [
    {
      key: 'name',
      label: 'Name',
      type: 'text',
      required: true,
      placeholder: 'e.g. ABC Steel Ltd.',
      span: 2,
    },
    {
      key: 'code',
      label: 'Code',
      type: 'text',
      placeholder: 'e.g. SUP-001',
      hint: 'Optional short code',
      span: 1,
    },
    {
      key: 'tin_number',
      label: 'TIN Number',
      type: 'text',
      placeholder: 'e.g. 0001234567',
      span: 1,
    },
    {
      key: 'contact_person',
      label: 'Contact Person',
      type: 'text',
      placeholder: 'e.g. Abebe Bekele',
      span: 1,
    },
    {
      key: 'phone',
      label: 'Phone',
      type: 'tel',
      placeholder: '+251...',
      span: 1,
    },
    {
      key: 'email',
      label: 'Email',
      type: 'email',
      placeholder: 'name@company.com',
      span: 2,
    },
    {
      key: 'address',
      label: 'Address',
      type: 'textarea',
      placeholder: 'Street, city, country…',
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
    name: String(v.name ?? '').trim(),
    code: v.code ? String(v.code).trim() : undefined,
    contact_person: v.contact_person ? String(v.contact_person).trim() : undefined,
    phone: v.phone ? String(v.phone).trim() : undefined,
    email: v.email ? String(v.email).trim() : undefined,
    address: v.address ? String(v.address).trim() : undefined,
    tin_number: v.tin_number ? String(v.tin_number).trim() : undefined,
  }),

  toUpdatePayload: (v) => ({
    name: String(v.name ?? '').trim() || undefined,
    code: v.code ? String(v.code).trim() : null,
    contact_person: v.contact_person ? String(v.contact_person).trim() : null,
    phone: v.phone ? String(v.phone).trim() : null,
    email: v.email ? String(v.email).trim() : null,
    address: v.address ? String(v.address).trim() : null,
    tin_number: v.tin_number ? String(v.tin_number).trim() : null,
    is_active: typeof v.is_active === 'boolean' ? v.is_active : undefined,
  }),
}

export default function SuppliersPage() {
  return <MasterCrudPage config={config} />
}