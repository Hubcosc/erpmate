import { useEffect, useState } from 'react'
import MasterCrudForm, { type FieldConfig } from './MasterCrudForm'
import './MasterCrudPage.css'
import { Icon } from '../../shared/components/Icon'
import Modal from '../../shared/components/Modal'
import type { CrudApi, ApiEnvelope } from '../../api/master-factory'

export interface ColumnConfig<T> {
  key: keyof T & string
  label: string
  /** Custom cell renderer; falls back to raw value */
  render?: (row: T) => React.ReactNode
  align?: 'left' | 'right' | 'center'
}

export interface MasterCrudConfig<T extends { id: string }, CreateInput, UpdateInput> {
  /** Page title e.g. "Departments" */
  title: string
  /** Subtitle under title */
  subtitle?: string
  /** API built from createMasterApi() */
  api: CrudApi<T, CreateInput, UpdateInput>
  /** Table columns */
  columns: ColumnConfig<T>[]
  /** Form field configs */
  fields: FieldConfig[]
  /** Display name for one item, e.g. "Department" (for messages) */
  entityLabel: string
  /** Extract unique label from a row (for edit modal title) */
  getLabel: (row: T) => string
  /** Transform form values before sending to create */
  toCreatePayload?: (values: Record<string, unknown>) => CreateInput
  /** Transform form values before sending to update */
  toUpdatePayload?: (values: Record<string, unknown>, entity: T) => UpdateInput
  /** Search placeholder */
  searchPlaceholder?: string
}

interface Props<T extends { id: string }, CreateInput, UpdateInput> {
  config: MasterCrudConfig<T, CreateInput, UpdateInput>
}

export default function MasterCrudPage<
  T extends { id: string },
  CreateInput,
  UpdateInput,
>({ config }: Props<T, CreateInput, UpdateInput>) {
  const {
    title,
    subtitle,
    api,
    columns,
    fields,
    entityLabel,
    getLabel,
    toCreatePayload,
    toUpdatePayload,
    searchPlaceholder,
  } = config

  const [items, setItems] = useState<T[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')

  const [showCreate, setShowCreate] = useState(false)
  const [editing, setEditing] = useState<T | null>(null)
  const [deleting, setDeleting] = useState<T | null>(null)
  const [deletingBusy, setDeletingBusy] = useState(false)

  const load = async (searchValue = '') => {
    setLoading(true)
    setError('')
    try {
      const res: ApiEnvelope<T[]> = await api.list({
        search: searchValue,
        page: 1,
        limit: 100,
      })
      setItems(res.data ?? [])
      setTotal(res.meta?.total ?? res.data?.length ?? 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    void load(search)
  }

  const handleCreate = async (values: Record<string, unknown>) => {
    const payload = toCreatePayload ? toCreatePayload(values) : (values as unknown as CreateInput)
    await api.create(payload)
    setShowCreate(false)
    void load(search)
  }

  const handleUpdate = async (values: Record<string, unknown>) => {
    if (!editing) return
    const payload = toUpdatePayload
      ? toUpdatePayload(values, editing)
      : (values as unknown as UpdateInput)
    await api.update(editing.id, payload)
    setEditing(null)
    void load(search)
  }

  const handleDelete = async () => {
    if (!deleting) return
    setDeletingBusy(true)
    try {
      await api.remove(deleting.id)
      setDeleting(null)
      void load(search)
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setDeletingBusy(false)
    }
  }

  const pluralLabel = entityLabel.toLowerCase() + 's'

  return (
    <div className="mcp-page">
      <header className="mcp-header">
        <div>
          <p className="mcp-eyebrow">Master Data</p>
          <h1>{title}</h1>
          <p className="mcp-subtitle">
            {subtitle ?? `${total} ${total === 1 ? entityLabel.toLowerCase() : pluralLabel} in the system`}
          </p>
        </div>
        <button
          className="mcp-btn mcp-btn-primary"
          type="button"
          onClick={() => setShowCreate(true)}
        >
          <Icon name="settings" size={16} /> Add {entityLabel.toLowerCase()}
        </button>
      </header>

      <form className="mcp-toolbar" onSubmit={handleSearch}>
        <label className="mcp-search">
          <Icon name="search" size={16} />
          <input
            type="search"
            placeholder={searchPlaceholder ?? `Search ${title.toLowerCase()}…`}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button className="mcp-btn" type="submit">
          Search
        </button>
        {search && (
          <button
            className="mcp-btn mcp-btn-ghost"
            type="button"
            onClick={() => {
              setSearch('')
              void load('')
            }}
          >
            Clear
          </button>
        )}
      </form>

      {loading && <p className="mcp-state">Loading…</p>}
      {error && <p className="mcp-state mcp-state-error">{error}</p>}
      {!loading && !error && items.length === 0 && (
        <p className="mcp-state">No {pluralLabel} yet.</p>
      )}

      {!loading && !error && items.length > 0 && (
        <div className="mcp-table-wrap">
          <table className="mcp-table">
            <thead>
              <tr>
                {columns.map((c) => (
                  <th key={c.key} style={{ textAlign: c.align ?? 'left' }}>
                    {c.label}
                  </th>
                ))}
                <th aria-label="Actions"></th>
              </tr>
            </thead>
            <tbody>
              {items.map((row: T) => (
                <tr key={row.id}>
                  {columns.map((c) => (
                    <td key={c.key} style={{ textAlign: c.align ?? 'left' }}>
                      {c.render
                        ? c.render(row)
                        : ((row as Record<string, unknown>)[c.key] ?? '—') as React.ReactNode}
                    </td>
                  ))}
                  <td className="mcp-actions-cell">
                    <div className="mcp-row-actions">
                      <button
                        className="mcp-row-btn"
                        type="button"
                        onClick={() => setEditing(row)}
                      >
                        Edit
                      </button>
                      <button
                        className="mcp-row-btn mcp-row-btn-danger"
                        type="button"
                        onClick={() => setDeleting(row)}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create modal */}
      <Modal
        open={showCreate}
        title={`Add ${entityLabel.toLowerCase()}`}
        onClose={() => setShowCreate(false)}
      >
        <MasterCrudForm
          fields={fields}
          entity={null}
          onSubmit={handleCreate}
          onCancel={() => setShowCreate(false)}
        />
      </Modal>

      {/* Edit modal */}
      <Modal
        open={editing !== null}
        title={
          editing
            ? `Edit ${entityLabel.toLowerCase()}: ${getLabel(editing)}`
            : `Edit ${entityLabel.toLowerCase()}`
        }
        onClose={() => setEditing(null)}
      >
        {editing && (
          <MasterCrudForm
            fields={fields}
            entity={editing}
            onSubmit={handleUpdate}
            onCancel={() => setEditing(null)}
          />
        )}
      </Modal>

      {/* Delete modal */}
      <Modal
        open={deleting !== null}
        title={`Delete ${entityLabel.toLowerCase()}`}
        onClose={() => !deletingBusy && setDeleting(null)}
      >
        {deleting && (
          <div className="mcp-delete-confirm">
            <p>
              Are you sure you want to delete <strong>{getLabel(deleting)}</strong>?
            </p>
            <p className="mcp-delete-note">
              This will deactivate the record. It can be restored by an administrator.
            </p>
            <div className="mcf-actions">
              <button
                type="button"
                className="mcf-btn mcf-btn-ghost"
                onClick={() => setDeleting(null)}
                disabled={deletingBusy}
              >
                Cancel
              </button>
              <button
                type="button"
                className="mcf-btn mcf-btn-danger"
                onClick={() => void handleDelete()}
                disabled={deletingBusy}
              >
                {deletingBusy ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}