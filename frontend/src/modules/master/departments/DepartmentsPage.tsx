import MasterCrudPage, { type MasterCrudConfig } from '../MasterCrudPage'
import {
  departmentsApi,
  type Department,
  type CreateDepartmentInput,
  type UpdateDepartmentInput,
} from '../../../api/departments'

const config: MasterCrudConfig<Department, CreateDepartmentInput, UpdateDepartmentInput> = {
  title: 'Departments',
  entityLabel: 'Department',
  api: departmentsApi,
  searchPlaceholder: 'Search departments by name or code…',

  getLabel: (row: Department) => row.name,

  columns: [
    { key: 'name', label: 'Name' },
    {
      key: 'code',
      label: 'Code',
      render: (row: Department) => row.code ?? '—',
    },
    {
      key: 'is_active',
      label: 'Status',
      render: (row: Department) => (
        <span className={`mcp-pill ${row.is_active ? 'is-active' : 'is-inactive'}`}>
          {row.is_active ? 'Active' : 'Inactive'}
        </span>
      ),
    },
    {
      key: 'created_at',
      label: 'Created',
      render: (row: Department) => new Date(row.created_at).toLocaleDateString(),
    },
  ],

  fields: [
    {
      key: 'name',
      label: 'Name',
      type: 'text',
      required: true,
      placeholder: 'e.g. Finance',
      span: 2,
    },
    {
      key: 'code',
      label: 'Code',
      type: 'text',
      placeholder: 'e.g. FIN',
      hint: 'Short code for reports and documents (optional)',
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
  }),

  toUpdatePayload: (v) => ({
    name: String(v.name ?? '').trim() || undefined,
    code: v.code ? String(v.code).trim() : null,
    is_active: typeof v.is_active === 'boolean' ? v.is_active : undefined,
  }),
}

export default function DepartmentsPage() {
  return <MasterCrudPage config={config} />
}