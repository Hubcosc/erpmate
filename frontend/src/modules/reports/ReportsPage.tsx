import { useEffect, useState } from 'react'
import {
  getMaterialConsumptionReport,
  getPurchaseCommitmentsReport,
  getReportSummary,
  getSalesReport,
  type ConsumptionReportRow,
  type PurchaseCommitmentRow,
  type ReportSummary,
  type SalesReportRow,
} from '../../api/reports'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusLabel, statusTone } from '../../shared/components/StatusBadge'
import Tabs from '../../shared/components/Tabs'
import { fmtMoney, fmtQty, monthStartISO, todayISO, unwrap } from '../../shared/utils/format'

const REPORT_TABS = [
  { key: 'summary', label: 'Summary' },
  { key: 'sales', label: 'Sales' },
  { key: 'consumption', label: 'Material Consumption' },
  { key: 'commitments', label: 'Purchase Commitments' },
]

export default function ReportsPage() {
  const [tab, setTab] = useState('summary')
  const [from, setFrom] = useState(monthStartISO())
  const [to, setTo] = useState(todayISO())
  const [groupBy, setGroupBy] = useState<'product' | 'customer'>('product')

  const [summary, setSummary] = useState<ReportSummary | null>(null)
  const [sales, setSales] = useState<SalesReportRow[]>([])
  const [consumption, setConsumption] = useState<ConsumptionReportRow[]>([])
  const [commitments, setCommitments] = useState<PurchaseCommitmentRow[]>([])

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      if (tab === 'summary') {
        setSummary(unwrap<ReportSummary>(await getReportSummary({ from, to })))
      } else if (tab === 'sales') {
        setSales(unwrap<SalesReportRow[]>(await getSalesReport({ from, to }, groupBy)) ?? [])
      } else if (tab === 'consumption') {
        setConsumption(unwrap<ConsumptionReportRow[]>(await getMaterialConsumptionReport({ from, to })) ?? [])
      } else {
        setCommitments(unwrap<PurchaseCommitmentRow[]>(await getPurchaseCommitmentsReport()) ?? [])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load the report')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, from, to, groupBy])

  const salesColumns: Column<SalesReportRow>[] = [
    { key: 'name', header: groupBy === 'product' ? 'Product' : 'Customer', render: (r) => <strong>{r.name}</strong> },
    { key: 'units', header: 'Units Delivered', align: 'right', render: (r) => fmtQty(r.units_delivered) },
    { key: 'revenue', header: 'Revenue', align: 'right', render: (r) => fmtMoney(r.revenue) },
  ]

  const consumptionColumns: Column<ConsumptionReportRow>[] = [
    { key: 'code', header: 'Item Code', render: (r) => <strong>{r.item_code}</strong> },
    { key: 'name', header: 'Item', render: (r) => r.item_name },
    { key: 'uom', header: 'Unit', render: (r) => r.unit_of_measure },
    { key: 'qty', header: 'Quantity Issued', align: 'right', render: (r) => fmtQty(r.quantity_issued) },
    { key: 'count', header: 'Issues', align: 'right', render: (r) => r.issue_count },
  ]

  const commitmentColumns: Column<PurchaseCommitmentRow>[] = [
    { key: 'po', header: 'PO Number', render: (r) => <strong>{r.po_number}</strong> },
    { key: 'supplier', header: 'Supplier', render: (r) => r.supplier_name || '—' },
    { key: 'status', header: 'Status', render: (r) => <StatusBadge label={statusLabel(r.status)} tone={statusTone(r.status)} /> },
    { key: 'ordered', header: 'Ordered Value', align: 'right', render: (r) => fmtMoney(r.ordered_value) },
    { key: 'received', header: 'Received Value', align: 'right', render: (r) => fmtMoney(r.received_value) },
    { key: 'outstanding', header: 'Outstanding', align: 'right', render: (r) => fmtMoney(r.outstanding_value) },
  ]

  return (
    <>
      <PageHeader
        title={<>Reports</>}
        subtitle="Sales revenue, material consumption and open purchase commitments."
      />

      <Tabs tabs={REPORT_TABS} active={tab} onChange={setTab} />

      <div className="ui-table-toolbar" style={{ marginTop: 16 }}>
        <div className="ui-table-filters">
          <label className="ui-field" style={{ margin: 0 }}>
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} aria-label="From date" />
          </label>
          <label className="ui-field" style={{ margin: 0 }}>
            <input type="date" value={to} onChange={(e) => setTo(e.target.value)} aria-label="To date" />
          </label>
          {tab === 'sales' && (
            <select className="ui-select" value={groupBy} onChange={(e) => setGroupBy(e.target.value as 'product' | 'customer')}>
              <option value="product">By product</option>
              <option value="customer">By customer</option>
            </select>
          )}
        </div>
        <div className="ui-table-filters">
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      {loading ? (
        <div className="ui-state">Loading report…</div>
      ) : tab === 'summary' ? (
        <div className="ui-tiles" style={{ gridTemplateColumns: 'repeat(4, minmax(0, 1fr))' }}>
          <Tile label="Sales Revenue" value={fmtMoney(summary?.sales_revenue)} note={`${summary?.delivery_count ?? 0} deliveries`} />
          <Tile label="Units Delivered" value={fmtQty(summary?.units_delivered)} note={`${summary?.delivery_count ?? 0} delivery notes`} />
          <Tile label="Purchase Commitments" value={fmtMoney(summary?.purchase_commitments)} note="Outstanding on issued POs" />
          <Tile label="Pending Requests" value={fmtQty((summary?.pending_purchase_requests ?? 0) + (summary?.pending_material_requests ?? 0))} note={`${summary?.pending_purchase_requests ?? 0} PR · ${summary?.pending_material_requests ?? 0} MR`} />
          <Tile label="Low Stock Items" value={fmtQty(summary?.low_stock_items)} note="At or below reorder level" />
          <Tile label="Near Expiry Batches" value={fmtQty(summary?.near_expiry_batches)} note="Within the expiry window" />
          <Tile label="Expired Batches" value={fmtQty(summary?.expired_batches)} note="Already expired" />
          <Tile label="Period" value={`${from} → ${to}`} note="Report period" />
        </div>
      ) : tab === 'sales' ? (
        <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
          <DataTable columns={salesColumns} rows={sales} rowKey={(r) => r.key} empty={<EmptyState title="No sales in this period" description="Nothing was delivered in the selected dates." icon="reports" />} />
        </div>
      ) : tab === 'consumption' ? (
        <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
          <DataTable columns={consumptionColumns} rows={consumption} rowKey={(r) => r.item_id} empty={<EmptyState title="No material issued" description="No material was issued in the selected dates." icon="materials" />} />
        </div>
      ) : (
        <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
          <DataTable columns={commitmentColumns} rows={commitments} rowKey={(r) => r.purchase_order_id} empty={<EmptyState title="No open commitments" description="Every issued purchase order is fully received." icon="purchase" />} />
        </div>
      )}
    </>
  )
}

function Tile({ label, value, note }: { label: string; value: string; note?: string }) {
  return (
    <div className="ui-tile">
      <span className="ui-tile-label">{label}</span>
      <span className="ui-tile-value">{value}</span>
      {note && <span className="ui-tile-note">{note}</span>}
    </div>
  )
}
