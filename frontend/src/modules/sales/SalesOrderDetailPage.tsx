import { useEffect, useState } from 'react'
import { createDeliveryNote, getSalesOrder, type SalesOrder } from '../../api/sales'
import Button from '../../shared/components/Button'
import Card from '../../shared/components/Card'
import { ErrorAlert, InfoAlert } from '../../shared/components/EmptyState'
import Field from '../../shared/components/Field'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge, { statusLabel, statusTone } from '../../shared/components/StatusBadge'
import Toolbar from '../../shared/components/Toolbar'
import { fmtDate, fmtMoney, fmtQty, unwrap } from '../../shared/utils/format'

type Props = {
  id: string
  role: string
  onNavigate: (page: string, id?: string) => void
}

export default function SalesOrderDetailPage({ id, role, onNavigate }: Props) {
  const [order, setOrder] = useState<SalesOrder | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const [showDelivery, setShowDelivery] = useState(false)
  const [recipientName, setRecipientName] = useState('')
  const [signatureRef, setSignatureRef] = useState('')
  const [deliveryNotes, setDeliveryNotes] = useState('')
  const [quantities, setQuantities] = useState<Record<string, string>>({})

  const canDeliver = role === 'sales' || role === 'sales_head' || role === 'admin'

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await getSalesOrder(id)
      setOrder(unwrap<SalesOrder>(res))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load sales order')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  const openDelivery = () => {
    const initial: Record<string, string> = {}
    for (const item of order?.items ?? []) {
      initial[item.id] = String(Math.max(0, item.quantity - item.delivered_quantity))
    }
    setQuantities(initial)
    setShowDelivery(true)
  }

  const handleCreateDelivery = async () => {
    if (!order) return
    const items = (order.items ?? [])
      .map((item) => ({
        sales_order_item_id: item.id,
        quantity_delivered: Number(quantities[item.id] ?? 0),
      }))
      .filter((l) => l.quantity_delivered > 0)

    if (items.length === 0) {
      setError('Enter a delivery quantity on at least one line.')
      return
    }

    setBusy(true)
    setError('')
    try {
      await createDeliveryNote(id, {
        recipient_name: recipientName || null,
        recipient_signature_reference: signatureRef || null,
        notes: deliveryNotes || null,
        items,
      })
      setShowDelivery(false)
      setRecipientName('')
      setSignatureRef('')
      setDeliveryNotes('')
      await load()
      window.alert('Delivery note created.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create delivery note')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <div className="ui-state">Loading sales order…</div>
  if (error && !order) return <ErrorAlert message={error} onRetry={() => void load()} />
  if (!order) return null

  const delivered = (order.items ?? []).reduce((sum, i) => sum + (i.delivered_quantity ?? 0), 0)
  const ordered = (order.items ?? []).reduce((sum, i) => sum + (i.quantity ?? 0), 0)

  return (
    <>
      <PageHeader
        title={
          <>
            Sales Order <strong>{order.order_number}</strong>
          </>
        }
        subtitle={order.notes || `Sales order for ${order.customer_name || 'customer'}.`}
        crumbs={[
          { label: 'Sales orders', onClick: () => onNavigate('Sales orders') },
          { label: order.order_number },
        ]}
        actions={<StatusBadge label={statusLabel(order.status)} tone={statusTone(order.status)} />}
      />

      <Toolbar
        actions={[
          {
            label: 'Deliver',
            icon: 'arrow',
            onClick: openDelivery,
            disabled: busy || !canDeliver || order.status === 'delivered',
          },
          { label: 'Back', icon: 'arrow', onClick: () => onNavigate('Sales orders') },
        ]}
      />

      {error && <ErrorAlert message={error} />}
      {order.status === 'delivered' && <InfoAlert>This order has been fully delivered.</InfoAlert>}

      <Card title="Order Details" bordered>
        <div className="ui-facts">
          <div className="ui-fact">
            <span>Customer</span>
            <strong>{order.customer_name || '—'}</strong>
          </div>
          <div className="ui-fact">
            <span>Order date</span>
            <strong>{fmtDate(order.order_date)}</strong>
          </div>
          <div className="ui-fact">
            <span>Required date</span>
            <strong>{fmtDate(order.required_date)}</strong>
          </div>
          <div className="ui-fact">
            <span>Ordered quantity</span>
            <strong>{fmtQty(ordered)}</strong>
          </div>
          <div className="ui-fact">
            <span>Delivered quantity</span>
            <strong>{fmtQty(delivered)}</strong>
          </div>
          <div className="ui-fact">
            <span>Total amount</span>
            <strong>{fmtMoney(order.total_amount)}</strong>
          </div>
        </div>
      </Card>

      <Card title="Order Lines" bordered>
        <table className="ui-lines">
          <thead>
            <tr>
              <th>Item</th>
              <th style={{ textAlign: 'right' }}>Ordered</th>
              <th style={{ textAlign: 'right' }}>Reserved</th>
              <th style={{ textAlign: 'right' }}>Delivered</th>
              <th style={{ textAlign: 'right' }}>Outstanding</th>
              <th style={{ textAlign: 'right' }}>Unit Price</th>
              <th style={{ textAlign: 'right' }}>Line Total</th>
            </tr>
          </thead>
          <tbody>
            {(order.items ?? []).map((item) => (
              <tr key={item.id}>
                <td>
                  <strong>{item.item_code}</strong> {item.item_name}
                </td>
                <td className="num">{fmtQty(item.quantity)}</td>
                <td className="num">{fmtQty(item.reserved_quantity)}</td>
                <td className="num">{fmtQty(item.delivered_quantity)}</td>
                <td className="num">{fmtQty(Math.max(0, item.quantity - item.delivered_quantity))}</td>
                <td className="num">{fmtMoney(item.unit_price)}</td>
                <td className="num">{fmtMoney(item.line_total)}</td>
              </tr>
            ))}
            {(order.items ?? []).length === 0 && (
              <tr>
                <td className="table-empty" colSpan={7}>
                  No lines on this order.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {showDelivery && (
        <Card title="Create Delivery Note" subtitle="Enter the quantity being dispatched on each line." bordered>
          <div className="ui-form-grid is-3">
            <Field
              label="Recipient Name"
              value={recipientName}
              placeholder="Who receives the delivery?"
              onChange={(e) => setRecipientName(e.target.value)}
            />
            <Field
              label="Signature Reference"
              value={signatureRef}
              placeholder="Optional reference"
              onChange={(e) => setSignatureRef(e.target.value)}
            />
            <Field
              label="Notes"
              value={deliveryNotes}
              placeholder="Optional notes"
              onChange={(e) => setDeliveryNotes(e.target.value)}
            />
          </div>

          <table className="ui-lines" style={{ marginTop: 16 }}>
            <thead>
              <tr>
                <th>Item</th>
                <th style={{ textAlign: 'right' }}>Outstanding</th>
                <th style={{ width: 180 }}>Delivering Now</th>
              </tr>
            </thead>
            <tbody>
              {(order.items ?? []).map((item) => {
                const outstanding = Math.max(0, item.quantity - item.delivered_quantity)
                return (
                  <tr key={item.id}>
                    <td>
                      <strong>{item.item_code}</strong> {item.item_name}
                    </td>
                    <td className="num">{fmtQty(outstanding)}</td>
                    <td>
                      <input
                        type="number"
                        min="0"
                        max={outstanding}
                        step="0.0001"
                        value={quantities[item.id] ?? ''}
                        onChange={(e) => setQuantities((prev) => ({ ...prev, [item.id]: e.target.value }))}
                        aria-label={`Quantity to deliver for ${item.item_code ?? item.item_name}`}
                      />
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>

          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 16 }}>
            <Button onClick={() => setShowDelivery(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => void handleCreateDelivery()} disabled={busy}>
              {busy ? 'Posting…' : 'Post Delivery'}
            </Button>
          </div>
        </Card>
      )}

      <div className="ui-totals">
        <div className="ui-totals-item">
          <small>Total amount</small>
          {fmtMoney(order.total_amount)}
        </div>
      </div>
    </>
  )
}
