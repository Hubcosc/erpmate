import { useEffect, useState } from 'react'
import { inventoryApi } from '../../api/inventoryClient'
import type { StockMovement } from '../../api/inventory'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import { fmtDate, fmtMoney, fmtQty } from '../../shared/utils/format'

type Props = {
  onNavigate?: (page: string, id?: string) => void
  title?: string
  subtitle?: string
}

/** Every receipt, issue and movement — the system's audit trail for stock. */
export default function StockMovementsPage({
  title = 'Stock Movements',
  subtitle = 'Every receipt, issue and movement recorded against stock.',
}: Props) {
  const [movements, setMovements] = useState<StockMovement[]>([])
  const [types, setTypes] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [movementType, setMovementType] = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo, setDateTo] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await inventoryApi.listMovements({
        search: search || undefined,
        movement_type: movementType ? [movementType] : undefined,
        date_from: dateFrom || undefined,
        date_to: dateTo || undefined,
        limit: 100,
      })
      setMovements(res.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load stock movements')
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
  }, [movementType, dateFrom, dateTo])

  const columns: Column<StockMovement>[] = [
    { key: 'date', header: 'Date', render: (r) => fmtDate(r.movement_date) },
    { key: 'type', header: 'Type', render: (r) => r.movement_type.replace(/_/g, ' ') },
    { key: 'item', header: 'Item', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'name', header: 'Description', render: (r) => r.item_name },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name },
    { key: 'batch', header: 'Batch', render: (r) => r.batch_number || '—' },
    { key: 'qty', header: 'Quantity', align: 'right', render: (r) => fmtQty(r.quantity) },
    {
      key: 'value',
      header: 'Value',
      align: 'right',
      render: (r) => (r.total_value === null ? '—' : fmtMoney(r.total_value)),
    },
    { key: 'by', header: 'Performed By', render: (r) => r.performed_by_name || '—' },
  ]

  return (
    <>
      <PageHeader title={title} subtitle={subtitle} />

      <div className="ui-table-toolbar">
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
          <select className="ui-select" value={movementType} onChange={(e) => setMovementType(e.target.value)}>
            <option value="">All movement types</option>
            {types.map((t) => (
              <option key={t} value={t}>
                {t.replace(/_/g, ' ')}
              </option>
            ))}
          </select>
          <input
            className="ui-select"
            type="date"
            value={dateFrom}
            onChange={(e) => setDateFrom(e.target.value)}
            aria-label="From date"
          />
          <input
            className="ui-select"
            type="date"
            value={dateTo}
            onChange={(e) => setDateTo(e.target.value)}
            aria-label="To date"
          />
        </div>
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${movements.length} movement${movements.length === 1 ? '' : 's'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading stock movements…</div>
        ) : movements.length === 0 ? (
          <EmptyState title="No stock movements" description="No receipts, issues or transfers match these filters." icon="inventory" />
        ) : (
          <DataTable columns={columns} rows={movements} rowKey={(r) => r.id} />
        )}
      </div>
    </>
  )
}
