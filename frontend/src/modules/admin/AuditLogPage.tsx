import { useEffect, useState } from 'react'
import { inventoryApi } from '../../api/inventoryClient'
import type { StockMovement } from '../../api/inventory'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert, InfoAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import Tabs from '../../shared/components/Tabs'
import { fmtDate, fmtMoney, fmtQty } from '../../shared/utils/format'

type Props = {
  onNavigate?: (page: string, id?: string) => void
}

/** Audit Log — the system's traceable history.
 *  Stock movements are the global ledger: every receipt, issue and movement is
 *  recorded and can never be removed. Workflow approvals are tracked per
 *  document on the requisition detail pages. */
export default function AuditLogPage({ onNavigate }: Props) {
  const [movements, setMovements] = useState<StockMovement[]>([])
  const [types, setTypes] = useState<string[]>([])
  const [tab, setTab] = useState('all')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await inventoryApi.listMovements({
        search: search || undefined,
        movement_type: tab !== 'all' && tab !== 'receipt' && tab !== 'issue' ? [tab] : undefined,
        limit: 200,
      })
      setMovements(res.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load the audit log')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void (async () => {
      try {
        setTypes(await inventoryApi.movementTypes())
      } catch {
        setTypes([])
      }
    })()
  }, [])

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  const filtered = movements.filter((m) => {
    if (tab === 'all') return true
    if (tab === 'receipt') return m.movement_type.includes('receipt') || m.movement_type.includes('inbound')
    if (tab === 'issue') return m.movement_type.includes('issue') || m.movement_type.includes('outbound')
    return true
  })

  const columns: Column<StockMovement>[] = [
    { key: 'date', header: 'Date', render: (r) => fmtDate(r.movement_date) },
    { key: 'type', header: 'Movement', render: (r) => r.movement_type.replace(/_/g, ' ') },
    { key: 'ref', header: 'Reference', render: (r) => r.reference_type?.replace(/_/g, ' ') || '—' },
    { key: 'item', header: 'Item', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'name', header: 'Description', render: (r) => r.item_name },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name },
    { key: 'batch', header: 'Batch', render: (r) => r.batch_number || '—' },
    { key: 'qty', header: 'Quantity', align: 'right', render: (r) => fmtQty(r.quantity) },
    { key: 'value', header: 'Value', align: 'right', render: (r) => (r.total_value === null ? '—' : fmtMoney(r.total_value)) },
    { key: 'by', header: 'Performed By', render: (r) => r.performed_by_name || '—' },
  ]

  const tabs = [
    { key: 'all', label: 'All movements' },
    { key: 'receipt', label: 'Receipts' },
    { key: 'issue', label: 'Issues' },
    ...types
      .filter((t) => !t.includes('receipt') && !t.includes('issue') && !t.includes('inbound') && !t.includes('outbound'))
      .map((t) => ({ key: t, label: t.replace(/_/g, ' ') })),
  ]

  return (
    <>
      <PageHeader
        title={<>Audit Log</>}
        subtitle="The traceable history of every stock movement in the system."
        actions={
          onNavigate ? <Button onClick={() => onNavigate('Stock Movements')}>Stock ledger</Button> : undefined
        }
      />

      <InfoAlert>
        Every receipt, issue and movement is recorded with its reference, value and the user who performed it.
        Stock movements are never deleted. Approval history for requisitions is kept on each document.
      </InfoAlert>

      <Tabs tabs={tabs} active={tab} onChange={setTab} />

      <div className="ui-table-toolbar" style={{ marginTop: 16 }}>
        <div className="ui-table-filters">
          <label className="ui-table-search">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
            <input
              type="search"
              placeholder="Search by item, batch or reference…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
        </div>
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${filtered.length} entr${filtered.length === 1 ? 'y' : 'ies'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading audit log…</div>
        ) : filtered.length === 0 ? (
          <EmptyState title="Nothing recorded yet" description="Movements will appear here as documents are posted." icon="clock" />
        ) : (
          <DataTable columns={columns} rows={filtered} rowKey={(r) => r.id} />
        )}
      </div>
    </>
  )
}
