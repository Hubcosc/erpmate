import { useEffect, useState } from 'react'
import { listPOs, type POSummary } from '../../api/purchaseOrders'
import { Icon } from '../../shared/components/Icon'
import './PurchaseOrdersListPage.css'

type Props = {
  onNavigate: (page: string, id?: string) => void
}

const STATUS_LABELS: Record<string, string> = {
  all: 'All',
  draft: 'Draft',
  issued: 'Issued',
  partially_received: 'Partially Received',
  received: 'Received',
}

export default function PurchaseOrdersListPage({ onNavigate }: Props) {
  const [pos, setPOs] = useState<POSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [activeTab, setActiveTab] = useState('all')
  const [search, setSearch] = useState('')

  const loadData = async (statusFilter: string, searchValue: string) => {
    setLoading(true)
    setError('')
    try {
      const res = await listPOs({
        status: statusFilter === 'all' ? undefined : statusFilter,
        search: searchValue || undefined,
        limit: 50,
      })
      // Handle envelope vs raw array
      const list = (res as any).data ? (res as any).data : res
      setPOs(Array.isArray(list) ? list : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load Purchase Orders')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData(activeTab, search)
  }, [activeTab])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    void loadData(activeTab, search)
  }

  return (
    <div className="po-page">
      <header className="po-header">
        <div>
          <p className="po-eyebrow">Procurement</p>
          <h1>Purchase Orders</h1>
          <p className="po-subtitle">Manage and issue purchase orders to suppliers.</p>
        </div>
      </header>

      <div className="po-tabs">
        {Object.keys(STATUS_LABELS).map((key) => (
          <button
            key={key}
            className={`po-tab ${activeTab === key ? 'is-active' : ''}`}
            onClick={() => setActiveTab(key)}
          >
            {STATUS_LABELS[key]}
          </button>
        ))}
      </div>

      <form className="po-toolbar" onSubmit={handleSearch}>
        <label className="po-search">
          <Icon name="search" size={16} />
          <input
            type="search"
            placeholder="Search by PO number or supplier…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button className="po-btn" type="submit">Search</button>
      </form>

      {loading && <p className="po-state">Loading Purchase Orders…</p>}
      {error && <p className="po-state po-state-error">{error}</p>}
      {!loading && !error && pos.length === 0 && (
        <p className="po-state">No Purchase Orders found in this view.</p>
      )}

      {!loading && !error && pos.length > 0 && (
        <div className="po-table-wrap">
          <table className="po-table">
            <thead>
              <tr>
                <th>PO Number</th>
                <th>Supplier</th>
                <th>Expected Date</th>
                <th>Items</th>
                <th style={{ textAlign: 'right' }}>Total Amount</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {pos.map((po) => (
                <tr key={po.id} onClick={() => onNavigate('PO Detail', po.id)} style={{ cursor: 'pointer' }}>
                  <td><strong>{po.po_number}</strong></td>
                  <td>{po.supplier_name || '—'}</td>
                  <td>{po.expected_delivery_date ? new Date(po.expected_delivery_date).toLocaleDateString() : '—'}</td>
                  <td>{po.item_count}</td>
                  <td style={{ textAlign: 'right' }}>
                    {po.total_amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                  </td>
                  <td>
                    <span className={`po-pill status-${po.status}`}>
                      {STATUS_LABELS[po.status] || po.status}
                    </span>
                  </td>
                  <td style={{ textAlign: 'right' }}><Icon name="chevron" size={16} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}