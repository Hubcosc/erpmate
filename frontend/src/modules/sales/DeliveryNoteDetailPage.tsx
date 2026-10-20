import { useEffect, useState } from 'react'
import { downloadDeliveryNotePDF, getDeliveryNote, type DeliveryNote } from '../../api/sales'
import Button from '../../shared/components/Button'
import Card from '../../shared/components/Card'
import { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import Toolbar from '../../shared/components/Toolbar'
import { fmtDate, fmtQty, unwrap } from '../../shared/utils/format'

type Props = {
  id: string
  onNavigate: (page: string, id?: string) => void
}

export default function DeliveryNoteDetailPage({ id, onNavigate }: Props) {
  const [note, setNote] = useState<DeliveryNote | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getDeliveryNote(id)
      setNote(unwrap<DeliveryNote>(res))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load delivery note')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const handleDownload = async () => {
    if (!note) return
    try {
      const blob = await downloadDeliveryNotePDF(note.id)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${note.delivery_number}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to download the delivery note PDF')
    }
  }

  if (loading) return <div className="ui-state">Loading delivery note…</div>
  if (error && !note) return <ErrorAlert message={error} onRetry={() => void load()} />
  if (!note) return null

  const totalQty = (note.items ?? []).reduce((sum, i) => sum + (i.quantity_delivered ?? 0), 0)

  return (
    <>
      <PageHeader
        title={
          <>
            Delivery Note <strong>{note.delivery_number}</strong>
          </>
        }
        subtitle={`Dispatch document for sales order ${note.order_number || '—'}.`}
        crumbs={[
          { label: 'Delivery notes', onClick: () => onNavigate('Delivery notes') },
          { label: note.delivery_number },
        ]}
        actions={
          <Button variant="primary" onClick={() => void handleDownload()}>
            Download PDF
          </Button>
        }
      />

      <Toolbar actions={[{ label: 'Back', icon: 'arrow', onClick: () => onNavigate('Delivery notes') }]} />

      {error && <ErrorAlert message={error} />}

      <Card title="Delivery Details" bordered>
        <div className="ui-facts">
          <div className="ui-fact">
            <span>Sales order</span>
            <strong>{note.order_number || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Delivered by</span>
            <strong>{note.delivered_by_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Delivery date</span>
            <strong>{fmtDate(note.delivery_date)}</strong>
          </div>
          <div className="ui-fact">
            <span>Recipient</span>
            <strong>{note.recipient_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Signature reference</span>
            <strong>{note.recipient_signature_reference || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Total quantity</span>
            <strong>{fmtQty(totalQty)}</strong>
          </div>
        </div>
        {note.notes && (
          <p style={{ margin: '12px 0 0', fontSize: 13, color: 'var(--color-text-muted)' }}>{note.notes}</p>
        )}
      </Card>

      <Card title="Delivered Items" bordered>
        <table className="ui-lines">
          <thead>
            <tr>
              <th>Item</th>
              <th>Batch</th>
              <th style={{ textAlign: 'right' }}>Quantity Delivered</th>
            </tr>
          </thead>
          <tbody>
            {(note.items ?? []).map((item) => (
              <tr key={item.id}>
                <td>
                  <strong>{item.item_code}</strong> {item.item_name}
                </td>
                <td>{item.batch_number || '—'}</td>
                <td className="num">{fmtQty(item.quantity_delivered)}</td>
              </tr>
            ))}
            {(note.items ?? []).length === 0 && (
              <tr>
                <td className="table-empty" colSpan={3}>
                  No lines on this delivery note.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>
    </>
  )
}
