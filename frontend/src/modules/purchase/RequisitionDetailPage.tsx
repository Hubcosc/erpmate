import { useCallback, useEffect, useRef, useState } from 'react'
import {
  getPR,
  getPRHistory,
  prAction,
  deletePR,
  type PurchaseRequisition,
  type PRHistory,
} from '../../api/purchaseRequisitions'
import Modal from '../../shared/components/Modal'
import './RequisitionDetailPage.css'

type Props = {
  id: string
  onNavigate: (page: string, id?: string) => void
  role: string
}

const STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  pending_pm: 'Pending PM Approval',
  pending_finance: 'Pending Finance Approval',
  approved: 'Approved',
  rejected: 'Rejected',
  returned_for_correction: 'Returned for Correction',
  suspended: 'Suspended',
}

const ACTION_LABELS: Record<string, string> = {
  submit: 'Submit for Approval',
  edit: 'Edit Requisition',
  delete: 'Delete Requisition',
  pm_approve: 'Approve (PM)',
  pm_return: 'Return for Correction',
  pm_reject: 'Reject',
  finance_approve: 'Approve (Finance)',
  finance_suspend: 'Suspend',
  finance_reject: 'Reject',
}

const ACTION_CONFIRM: Record<string, string> = {
  submit: 'Submit this requisition for production manager approval? You will not be able to edit it afterwards.',
  delete: 'Delete this requisition? This cannot be undone.',
  pm_approve: 'Approve this requisition and send it to Finance?',
  pm_return: 'Return this requisition to the requester for correction?',
  pm_reject: 'Reject this requisition?',
  finance_approve: 'Approve this requisition? A purchase order draft will be created for the purchasing team.',
  finance_suspend: 'Suspend this requisition until more information is available?',
  finance_reject: 'Reject this requisition?',
}

// Endpoint name for every workflow action (UI action -> URL segment).
const ENDPOINTS: Record<string, string> = {
  submit: 'submit',
  pm_approve: 'pm-approve',
  pm_reject: 'pm-reject',
  pm_return: 'pm-return',
  finance_approve: 'finance-approve',
  finance_suspend: 'finance-suspend',
  finance_reject: 'finance-reject',
}

const REASON_REQUIRED = ['pm_reject', 'pm_return', 'finance_suspend', 'finance_reject']
const MAX_REASON = 1000

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Accepts either the raw payload or a { data: payload } envelope. */
function unwrap<T>(res: unknown): T {
  if (res && typeof res === 'object' && 'data' in (res as Record<string, unknown>)) {
    return (res as { data: T }).data
  }
  return res as T
}

/** Always returns an array, whatever the API sent (null, object, envelope...). */
function toArray<T>(res: unknown): T[] {
  const value = unwrap<unknown>(res)
  if (Array.isArray(value)) return value as T[]
  if (value && typeof value === 'object' && Array.isArray((value as { data?: unknown }).data)) {
    return (value as { data: T[] }).data
  }
  return []
}

// required_date arrives as midnight UTC, so format it in UTC to avoid off-by-one days.
function formatDate(value?: string | null): string {
  if (!value) return '—'
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleDateString(undefined, { timeZone: 'UTC' })
}

function formatDateTime(value?: string | null): string {
  if (!value) return ''
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString()
}

