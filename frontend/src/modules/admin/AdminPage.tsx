import { useCallback, useEffect, useState } from 'react'
import {
  deleteAdminRecord,
  getAdminLookups,
  getAdminModels,
  getAdminOverview,
  listAdminRecords,
  type AdminCount,
  type AdminLookups,
  type AdminModel,
  type AdminModelsResponse,
  type AdminRow,
} from '../../api/admin'
import { Icon, type IconName } from '../../shared/components/Icon'
import Button from '../../shared/components/Button'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import AdminRecordForm from './AdminRecordForm'
import './AdminPage.css'

type View = 'overview' | 'list' | 'form'

const PAGE_SIZE = 25

/** The icon names the Icon component knows about. */
const ICON_NAMES: IconName[] = [
  'overview', 'purchase', 'materials', 'inventory', 'production',
  'sales', 'reports', 'settings', 'search', 'bell', 'menu',
  'arrow', 'clock', 'spark', 'chevron', 'close', 'categories',
]

/** The registry sends an icon name; fall back to a generic one if it is unknown. */
function safeIcon(name: string | undefined): IconName {
  return name && ICON_NAMES.includes(name as IconName) ? (name as IconName) : 'overview'
}

export default function AdminPage() {
  const [models, setModels] = useState<AdminModel[]>([])
  const [groups, setGroups] = useState<string[]>([])
  const [lookups, setLookups] = useState<AdminLookups | null>(null)
  const [counts, setCounts] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const [view, setView] = useState<View>('overview')
  const [activeKey, setActiveKey] = useState<string>('')
  const [recordId, setRecordId] = useState<string>('')

  const [rows, setRows] = useState<AdminRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [listError, setListError] = useState('')
  const [listLoading, setListLoading] = useState(false)

  const active = models.find((m) => m.key === activeKey) ?? null

  // ---- load the registry ------------------------------------------------
  useEffect(() => {
    void (async () => {
      setLoading(true)
      setError('')
      try {
        const [modelsRes, lookupsRes, overviewRes] = await Promise.all([
          getAdminModels(),
          getAdminLookups(),
          getAdminOverview(),
        ])
        const registry = modelsRes.data as AdminModelsResponse
        setModels(registry.models ?? [])
        setGroups(registry.groups ?? [])
        setLookups(lookupsRes.data)
        const list = (overviewRes.data ?? []) as AdminCount[]
        const map: Record<string, number> = {}
        for (const item of list) map[item.key] = item.count
        setCounts(map)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Could not load the administration area')
      } finally {
        setLoading(false)
      }
    })()
  }, [])

  // ---- load the records of the selected model ----------------------------
  const loadRows = useCallback(async () => {
    if (!activeKey) return
    setListLoading(true)
    setListError('')
    try {
      const res = await listAdminRecords(activeKey, { page, limit: PAGE_SIZE, search: search || undefined })
      setRows(res.data ?? [])
      setTotal(res.meta?.total ?? (res.data ?? []).length)
    } catch (err) {
      setListError(err instanceof Error ? err.message : 'Could not load the records')
    } finally {
      setListLoading(false)
    }
  }, [activeKey, page, search])

  useEffect(() => {
    if (view === 'list') void loadRows()
  }, [view, loadRows])

  const selectModel = (key: string) => {
    setActiveKey(key)
    setRecordId('')
    setPage(1)
    setSearch('')
    setView('list')
  }

  const refreshCounts = useCallback(async () => {
    try {
      const overviewRes = await getAdminOverview()
      const list = (overviewRes.data ?? []) as AdminCount[]
      const map: Record<string, number> = {}
      for (const item of list) map[item.key] = item.count
      setCounts(map)
    } catch {
      /* the counts are a convenience, ignore a failure here */
    }
  }, [])

  const handleDelete = async (id: string) => {
    if (!active) return
    if (!window.confirm(`Delete this ${active.label.toLowerCase()}?`)) return
    try {
      await deleteAdminRecord(active.key, id)
      await Promise.all([loadRows(), refreshCounts()])
    } catch (err) {
      setListError(err instanceof Error ? err.message : 'Could not delete the record')
    }
  }

  if (loading) return <div className="ui-state">Loading the administration area…</div>
  if (error) return <ErrorAlert message={error} />

  return (
    <div className="admin-shell">
      <nav className="admin-registry" aria-label="Model registry">
        <p className="admin-registry-head">Super admin</p>
        <button
          type="button"
          className={`admin-registry-link ${view === 'overview' ? 'is-active' : ''}`}
          onClick={() => {
            setView('overview')
            setActiveKey('')
          }}
        >
          <Icon name="overview" size={16} />
          <span>Overview</span>
        </button>

        {groups.map((group) => (
          <div className="admin-registry-group" key={group}>
            <span>{group}</span>
            {models
              .filter((m) => m.group === group)
              .map((model) => (
                <button
                  key={model.key}
                  type="button"
                  className={`admin-registry-link ${activeKey === model.key ? 'is-active' : ''}`}
                  onClick={() => selectModel(model.key)}
                >
                  <span>{model.label_plural}</span>
                  <span className="count">{counts[model.key] ?? 0}</span>
                </button>
              ))}
          </div>
        ))}
      </nav>

      <div className="admin-content">
        {view === 'overview' && (
          <>
            <div className="admin-bar">
              <div>
                <h1 className="admin-bar-title">
                  <strong>Super admin</strong>
                </h1>
                <p className="admin-bar-caption">
                  Every model in the system. Master data can be created and changed here; workflow documents are
                  read-only so the approval trail stays intact.
                </p>
              </div>
            </div>

            <div className="admin-overview-grid">
              {models.map((model) => (
                <button key={model.key} type="button" className="admin-overview-card" onClick={() => selectModel(model.key)}>
                  <span>
                    <Icon name={safeIcon(model.icon)} size={18} />
                  </span>
                  <span>
                    <strong>{counts[model.key] ?? 0}</strong>
                    <small>{model.label_plural}</small>
                  </span>
                </button>
              ))}
            </div>
          </>
        )}

        {view === 'list' && active && (
          <>
            <div className="admin-bar">
              <div>
                <h1 className="admin-bar-title">
                  <strong>{active.label_plural}</strong>
                </h1>
                <p className="admin-bar-caption">{active.description}</p>
              </div>
              <div className="admin-bar-actions">
                <label className="admin-search">
                  <Icon name="search" size={15} />
                  <input
                    type="search"
                    placeholder="Search…"
                    value={search}
                    onChange={(e) => {
                      setSearch(e.target.value)
                      setPage(1)
                    }}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') void loadRows()
                    }}
                  />
                </label>
                <Button size="sm" onClick={() => void loadRows()}>
                  Search
                </Button>
                {active.editable && (
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={() => {
                      setRecordId('')
                      setView('form')
                    }}
                  >
                    + Add {active.label.toLowerCase()}
                  </Button>
                )}
              </div>
            </div>

            {listError && <ErrorAlert message={listError} onRetry={() => void loadRows()} />}

            <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
              {listLoading ? (
                <div className="ui-state">Loading…</div>
              ) : rows.length === 0 ? (
                <EmptyState
                  title={`No ${active.label_plural.toLowerCase()} found`}
                  description={search ? 'Nothing matches this search.' : 'Nothing has been recorded yet.'}
                  icon="overview"
                />
              ) : (
                <>
                  <div className="ui-table-wrap">
                    <table className="ui-table">
                      <thead>
                        <tr>
                          {active.columns.map((column) => (
                            <th key={column.key} className={column.align === 'right' ? 'num' : undefined}>
                              {column.label}
                            </th>
                          ))}
                          <th style={{ textAlign: 'right' }}>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {rows.map((row) => (
                          <tr key={String(row.id)}>
                            {active.columns.map((column) => (
                              <td key={column.key} className={column.align === 'right' ? 'num' : undefined}>
                                {renderCell(row[column.key], column.type)}
                              </td>
                            ))}
                            <td>
                              <div className="admin-row-actions">
                                <button
                                  type="button"
                                  className="admin-link"
                                  onClick={() => {
                                    setRecordId(String(row.id))
                                    setView('form')
                                  }}
                                >
                                  {active.editable ? 'Change' : 'View'}
                                </button>
                                {active.editable && (
                                  <button type="button" className="admin-link is-danger" onClick={() => void handleDelete(String(row.id))}>
                                    Delete
                                  </button>
                                )}
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  <div className="ui-pager">
                    <span>
                      {total === 0 ? 'No records' : `${(page - 1) * PAGE_SIZE + 1}–${Math.min(page * PAGE_SIZE, total)} of ${total}`}
                    </span>
                    <div className="ui-pager-group">
                      <button type="button" onClick={() => setPage(1)} disabled={page <= 1}>
                        First
                      </button>
                      <button type="button" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
                        Previous
                      </button>
                      <button type="button" onClick={() => setPage((p) => p + 1)} disabled={page * PAGE_SIZE >= total}>
                        Next
                      </button>
                      <button
                        type="button"
                        onClick={() => setPage(Math.max(1, Math.ceil(total / PAGE_SIZE)))}
                        disabled={page * PAGE_SIZE >= total}
                      >
                        Last
                      </button>
                    </div>
                  </div>
                </>
              )}
            </div>
          </>
        )}

        {view === 'form' && active && (
          <AdminRecordForm
            model={active}
            lookups={lookups}
            recordId={recordId || null}
            onCancel={() => {
              setRecordId('')
              setView('list')
            }}
            onSaved={() => {
              setRecordId('')
              setView('list')
              void Promise.all([loadRows(), refreshCounts()])
            }}
          />
        )}
      </div>
    </div>
  )
}

/** Renders one cell according to the column type declared in the registry. */
function renderCell(value: unknown, type?: string) {
  if (value === null || value === undefined || value === '') return '—'
  if (type === 'boolean') {
    return value ? 'Yes' : 'No'
  }
  if (type === 'number') {
    const number = Number(value)
    return Number.isFinite(number) ? new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 }).format(number) : String(value)
  }
  if (type === 'date' || type === 'datetime') {
    const date = new Date(String(value))
    if (Number.isNaN(date.getTime())) return String(value)
    return type === 'date'
      ? date.toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })
      : date.toLocaleString(undefined, { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
  }
  if (type === 'badge') {
    return String(value).replace(/_/g, ' ')
  }
  return String(value)
}
