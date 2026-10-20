import { useEffect, useState } from 'react'
import {
  acknowledgeMaterialIssue,
  approveMaterialRequisitionPM,
  createMaterialIssue,
  getMaterialRequisition,
  getMaterialRequisitionHistory,
  listIssuesForMaterialRequisition,
  rejectMaterialRequisitionPM,
  submitMaterialRequisition,
  type MaterialIssue,
  type MaterialRequisition,
  type StatusHistory,
} from '../../api/material'
import Button from '../../shared/components/Button'
import Card from '../../shared/components/Card'
import { ErrorAlert, InfoAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusLabel, statusTone } from '../../shared/components/StatusBadge'
import Toolbar from '../../shared/components/Toolbar'
import { fmtDate, fmtDateTime, fmtQty, unwrap } from '../../shared/utils/format'

type Props = {
  id: string
  role: string
  onNavigate: (page: string, id?: string) => void
}

export default function MaterialRequestDetailPage({ id, role, onNavigate }: Props) {
  const [mr, setMr] = useState<MaterialRequisition | null>(null)
  const [issues, setIssues] = useState<MaterialIssue[]>([])
  const [history, setHistory] = useState<StatusHistory[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const isRequester = role === 'department_head' || role === 'admin'
  const isPM = role === 'production_manager' || role === 'admin'
  const isStore = role === 'store_keeper' || role === 'admin'

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getMaterialRequisition(id)
      setMr(unwrap<MaterialRequisition>(res))
      const [issueRes, historyRes] = await Promise.all([
        listIssuesForMaterialRequisition(id).catch(() => null),
        getMaterialRequisitionHistory(id).catch(() => null),
      ])
      setIssues(issueRes ? unwrap<MaterialIssue[]>(issueRes) ?? [] : [])
      setHistory(historyRes ? unwrap<StatusHistory[]>(historyRes) ?? [] : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load material request')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const run = async (fn: () => Promise<unknown>, message: string) => {
    setBusy(true)
    setError('')
    try {
      await fn()
      await load()
      window.alert(message)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Action failed')
    } finally {
      setBusy(false)
    }
  }

  const handleReject = () => {
    const reason = window.prompt('Reason for rejection:')
    if (!reason) return
    void run(() => rejectMaterialRequisitionPM(id, reason), 'Material request rejected.')
  }

  const handleIssue = () => {
    if (!mr?.items?.length) return
    const lines = mr.items.map((item) => ({
      mr_item_id: item.id ?? '',
      item_id: item.item_id,
      quantity: Math.max(0, item.quantity - (item.issued_qty ?? 0)),
    }))
    const pending = lines.filter((l) => l.quantity > 0)
    if (pending.length === 0) {
      window.alert('Every line on this request has already been issued.')
      return
    }
    void run(() => createMaterialIssue({ mr_id: id, items: pending }), 'Material issue created.')
  }

  const handleAcknowledge = (issueId: string) => {
    void run(() => acknowledgeMaterialIssue(issueId), 'Receipt acknowledged.')
  }

  if (loading) return <div className="ui-state">Loading material request…</div>
  if (error && !mr) return <ErrorAlert message={error} onRetry={() => void load()} />
  if (!mr) return null

  const status = mr.status
  const totalQty = (mr.items ?? []).reduce((sum, i) => sum + (i.quantity ?? 0), 0)
  const issuedQty = (mr.items ?? []).reduce((sum, i) => sum + (i.issued_qty ?? 0), 0)

  return (
    <>
      <PageHeader
        title={
          <>
            Material Request <strong>{mr.mr_number}</strong>
          </>
        }
        subtitle={mr.purpose || 'No purpose recorded.'}
        crumbs={[
          { label: 'Material requests', onClick: () => onNavigate('Material requests') },
          { label: mr.mr_number },
        ]}
        actions={<StatusBadge label={statusLabel(status)} tone={statusTone(status)} />}
      />

      <Toolbar
        actions={[
          {
            label: 'Submit',
            icon: 'arrow',
            onClick: () => void run(() => submitMaterialRequisition(id), 'Material request submitted.'),
            disabled: busy || !(isRequester && status === 'draft'),
          },
          {
            label: 'Approve',
            icon: 'spark',
            tone: 'success',
            onClick: () => void run(() => approveMaterialRequisitionPM(id), 'Material request approved.'),
            disabled: busy || !(isPM && status === 'pending_pm'),
          },
          {
            label: 'Reject',
            icon: 'close',
            tone: 'danger',
            onClick: handleReject,
            disabled: busy || !(isPM && status === 'pending_pm'),
          },
          {
            label: 'Issue',
            icon: 'inventory',
            onClick: handleIssue,
            disabled: busy || !(isStore && (status === 'approved' || status === 'partially_issued')),
          },
          { label: 'Back', icon: 'arrow', onClick: () => onNavigate('Material requests') },
        ]}
      />

      {error && <ErrorAlert message={error} />}

      {status === 'pending_pm' && isPM && (
        <InfoAlert>This request is waiting for production manager approval.</InfoAlert>
      )}
      {(status === 'approved' || status === 'partially_issued') && isStore && (
        <InfoAlert>
          Issue the outstanding quantity from the raw-material store. The department must then acknowledge receipt.
        </InfoAlert>
      )}

      <Card title="Request Details" bordered>
        <div className="ui-facts">
          <div className="ui-fact">
            <span>Department</span>
            <strong>{mr.department_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Requested by</span>
            <strong>{mr.requester_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Required date</span>
            <strong>{fmtDate(mr.required_date)}</strong>
          </div>
          <div className="ui-fact">
            <span>Status</span>
            <strong>{statusLabel(status)}</strong>
          </div>
          <div className="ui-fact">
            <span>Total quantity</span>
            <strong>{fmtQty(totalQty)}</strong>
          </div>
          <div className="ui-fact">
            <span>Issued quantity</span>
            <strong>{fmtQty(issuedQty)}</strong>
          </div>
        </div>
        {mr.pm_rejection_reason && (
          <div className="ui-alert is-error" style={{ marginTop: 12, marginBottom: 0 }}>
            <span>
              <strong>Rejection reason: </strong>
              {mr.pm_rejection_reason}
            </span>
          </div>
        )}
      </Card>

      <Card title="Requested Items" bordered>
        <table className="ui-lines">
          <thead>
            <tr>
              <th>Item</th>
              <th style={{ textAlign: 'right' }}>Requested</th>
              <th style={{ textAlign: 'right' }}>Issued</th>
              <th style={{ textAlign: 'right' }}>Outstanding</th>
              <th>Stock</th>
            </tr>
          </thead>
          <tbody>
            {(mr.items ?? []).map((item) => {
              const outstanding = Math.max(0, item.quantity - (item.issued_qty ?? 0))
              return (
                <tr key={item.id ?? item.item_id}>
                  <td>
                    <strong>{item.item_code}</strong> {item.item_name}
                  </td>
                  <td className="num">{fmtQty(item.quantity)}</td>
                  <td className="num">{fmtQty(item.issued_qty ?? 0)}</td>
                  <td className="num">{fmtQty(outstanding)}</td>
                  <td>
                    {item.stock_sufficient === false ? (
                      <StatusBadge label="Short" tone="warning" />
                    ) : (
                      <StatusBadge label="Available" tone="positive" />
                    )}
                  </td>
                </tr>
              )
            })}
            {(mr.items ?? []).length === 0 && (
              <tr>
                <td className="table-empty" colSpan={5}>
                  No lines on this request.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <Card title="Material Issues" subtitle="Issues posted from the raw-material store." bordered>
        {issues.length === 0 ? (
          <p style={{ margin: 0, fontSize: 13, color: 'var(--color-text-muted)' }}>No issues recorded yet.</p>
        ) : (
          <table className="ui-lines">
            <thead>
              <tr>
                <th>Issue Number</th>
                <th>Issued by</th>
                <th>Date</th>
                <th>Lines</th>
                <th>Acknowledged</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {issues.map((issue) => (
                <tr key={issue.id}>
                  <td>
                    <strong>{issue.issue_number}</strong>
                  </td>
                  <td>{issue.issued_by_name || '—'}</td>
                  <td>{fmtDate(issue.issued_date)}</td>
                  <td>{issue.items?.length ?? 0}</td>
                  <td>
                    {issue.acknowledged_at ? (
                      <StatusBadge label="Acknowledged" tone="positive" />
                    ) : (
                      <StatusBadge label="Pending" tone="warning" />
                    )}
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    {isRequester && !issue.acknowledged_at && (
                      <Button size="sm" onClick={() => handleAcknowledge(issue.id)} disabled={busy}>
                        Acknowledge
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      <Card title="Approval History" bordered>
        {history.length === 0 ? (
          <p style={{ margin: 0, fontSize: 13, color: 'var(--color-text-muted)' }}>No status changes recorded yet.</p>
        ) : (
          <div className="ui-timeline">
            {history.map((entry) => (
              <div className="ui-timeline-entry" key={entry.id}>
                <span className="ui-timeline-dot" />
                <div>
                  <strong>{statusLabel(entry.new_status)}</strong>
                  <small>
                    {entry.changed_by_name || 'System'} · {fmtDateTime(entry.created_at)}
                  </small>
                  {entry.reason && <p>{entry.reason}</p>}
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </>
  )
}
