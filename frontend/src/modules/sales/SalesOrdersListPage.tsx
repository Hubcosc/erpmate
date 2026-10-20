import { useEffect, useState } from 'react'
import { listSalesOrders, type SalesOrder } from '../../api/sales'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusLabel, statusTone } from '../../shared/components/StatusBadge'
import Tabs from '../../shared/components/Tabs'
import { fmtDate, fmtMoney, unwrapPaged } from '../../shared/utils/format'

type Props = {
  role: string
  onNavigate: (page: string, id?: string) => void
}

const STATUS_TABS = [
  { key: '', label: 'All' },
  { key: 'confirmed', label: 'Confirmed' },
  { key: 'partially_delivered', label: 'Partially Delivered' },
  { key: 'delivered', label: 'Delivered' },
]

export default function SalesOrdersListPage({ role, onNavigate }: Props) {
  const [items, setItems] = useState<SalesOrder[]>([])
  const [total, setTotal] = useState(0)
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const canCreate = role === 'sales' || role === 'sales_head' || role === 'admin'

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listSalesOrders(1, 100, status || undefined)
      const { items: list, total: count } = unwrapPaged<SalesOrder>(res)
      setItems(list)
      setTotal(count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sales orders')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status])

  const visible = items.filter((order) => {
    if (!search.trim()) return true
    const q = search.trim().toLowerCase()
    return (
      order.order_number?.toLowerCase().includes(q) || order.customer_name?.toLowerCase().includes(q)
    )
  })

  const columns: Column<SalesOrder>[] = [
    { key: 'no', header: 'Order Number', render: (r) => <strong>{r.order_number}</strong> },
    { key: 'cust', header: 'Customer', render: (r) => r.customer_name || '—' },
    { key: 'date', header: 'Order Date', render: (r) => fmtDate(r.order_date) },
    { key: 'req', header: 'Required Date', render: (r) => fmtDate(r.required_date) },
    { key: 'lines', header: 'Lines', align: 'right', render: (r) => r.items?.length ?? 0 },
    { key: 'total', header: 'Total Amount', align: 'right', render: (r) => fmtMoney(r.total_amount) },
    {
      key: 'status',
      header: 'Status',
      render: (r) => <StatusBadge label={statusLabel(r.status)} tone={statusTone(r.status)} />,
    },
  ]

  return (
    <>
      <PageHeader
        title={<>Sales Orders</>}
        subtitle="Stock is reserved as soon as an order is created."
        actions={
          canCreate ? (
            <Button variant="primary" onClick={() => onNavigate('Create Sales Order')}>
              + New Sales Order
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
              placeholder="Search by order number or customer…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
        </div>
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${visible.length} of ${total} order${total === 1 ? '' : 's'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading sales orders…</div>
        ) : visible.length === 0 ? (
          <EmptyState
            title="No sales orders found"
            description={
              canCreate ? 'Create a sales order to reserve finished goods.' : 'No orders under this filter yet.'
            }
            icon="sales"
          />
        ) : (
          <DataTable
            columns={columns}
            rows={visible}
            rowKey={(r) => r.id}
            onRowClick={(r) => onNavigate('Sales Order Detail', r.id)}
          />
        )}
      </div>
    </>
  )
}
