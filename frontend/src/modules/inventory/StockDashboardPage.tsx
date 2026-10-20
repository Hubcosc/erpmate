import { useEffect, useState } from 'react'
import { 
  createInventoryApi, 
  STOCK_STATUS_LABEL, 
  formatQty, 
  type StockBalance, 
  type StockSummary,
  type StockStatus
} from '../../api/inventory'
import { Icon } from '../../shared/components/Icon'
import './StockDashboardPage.css'

// Initialize the API here (or export it from a shared file)
const api = createInventoryApi({
  baseUrl: 'http://localhost:8080/api/v1/inventory',
  getToken: () => localStorage.getItem('token'), // Adjust if your token key is different
})

type Props = {
  onNavigate: (page: string) => void
}

const TABS: { key: string; label: string }[] = [
  { key: 'all', label: 'All Items' },
  { key: 'in_stock', label: 'In Stock' },
  { key: 'low_stock', label: 'Low Stock' },
  { key: 'out_of_stock', label: 'Out of Stock' },
]

export default function StockDashboardPage({ onNavigate }: Props) {
  const [stock, setStock] = useState<StockBalance[]>([])
  const [summary, setSummary] = useState<StockSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  
  const [activeTab, setActiveTab] = useState('all')
  const [search, setSearch] = useState('')

  const loadData = async (statusFilter: string, searchValue: string) => {
    setLoading(true)
    setError('')
    try {
      const statusArray = statusFilter === 'all' ? undefined : [statusFilter as StockStatus]
      
      const [stockRes, summaryRes] = await Promise.all([
        api.listStock({ 
          status: statusArray, 
          search: searchValue || undefined,
          limit: 100 
        }),
        api.stockSummary()
      ])
      
      setStock(stockRes.items)
      setSummary(summaryRes)
    } catch (err: any) {
      setError(err.message || 'Failed to load inventory')
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
    <div className="inv-page">
      <header className="inv-header">
        <div>
          <p className="inv-eyebrow">Inventory</p>
          <h1>Stock Dashboard</h1>
          <p className="inv-subtitle">Current stock levels across all warehouses.</p>
        </div>
      </header>

      {/* Summary Cards */}
      {summary && (
        <div className="inv-summary-grid">
          <div className="inv-summary-card">
            <span>Total Items</span>
            <strong>{summary.total_items}</strong>
          </div>
          <div className="inv-summary-card text-green">
            <span>In Stock Lines</span>
            <strong>{summary.in_stock}</strong>
          </div>
          <div className="inv-summary-card text-amber">
            <span>Low Stock Lines</span>
            <strong>{summary.low_stock}</strong>
          </div>
          <div className="inv-summary-card text-red">
            <span>Out of Stock Lines</span>
            <strong>{summary.out_of_stock}</strong>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="inv-tabs">
        {TABS.map((tab) => {
          let count = summary?.total_lines
          if (tab.key === 'in_stock') count = summary?.in_stock
          if (tab.key === 'low_stock') count = summary?.low_stock
          if (tab.key === 'out_of_stock') count = summary?.out_of_stock

          return (
            <button
              key={tab.key}
              className={`inv-tab ${activeTab === tab.key ? 'is-active' : ''}`}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
              {count !== undefined && <span className="inv-tab-count">{count}</span>}
            </button>
          )
        })}
      </div>

      {/* Toolbar */}
      <form className="inv-toolbar" onSubmit={handleSearch}>
        <label className="inv-search">
          <Icon name="search" size={16} />
          <input
            type="search"
            placeholder="Search by item code or name..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button className="inv-btn inv-btn-primary" type="submit">Search</button>
      </form>

      {/* Table */}
      {loading && <p className="inv-state">Loading stock...</p>}
      {error && <p className="inv-state inv-state-error">{error}</p>}
      {!loading && !error && stock.length === 0 && (
        <p className="inv-state">No stock records found.</p>
      )}

      {!loading && !error && stock.length > 0 && (
        <div className="inv-table-wrap">
          <table className="inv-table">
            <thead>
              <tr>
                <th>Item</th>
                <th>Warehouse</th>
                <th style={{ textAlign: 'right' }}>Total Qty</th>
                <th style={{ textAlign: 'right' }}>Reserved</th>
                <th style={{ textAlign: 'right' }}>Available</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {stock.map((line, idx) => (
                <tr key={`${line.item_id}-${line.warehouse_id}-${idx}`}>
                  <td>
                    <strong>{line.item_code}</strong> — {line.item_name}
                  </td>
                  <td>{line.warehouse_name}</td>
                  <td style={{ textAlign: 'right' }}>{formatQty(line.quantity)} {line.unit_of_measure}</td>
                  <td style={{ textAlign: 'right' }}>{formatQty(line.reserved_quantity)}</td>
                  <td style={{ textAlign: 'right', fontWeight: 600 }}>{formatQty(line.available_quantity)}</td>
                  <td>
                    <span className={`inv-pill status-${line.stock_status}`}>
                      {STOCK_STATUS_LABEL[line.stock_status]}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}