function formatMoney(value?: number | null): string {
  if (value === null || value === undefined) return '—'
  return value.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

function actionButtonClass(action: string): string {
  if (action === 'delete' || action.includes('reject')) return 'pr-btn pr-btn-danger'
  if (action === 'submit' || action.includes('approve')) return 'pr-btn pr-btn-primary'
  return 'pr-btn pr-btn-ghost'
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function RequisitionDetailPage({ id, onNavigate }: Props) {
  const [pr, setPr] = useState<PurchaseRequisition | null>(null)
  const [history, setHistory] = useState<PRHistory[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Action modal state
  const [actionLoading, setActionLoading] = useState(false)
  const [actionType, setActionType] = useState<string | null>(null)
  const [actionReason, setActionReason] = useState('')
  const [actionError, setActionError] = useState('')

  // Ignore responses of outdated requests (e.g. when the id changes quickly).
  const requestId = useRef(0)

  const loadData = useCallback(
    async (silent = false) => {
      const current = ++requestId.current
      if (!silent) {
        setLoading(true)
        setError('')
      }
      try {
        const [prRes, histRes] = await Promise.all([getPR(id), getPRHistory(id)])
        if (current !== requestId.current) return
        setPr(unwrap<PurchaseRequisition>(prRes))
        setHistory(toArray<PRHistory>(histRes))
      } catch (err) {
        if (current !== requestId.current) return
        if (!silent) setError(err instanceof Error ? err.message : 'Failed to load requisition')
      } finally {
        if (current === requestId.current && !silent) setLoading(false)
      }
    },
    [id],
  )

  useEffect(() => {
    void loadData()
    return () => {
      requestId.current++ // cancel pending responses on unmount / id change
    }
  }, [loadData])

  const closeModal = () => {
    if (actionLoading) return
    setActionType(null)
    setActionReason('')
    setActionError('')
  }

  const handleActionClick = (action: string) => {
    if (action === 'edit') {
      onNavigate('Create PR', id)
      return
    }
    setActionType(action)
    setActionReason('')
    setActionError('')
  }

  const executeAction = async () => {
    if (!actionType) return
    const reason = actionReason.trim()
    if (REASON_REQUIRED.includes(actionType) && !reason) {
      setActionError('A reason is required.')
      return
    }

    setActionLoading(true)
    setActionError('')
    try {
      if (actionType === 'delete') {
        await deletePR(id)
        onNavigate('Purchase requisitions')
        return
      }

      const endpoint = ENDPOINTS[actionType]
      if (!endpoint) throw new Error('Unknown action')

      // The backend returns the updated requisition (with available_actions).
      const res = await prAction(id, endpoint as Parameters<typeof prAction>[1],
        (reason || undefined) as Parameters<typeof prAction>[2],
        )
      setPr(unwrap<PurchaseRequisition>(res))

      setActionType(null)
      setActionReason('')

      // Refresh the history timeline in the background.
      try {
        setHistory(toArray<PRHistory>(await getPRHistory(id)))
      } catch {
        /* history is secondary; keep the old timeline if it fails */
      }
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Action failed')
      // Someone else may have changed the requisition: refresh so buttons are correct.
      void loadData(true)
    } finally {
      setActionLoading(false)
    }
  }

  if (loading) return <div className="pr-detail-page"><p className="pr-state">Loading requisition...</p></div>
  if (error) {
    return (
      <div className="pr-detail-page">
        <p className="pr-state pr-state-error" role="alert">{error}</p>
        <div className="pr-actions-bar">
          <button type="button" className="pr-btn pr-btn-ghost" onClick={() => onNavigate('Purchase requisitions')}>
            Back to List
          </button>
          <button type="button" className="pr-btn pr-btn-primary" onClick={() => void loadData()}>
            Try again
          </button>
        </div>
      </div>
    )
  }
  if (!pr) return <div className="pr-detail-page"><p className="pr-state">Requisition not found.</p></div>

  const items = pr.items ?? []
  const stockCheck = pr.stock_check ?? []
  const actions = pr.available_actions ?? []
  const requiresReason = actionType ? REASON_REQUIRED.includes(actionType) : false
  const showComment = !!actionType && !requiresReason && actionType !== 'submit' && actionType !== 'delete'
  const destructive = actionType === 'delete' || (actionType?.includes('reject') ?? false)

  return (
    <div className="pr-detail-page">
      <header className="pr-detail-header">
        <div>
          <p className="pr-eyebrow">Purchase Requisition</p>
          <div className="pr-title-row">
            <h1>{pr.pr_number}</h1>
            <span className={`pr-pill status-${pr.status}`}>{STATUS_LABELS[pr.status] || pr.status}</span>
          </div>
          <p className="pr-subtitle">
            Requested by <strong>{pr.requester_name || '—'}</strong> on {formatDate(pr.created_at)}
          </p>
        </div>
        <div className="pr-actions-bar">
          <button type="button" className="pr-btn pr-btn-ghost" onClick={() => onNavigate('Purchase requisitions')}>
            Back to List
          </button>
          {actions.map((action) => (
            <button
              key={action}
              type="button"
              className={actionButtonClass(action)}
              onClick={() => handleActionClick(action)}
              disabled={actionLoading}
            >
              {ACTION_LABELS[action] || action}
            </button>
          ))}
        </div>
      </header>

      {pr.return_reason && (
        <div className="pr-alert pr-alert-warning" role="status">
          <strong>Returned for correction:</strong> {pr.return_reason}
        </div>
      )}
      {pr.pm_rejection_reason && (
        <div className="pr-alert pr-alert-danger" role="status">
          <strong>Rejected by PM:</strong> {pr.pm_rejection_reason}
        </div>
      )}
      {pr.finance_rejection_reason && (
        <div className="pr-alert pr-alert-danger" role="status">
          <strong>Rejected by Finance:</strong> {pr.finance_rejection_reason}
        </div>
      )}
      {pr.suspension_reason && pr.status === 'suspended' && (
        <div className="pr-alert pr-alert-warning" role="status">
          <strong>Suspended by Finance:</strong> {pr.suspension_reason}
        </div>
      )}

      <div className="pr-detail-grid">
        <div className="pr-detail-main">
          <section className="pr-card">
            <h2 className="pr-card-title">Request Details</h2>
            <div className="pr-info-grid">
              <div><span>Department</span><strong>{pr.department_name || '—'}</strong></div>
              <div><span>Required Date</span><strong>{formatDate(pr.required_date)}</strong></div>
              <div className="pr-info-wide"><span>Justification</span><p>{pr.justification || '—'}</p></div>
            </div>
          </section>

          <section className="pr-card">
            <h2 className="pr-card-title">Line Items ({pr.item_count ?? items.length})</h2>
            <div className="pr-line-table-wrap">
              <table className="pr-line-table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Description</th>
                    <th style={{ textAlign: 'right' }}>Qty</th>
                    <th>Unit</th>
                    <th style={{ textAlign: 'right' }}>Est. Price</th>
                    <th style={{ textAlign: 'right' }}>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {items.length === 0 && (
                    <tr><td colSpan={6} style={{ textAlign: 'center' }}>No items.</td></tr>
                  )}
                  {items.map((item) => (
                    <tr key={item.id}>
                      <td>{item.item_code ? `${item.item_code} - ${item.item_name ?? ''}` : '—'}</td>
                      <td>{item.description || '—'}</td>
                      <td style={{ textAlign: 'right' }}>{item.quantity}</td>
                      <td>{item.unit_of_measure || '—'}</td>
                      <td style={{ textAlign: 'right' }}>{formatMoney(item.estimated_unit_price)}</td>
                      <td style={{ textAlign: 'right' }}>{formatMoney(item.estimated_total)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={5} style={{ textAlign: 'right', fontWeight: 600 }}>Estimated Total:</td>
                    <td style={{ textAlign: 'right', fontWeight: 600, color: 'var(--pr-navy)' }}>
                      {formatMoney(pr.total_estimated ?? 0)}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </section>

          {stockCheck.length > 0 && (
            <section className="pr-card">
              <h2 className="pr-card-title">Stock Availability</h2>
              <div className="pr-line-table-wrap">
                <table className="pr-line-table">
                  <thead>
                    <tr>
                      <th>Item</th>
                      <th style={{ textAlign: 'right' }}>Requested</th>
                      <th style={{ textAlign: 'right' }}>Available</th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {stockCheck.map((sc) => (
                      <tr key={sc.item_id}>
                        <td>{sc.item_code} - {sc.item_name}</td>
                        <td style={{ textAlign: 'right' }}>{sc.requested_quantity}</td>
                        <td style={{ textAlign: 'right' }}>{sc.available_quantity}</td>
                        <td>
                          {sc.sufficient ? (
                            <span className="pr-pill status-approved">Sufficient</span>
                          ) : (
                            <span className="pr-pill status-rejected">Shortfall: {sc.shortfall_quantity}</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {pr.suggest_material_requisition && (
                <div className="pr-alert pr-alert-info" style={{ marginTop: 12 }}>
                  <strong>Note:</strong> All items are currently in stock. You may consider issuing a Material Requisition instead of purchasing new stock.
                </div>
              )}
            </section>
          )}
        </div>

        <div className="pr-detail-side">
          <section className="pr-card">
            <h2 className="pr-card-title">History</h2>
            <div className="pr-timeline">
              {history.length === 0 && <p className="pr-state">No history yet.</p>}
              {history.map((h) => (
                <div key={h.id} className="pr-timeline-item">
                  <div className="pr-timeline-dot" />
                  <div className="pr-timeline-content">
                    <strong>{STATUS_LABELS[h.new_status] || h.new_status}</strong>
                    <span className="pr-timeline-meta">
                      {h.changed_by_name || 'System'} • {formatDateTime(h.created_at)}
                    </span>
                    {h.reason && <p className="pr-timeline-reason">“{h.reason}”</p>}
                  </div>
                </div>
              ))}
            </div>
          </section>
        </div>
      </div>

      <Modal open={!!actionType} title={actionType ? ACTION_LABELS[actionType] ?? '' : ''} onClose={closeModal}>
        <div className="pr-action-modal">
          {actionError && <div className="pr-error" role="alert">{actionError}</div>}

          <p>{actionType ? ACTION_CONFIRM[actionType] ?? 'Are you sure?' : ''}</p>

          {(requiresReason || showComment) && (
            <label className="pr-field">
              <span>{requiresReason ? <>Reason <i>*</i></> : 'Comment (optional)'}</span>
              <textarea
                value={actionReason}
                onChange={(e) => setActionReason(e.target.value)}
                placeholder={requiresReason ? 'Please provide a reason...' : 'Add an optional comment...'}
                rows={requiresReason ? 3 : 2}
                maxLength={MAX_REASON}
                disabled={actionLoading}
              />
            </label>
          )}

          <div className="pr-modal-actions">
            <button type="button" className="pr-btn pr-btn-ghost" onClick={closeModal} disabled={actionLoading}>
              Cancel
            </button>
            <button
              type="button"
              className={destructive ? 'pr-btn pr-btn-danger' : 'pr-btn pr-btn-primary'}
              onClick={() => void executeAction()}
              disabled={actionLoading || (requiresReason && !actionReason.trim())}
            >
              {actionLoading ? 'Processing...' : 'Confirm'}
            </button>
          </div>
        </div>
      </Modal>
    </div>
  )
}