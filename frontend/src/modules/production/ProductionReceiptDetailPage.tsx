import { useEffect, useState } from 'react'
import { getProductionReceipt, type ProductionReceipt } from '../../api/production'
import Card from '../../shared/components/Card'
import { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge from '../../shared/components/StatusBadge'
import Toolbar from '../../shared/components/Toolbar'
import { fmtDate, fmtMoney, fmtQty, unwrap } from '../../shared/utils/format'

type Props = {
  id: string
  onNavigate: (page: string, id?: string) => void
}

export default function ProductionReceiptDetailPage({ id, onNavigate }: Props) {
  const [receipt, setReceipt] = useState<ProductionReceipt | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getProductionReceipt(id)
      setReceipt(unwrap<ProductionReceipt>(res))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load production receipt')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  if (loading) return <div className="ui-state">Loading production receipt…</div>
  if (error && !receipt) return <ErrorAlert message={error} onRetry={() => void load()} />
  if (!receipt) return null

  const accepted = (receipt.items ?? []).filter((i) => i.quality_status === 'accepted')

  return (
    <>
      <PageHeader
        title={
          <>
            Production Receipt <strong>{receipt.receipt_number}</strong>
          </>
        }
        subtitle={`Received into ${receipt.warehouse_name || 'warehouse'} on ${fmtDate(receipt.receipt_date)}.`}
        crumbs={[
          { label: 'Production receipts', onClick: () => onNavigate('Production receipts') },
          { label: receipt.receipt_number },
        ]}
      />

      <Toolbar actions={[{ label: 'Back', icon: 'arrow', onClick: () => onNavigate('Production receipts') }]} />

      {error && <ErrorAlert message={error} />}

      <Card title="Receipt Details" bordered>
        <div className="ui-facts">
          <div className="ui-fact">
            <span>Warehouse</span>
            <strong>{receipt.warehouse_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Recorded by</span>
            <strong>{receipt.recorded_by_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Receipt date</span>
            <strong>{fmtDate(receipt.receipt_date)}</strong>
          </div>
        </div>
        {receipt.notes && (
          <p style={{ margin: '12px 0 0', fontSize: 13, color: 'var(--color-text-muted)' }}>{receipt.notes}</p>
        )}
      </Card>

      <Card title="Produced Items" bordered>
        <table className="ui-lines">
          <thead>
            <tr>
              <th>Item</th>
              <th style={{ textAlign: 'right' }}>Quantity</th>
              <th>Batch</th>
              <th>Expiry</th>
              <th style={{ textAlign: 'right' }}>Unit Cost</th>
              <th>Quality</th>
            </tr>
          </thead>
          <tbody>
            {(receipt.items ?? []).map((item) => (
              <tr key={item.id}>
                <td>
                  <strong>{item.item_code}</strong> {item.item_name}
                </td>
                <td className="num">{fmtQty(item.quantity_produced)}</td>
                <td>{item.batch_number || '—'}</td>
                <td>{fmtDate(item.expiry_date)}</td>
                <td className="num">{item.unit_cost === null || item.unit_cost === undefined ? '—' : fmtMoney(item.unit_cost)}</td>
                <td>
                  {item.quality_status === 'accepted' ? (
                    <StatusBadge label="Accepted" tone="positive" />
                  ) : (
                    <StatusBadge label="Rejected" tone="negative" />
                  )}
                </td>
              </tr>
            ))}
            {(receipt.items ?? []).length === 0 && (
              <tr>
                <td className="table-empty" colSpan={6}>
                  No lines on this receipt.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <div className="ui-totals">
        <div className="ui-totals-item">
          <small>Accepted into stock</small>
          {fmtQty(accepted.reduce((sum, i) => sum + (i.quantity_produced ?? 0), 0))}
        </div>
      </div>
    </>
  )
}
