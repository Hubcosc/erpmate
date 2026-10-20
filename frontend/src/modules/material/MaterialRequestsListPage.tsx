import { useEffect, useState } from 'react'
import { listMaterialRequisitions, type MaterialRequisition } from '../../api/material'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusLabel, statusTone } from '../../shared/components/StatusBadge'
import Tabs from '../../shared/components/Tabs'
import { fmtDate, fmtQty, unwrapPaged } from '../../shared/utils/format'

type Props = {
  role: string
  onNavigate: (page: string, id?: string) => void
}

const STATUS_TABS = [
  { key: '', label: 'All' },
  { key: 'draft', label: 'Draft' },
  { key: 'pending_pm', label: 'Pending PM' },
  { key: 'approved', label: 'Approved' },
  { key: 'partially_issued', label: 'Partially Issued' },
  { key: 'fully_issued', label: 'Fully Issued' },
  { key: 'rejected', label: 'Rejected' },
]

export default function MaterialRequestsListPage({ role, onNavigate }: Props) {
  const [items, setItems] = useState<MaterialRequisition[]>([])
  const [total, setTotal] = useState(0)
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const canCreate = role === 'department_head' || role === 'admin'

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listMaterialRequisitions(1, 100, status || undefined)
      const { items: list, total: count } = unwrapPaged<MaterialRequisition>(res)
      setItems(list)
      setTotal(count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load material requests')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status])

  const visible = items.filter((mr) => {
    if (!search.trim()) return true
    const q = search.trim().toLowerCase()
    return (
      mr.mr_number?.toLowerCase().includes(q) ||
      mr.department_name?.toLowerCase().includes(q) ||
      mr.requester_name?.toLowerCase().includes(q) ||
      mr.purpose?.toLowerCase().includes(q)
    )
  })

  const columns: Column<MaterialRequisition>[] = [
    { key: 'mr', header: 'MR Number', render: (r) => <strong>{r.mr_number}</strong> },
    { key: 'dept', header: 'Department', render: (r) => r.department_name || '—' },
    { key: 'requester', header: 'Requested By', render: (r) => r.requester_name || '—' },
    { key: 'required', header: 'Required Date', render: (r) => fmtDate(r.required_date) },
    { key: 'items', header: 'Lines', align: 'right', render: (r) => r.items?.length ?? 0 },
    {
      key: 'qty',
      header: 'Total Qty',
      align: 'right',
      render: (r) => fmtQty((r.items ?? []).reduce((sum, i) => sum + (i.quantity ?? 0), 0)),
    },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <StatusBadge label={statusLabel(r.status)} tone={statusTone(r.status)} />,
    },
  ]

  return (
    <>
      <PageHeader
        title={<>Material Requests</>}
        subtitle="Department requests for raw material, and the store issues made against them."
        actions={
          canCreate ? (
            <Button variant="primary" onClick={() => onNavigate('Create MR')}>
              + New Material Request
            </Button>
          ) : undefined
        }
      />

      <Tabs tabs={STATUS_TABS} active={status} onChange={setStatus} />

      <div className="ui-table-toolbar" style={{ marginTop: 16 }}>
        <div className="ui-table-filters">
          <label className="ui-table-search">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
            <input
              type="search"
              placeholder="Search by MR number, department or requester…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
        </div>
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${visible.length} of ${total} request${total === 1 ? '' : 's'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading material requests…</div>
        ) : visible.length === 0 ? (
          <EmptyState
            title="No material requests found"
            description={
              canCreate
                ? 'Create a material request to ask the store for raw material.'
                : 'Nothing has been requested under this filter yet.'
            }
            icon="materials"
          />
        ) : (
          <DataTable
            columns={columns}
            rows={visible}
            rowKey={(r) => r.id}
            onRowClick={(r) => onNavigate('MR Detail', r.id)}
          />
        )}
      </div>
    </>
  )
}
