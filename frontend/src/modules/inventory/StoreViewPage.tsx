import { useEffect, useState } from 'react'
import { inventoryApi } from '../../api/inventoryClient'
import type { InventoryAlert, StockBalance, StockSummary } from '../../api/inventory'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusTone } from '../../shared/components/StatusBadge'
import Tabs from '../../shared/components/Tabs'
import { fmtDate, fmtQty } from '../../shared/utils/format'

type Props = {
  warehouseType: string
  itemType?: string
  title: string
  description: string
  moduleColor?: string
  onNavigate?: (page: string, id?: string) => void
}

const STATUS_LABELS: Record<string, string> = {
  in_stock: 'In stock',
  low_stock: 'Low stock',
  out_of_stock: 'Out of stock',
}

/** A single store view: stock lines and alerts for one warehouse type. */
export default function StoreViewPage({
  warehouseType,
  itemType,
  title,
  description,
  moduleColor,
  onNavigate,
}: Props) {
  const [stock, setStock] = useState<StockBalance[]>([])
  const [alerts, setAlerts] = useState<InventoryAlert[]>([])
  const [summary, setSummary] = useState<StockSummary | null>(null)
  const [tab, setTab] = useState('stock')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const [stockRes, summaryRes, alertRes] = await Promise.all([
        inventoryApi.listStock({
          warehouse_type: warehouseType,
          item_type: itemType,
          search: search || undefined,
          limit: 100,
        }),
        inventoryApi.stockSummary({ warehouse_type: warehouseType, item_type: itemType }),
        inventoryApi.listAlerts({ limit: 50 }).catch(() => null),
      ])
      setStock(stockRes.items)
      setSummary(summaryRes)
      setAlerts(alertRes ? alertRes.items : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to load ${title.toLowerCase()}`)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [warehouseType, itemType, tab])

  const columns: Column<StockBalance>[] = [
    { key: 'code', header: 'Item Code', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'name', header: 'Item', render: (r) => r.item_name },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name },
    { key: 'uom', header: 'Unit', render: (r) => r.unit_of_measure },
    { key: 'qty', header: 'On Hand', align: 'right', render: (r) => fmtQty(r.quantity) },
    { key: 'res', header: 'Reserved', align: 'right', render: (r) => fmtQty(r.reserved_quantity) },
    { key: 'avail', header: 'Available', align: 'right', render: (r) => fmtQty(r.available_quantity) },
    { key: 'reorder', header: 'Reorder Level', align: 'right', render: (r) => fmtQty(r.reorder_level ?? r.min_stock_level) },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <StatusBadge label={STATUS_LABELS[r.stock_status] ?? r.stock_status} tone={statusTone(r.stock_status)} />,
    },
  ]

  const alertColumns: Column<InventoryAlert>[] = [
    { key: 'type', header: 'Alert', render: (r) => r.alert_type.replace(/_/g, ' ') },
    { key: 'severity', header: 'Severity', render: (r) => <StatusBadge label={r.severity} tone={statusTone(r.severity)} /> },
    { key: 'item', header: 'Item', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'name', header: 'Description', render: (r) => r.item_name },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name },
    { key: 'batch', header: 'Batch', render: (r) => r.batch_number || '—' },
    { key: 'expiry', header: 'Expiry', render: (r) => fmtDate(r.expiry_date) },
    { key: 'message', header: 'Message', render: (r) => r.message },
  ]

  return (
    <>
      <PageHeader
        title={title}
        subtitle={description}
        actions={
          onNavigate ? (
            <Button onClick={() => onNavigate('Stock Movements')}>View movements</Button>
          ) : undefined
        }
      />

      {summary && (
        <div className="ui-tiles" style={{ gridTemplateColumns: 'repeat(4, minmax(0, 1fr))' }}>
          <div className="ui-tile" style={moduleColor ? ({ ['--module-color' as string]: moduleColor } as React.CSSProperties) : undefined}>
            <span className="ui-tile-label">Stock lines</span>
            <span className="ui-tile-value">{summary.total_lines}</span>
            <span className="ui-tile-note">{summary.total_items} distinct items</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: 'var(--color-success)' } as React.CSSProperties}>
            <span className="ui-tile-label">In stock</span>
            <span className="ui-tile-value">{summary.in_stock}</span>
            <span className="ui-tile-note">Lines above the reorder level</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: '#B54708' } as React.CSSProperties}>
            <span className="ui-tile-label">Low stock</span>
            <span className="ui-tile-value">{summary.low_stock}</span>
            <span className="ui-tile-note">Needs replenishment</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: 'var(--color-danger)' } as React.CSSProperties}>
            <span className="ui-tile-label">Out of stock</span>
            <span className="ui-tile-value">{summary.out_of_stock}</span>
            <span className="ui-tile-note">Nothing on hand</span>
          </div>
        </div>
      )}

      <Tabs
        tabs={[
          { key: 'stock', label: 'Stock' },
          { key: 'alerts', label: 'Alerts', count: alerts.length },
        ]}
        active={tab}
        onChange={setTab}
      />

      <div className="ui-table-toolbar" style={{ marginTop: 16 }}>
        <div className="ui-table-filters">
          {tab === 'stock' && (
            <label className="ui-table-search">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                <circle cx="11" cy="11" r="7" />
                <path d="m20 20-3.5-3.5" />
              </svg>
              <input
                type="search"
                placeholder="Search by item code or name…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </label>
          )}
        </div>
        <div className="ui-table-filters">
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading…</div>
        ) : tab === 'stock' ? (
          stock.length === 0 ? (
            <EmptyState title="No stock in this store" description="Stock lines will appear here once items are received." icon="inventory" />
          ) : (
            <DataTable columns={columns} rows={stock} rowKey={(r) => `${r.item_id}-${r.warehouse_id}`} />
          )
        ) : alerts.length === 0 ? (
          <EmptyState title="No active alerts" description="Low-stock and expiry alerts will appear here." icon="bell" />
        ) : (
          <DataTable columns={alertColumns} rows={alerts} rowKey={(r) => r.id} />
        )}
      </div>
    </>
  )
}
