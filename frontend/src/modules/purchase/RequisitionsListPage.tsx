import { useEffect, useState } from 'react'
import { listPRs, getPRSummary, type PRSummary, type PRStatus, type PRStatusSummary } from '../../api/purchaseRequisitions'
import { Icon } from '../../shared/components/Icon'
import './RequisitionsListPage.css'

type Props = {
  onNavigate: (page: string, id?: string) => void
  role: string
}

const STATUS_LABELS: Record<string, string> = {
  all: 'All',
  draft: 'Draft',
  pending_pm: 'Pending PM',
  pending_finance: 'Pending Finance',
  approved: 'Approved',
  rejected: 'Rejected',
  returned_for_correction: 'Returned',
  suspended: 'Suspended',
}

export default function RequisitionsListPage({ onNavigate, role }: Props) {
  const [prs, setPrs] = useState<PRSummary[]>([])
  const [summary, setSummary] = useState<PRStatusSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  
  const [activeTab, setActiveTab] = useState<string>('all')
  const [search, setSearch] = useState('')

  const loadData = async (statusFilter: string, searchValue: string) => {
    setLoading(true)
    setError('')
    try {
      const [listRes, summaryRes] = await Promise.all([
        listPRs({ status: statusFilter === 'all' ? undefined : statusFilter, q: searchValue, limit: 50 }),
        getPRSummary(),
      ])
      setPrs(listRes.data)
      setSummary(summaryRes)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load requisitions')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData(activeTab, search)
  }, [activeTab]) // Reload when tab changes

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    void loadData(activeTab, search)
  }

  const isDeptHead = role === 'department_head' || role === 'admin'

  return (
    <div className="pr-page">
      <header className="pr-header">
        <div>
          <p className="pr-eyebrow">Operations</p>
          <h1>Purchase Requisitions</h1>
          <p className="pr-subtitle">Create, track, and approve purchase requests.</p>
        </div>
        {isDeptHead && (
          <button className="pr-btn pr-btn-primary" onClick={() => onNavigate('Create PR')}>
            <Icon name="settings" size={16} /> New Requisition
          </button>
        )}
      </header>

      {/* Tabs */}
      <div className="pr-tabs">
        {Object.keys(STATUS_LABELS).map((key) => {
          const count = key === 'all' ? summary?.total : summary?.counts?.[key as PRStatus]
          return (
            <button
              key={key}
              className={`pr-tab ${activeTab === key ? 'is-active' : ''}`}
              onClick={() => setActiveTab(key)}
            >
              {STATUS_LABELS[key]}
              {count !== undefined && <span className="pr-tab-count">{count}</span>}
            </button>
          )
        })}
      </div>

      <form className="pr-toolbar" onSubmit={handleSearch}>
        <label className="pr-search">
          <Icon name="search" size={16} />
          <input
            type="search"
            placeholder="Search by PR number, requester, or justification…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button className="pr-btn" type="submit">Search</button>
      </form>

      {loading && <p className="pr-state">Loading requisitions…</p>}
      {error && <p className="pr-state pr-state-error">{error}</p>}
      {!loading && !error && prs.length === 0 && (
        <p className="pr-state">No requisitions found in this view.</p>
      )}

      {!loading && !error && prs.length > 0 && (
        <div className="pr-table-wrap">
          <table className="pr-table">
            <thead>
              <tr>
                <th>PR Number</th>
                <th>Department</th>
                <th>Requester</th>
                <th>Date</th>
                <th>Items</th>
                <th style={{ textAlign: 'right' }}>Est. Total</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {prs.map((pr) => (
                <tr key={pr.id} onClick={() => onNavigate('PR Detail', pr.id)} style={{ cursor: 'pointer' }}>
                  <td><strong>{pr.pr_number}</strong></td>
                  <td>{pr.department_name}</td>
                  <td>{pr.requester_name}</td>
                  <td>{new Date(pr.created_at).toLocaleDateString()}</td>
                  <td>{pr.item_count}</td>
                  <td style={{ textAlign: 'right' }}>
                    {pr.total_estimated > 0 ? pr.total_estimated.toLocaleString(undefined, { minimumFractionDigits: 2 }) : '—'}
                  </td>
                  <td>
                    <span className={`pr-pill status-${pr.status}`}>
                      {STATUS_LABELS[pr.status] || pr.status}
                    </span>
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <Icon name="chevron" size={16} />
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