import { useEffect, useState } from 'react'
import { listProductionReceipts, type ProductionReceipt } from '../../api/production'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge from '../../shared/components/StatusBadge'
import { fmtDate, fmtQty, unwrapPaged } from '../../shared/utils/format'

type Props = {
  role: string
  onNavigate: (page: string, id?: string) => void
}

export default function ProductionReceiptsListPage({ role, onNavigate }: Props) {
  const [items, setItems] = useState<ProductionReceipt[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const canCreate = role === 'production_store_keeper' || role === 'admin'

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listProductionReceipts(1, 100)
      const { items: list, total: count } = unwrapPaged<ProductionReceipt>(res)
      setItems(list)
      setTotal(count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load production receipts')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const columns: Column<ProductionReceipt>[] = [
    { key: 'no', header: 'Receipt Number', render: (r) => <strong>{r.receipt_number}</strong> },
    { key: 'wh', header: 'Warehouse', render: (r) => r.warehouse_name || '—' },
    { key: 'by', header: 'Recorded By', render: (r) => r.recorded_by_name || '—' },
    { key: 'date', header: 'Receipt Date', render: (r) => fmtDate(r.receipt_date) },
    { key: 'lines', header: 'Lines', align: 'right', render: (r) => r.items?.length ?? 0 },
    {
      key: 'qty',
      header: 'Total Produced',
      align: 'right',
      render: (r) => fmtQty((r.items ?? []).reduce((sum, i) => sum + (i.quantity_produced ?? 0), 0)),
    },
    {
      key: 'quality',
      header: 'Quality',
      render: (r) => {
        const rejected = (r.items ?? []).filter((i) => i.quality_status === 'rejected').length
        if (rejected === 0) return <StatusBadge label="Accepted" tone="positive" />
        if (rejected === (r.items ?? []).length) return <StatusBadge label="Rejected" tone="negative" />
        return <StatusBadge label={`${rejected} rejected`} tone="warning" />
      },
    },
  ]

  return (
    <>
      <PageHeader
        title={<>Production Receipts</>}
        subtitle="Finished goods received from production, with batch, expiry and quality status."
        actions={
          canCreate ? (
            <Button variant="primary" onClick={() => onNavigate('Create Production Receipt')}>
              + New Production Receipt
            </Button>
          ) : undefined
        }
      />

      <div className="ui-table-toolbar">
        <div className="ui-table-filters" />
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${items.length} of ${total} receipt${total === 1 ? '' : 's'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading production receipts…</div>
        ) : items.length === 0 ? (
          <EmptyState
            title="No production receipts yet"
            description={
              canCreate
                ? 'Record finished goods to add them to the finished-goods store.'
                : 'Nothing has been received from production yet.'
            }
            icon="production"
          />
        ) : (
          <DataTable
            columns={columns}
            rows={items}
            rowKey={(r) => r.id}
            onRowClick={(r) => onNavigate('Production Receipt Detail', r.id)}
          />
        )}
      </div>
    </>
  )
}
