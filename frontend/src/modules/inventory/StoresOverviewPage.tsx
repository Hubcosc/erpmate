import { useEffect, useState } from 'react'
import { inventoryApi } from '../../api/inventoryClient'
import { listWarehouses, type Warehouse } from '../../api/master'
import type { InventoryAlert, StockBalance, ValuationSummary } from '../../api/inventory'
import Button from '../../shared/components/Button'
import Card from '../../shared/components/Card'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusTone } from '../../shared/components/StatusBadge'
import { fmtDate, fmtMoney, fmtQty, unwrap } from '../../shared/utils/format'

type Props = {
  onNavigate: (page: string, id?: string) => void
}

const WAREHOUSE_TYPE_LABEL: Record<string, string> = {
  raw_material: 'Main Store',
  finished_goods: 'Finished Goods Store',
  quarantine: 'Quarantine',
}

/** Stores Overview — every warehouse, its stock position, valuation and alerts. */
export default function StoresOverviewPage({ onNavigate }: Props) {
  const [warehouses, setWarehouses] = useState<Warehouse[]>([])
  const [stock, setStock] = useState<StockBalance[]>([])
  const [alerts, setAlerts] = useState<InventoryAlert[]>([])
  const [valuation, setValuation] = useState<ValuationSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const [whs, stockRes, alertRes, valRes] = await Promise.all([
        listWarehouses(),
        inventoryApi.listStock({ limit: 200 }),
        inventoryApi.listAlerts({ limit: 50 }).catch(() => null),
        inventoryApi.valuationSummary().catch(() => null),
      ])
      setWarehouses(Array.isArray(whs) ? whs : unwrap<Warehouse[]>(whs))
      setStock(stockRes.items)
      setAlerts(alertRes ? alertRes.items : [])
      setValuation(valRes)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load stores overview')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const byWarehouse = new Map<string, StockBalance[]>()
  for (const line of stock) {
    const list = byWarehouse.get(line.warehouse_id) ?? []
    list.push(line)
    byWarehouse.set(line.warehouse_id, list)
  }

  const columns: Column<Warehouse>[] = [
    { key: 'code', header: 'Code', render: (w) => <strong>{w.code}</strong> },
    { key: 'name', header: 'Warehouse', render: (w) => w.name },
    {
      key: 'type',
      header: 'Type',
      render: (w) => WAREHOUSE_TYPE_LABEL[w.warehouse_type] ?? w.warehouse_type.replace(/_/g, ' '),
    },
    {
      key: 'lines',
      header: 'Stock Lines',
      align: 'right',
      render: (w) => byWarehouse.get(w.id)?.length ?? 0,
    },
    {
      key: 'onhand',
      header: 'On Hand',
      align: 'right',
      render: (w) =>
        fmtQty((byWarehouse.get(w.id) ?? []).reduce((sum, l) => sum + (l.quantity ?? 0), 0)),
    },
    {
      key: 'available',
      header: 'Available',
      align: 'right',
      render: (w) =>
        fmtQty((byWarehouse.get(w.id) ?? []).reduce((sum, l) => sum + (l.available_quantity ?? 0), 0)),
    },
    {
      key: 'alerts',
      header: 'Alerts',
      align: 'right',
      render: (w) => alerts.filter((a) => a.warehouse_id === w.id).length,
    },
    {
      key: 'active',
      header: 'Status',
      render: (w) => (
        <StatusBadge label={w.is_active ? 'Active' : 'Inactive'} tone={w.is_active ? 'positive' : 'neutral'} />
      ),
    },
  ]

  const alertColumns: Column<InventoryAlert>[] = [
    { key: 'type', header: 'Alert', render: (r) => r.alert_type.replace(/_/g, ' ') },
    { key: 'severity', header: 'Severity', render: (r) => <StatusBadge label={r.severity} tone={statusTone(r.severity)} /> },
    { key: 'item', header: 'Item', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name },
    { key: 'batch', header: 'Batch', render: (r) => r.batch_number || '—' },
    { key: 'expiry', header: 'Expiry', render: (r) => fmtDate(r.expiry_date) },
    { key: 'message', header: 'Message', render: (r) => r.message },
  ]

  return (
    <>
      <PageHeader
        title={<>Stores Overview</>}
        subtitle="Every warehouse, its stock position, valuation and open alerts."
        actions={
          <Button onClick={() => onNavigate('Stock Movements')}>View movements</Button>
        }
      />

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      {valuation && (
        <div className="ui-tiles" style={{ gridTemplateColumns: 'repeat(4, minmax(0, 1fr))' }}>
          <div className="ui-tile">
            <span className="ui-tile-label">Inventory value</span>
            <span className="ui-tile-value">{fmtMoney(valuation.total_value)}</span>
            <span className="ui-tile-note">Weighted-average estimate</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: 'var(--color-accent)' } as React.CSSProperties}>
            <span className="ui-tile-label">Valuation lines</span>
            <span className="ui-tile-value">{valuation.total_lines}</span>
            <span className="ui-tile-note">Item × warehouse rows</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: '#B54708' } as React.CSSProperties}>
            <span className="ui-tile-label">Lines without cost</span>
            <span className="ui-tile-value">{valuation.lines_without_cost}</span>
            <span className="ui-tile-note">No recorded cost</span>
          </div>
          <div className="ui-tile" style={{ ['--module-color' as string]: 'var(--color-danger)' } as React.CSSProperties}>
            <span className="ui-tile-label">Open alerts</span>
            <span className="ui-tile-value">{alerts.length}</span>
            <span className="ui-tile-note">Low stock and expiry</span>
          </div>
        </div>
      )}

      <Card title="Warehouses" bordered>
        {loading ? (
          <div className="ui-state">Loading warehouses…</div>
        ) : warehouses.length === 0 ? (
          <EmptyState title="No warehouses" description="Create warehouses in master data." icon="inventory" />
        ) : (
          <DataTable columns={columns} rows={warehouses} rowKey={(w) => w.id} />
        )}
      </Card>

      <Card title="Open Alerts" subtitle="Raised automatically by the background scanner every 15 minutes." bordered>
        {loading ? (
          <div className="ui-state">Loading alerts…</div>
        ) : alerts.length === 0 ? (
          <EmptyState title="No active alerts" description="Low-stock and expiry alerts will appear here." icon="bell" />
        ) : (
          <DataTable columns={alertColumns} rows={alerts} rowKey={(r) => r.id} />
        )}
      </Card>
    </>
  )
}
