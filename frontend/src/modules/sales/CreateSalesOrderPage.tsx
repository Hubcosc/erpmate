import { useEffect, useMemo, useState } from 'react'
import { createSalesOrder, type SalesOrderInput } from '../../api/sales'
import { listCustomers, listItems, type CatalogItem, type Customer } from '../../api/master'
import { inventoryApi } from '../../api/inventoryClient'
import Card from '../../shared/components/Card'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import Field from '../../shared/components/Field'
import PageHeader from '../../shared/components/PageHeader'
import Toolbar from '../../shared/components/Toolbar'
import { fmtMoney, fmtQty, unwrap } from '../../shared/utils/format'

type Line = { key: number; item_id: string; quantity: string; unit_price: string }

type Props = {
  onNavigate: (page: string, id?: string) => void
}

let seq = 0
const newLine = (): Line => ({ key: ++seq, item_id: '', quantity: '', unit_price: '' })

export default function CreateSalesOrderPage({ onNavigate }: Props) {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [items, setItems] = useState<CatalogItem[]>([])
  const [customerId, setCustomerId] = useState('')
  const [requiredDate, setRequiredDate] = useState('')
  const [notes, setNotes] = useState('')
  const [lines, setLines] = useState<Line[]>([newLine()])
  const [available, setAvailable] = useState<Record<string, number>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)

  useEffect(() => {
    void (async () => {
      try {
        const [cust, catalog] = await Promise.all([listCustomers(1, 200), listItems(undefined, 1, 200)])
        setCustomers(unwrap<Customer[]>(cust))
        setItems(unwrap<CatalogItem[]>(catalog))
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load reference data')
      }
    })()
  }, [])

  useEffect(() => {
    const ids = Array.from(new Set(lines.map((l) => l.item_id).filter(Boolean)))
    if (ids.length === 0) return
    void (async () => {
      const entries = await Promise.all(
        ids.map(async (id) => {
          try {
            const stock = await inventoryApi.getItemStock(id)
            return [id, stock.total_available] as const
          } catch {
            return [id, Number.NaN] as const
          }
        }),
      )
      setAvailable((prev) => {
        const next = { ...prev }
        for (const [id, qty] of entries) next[id] = qty
        return next
      })
    })()
  }, [lines])

  const itemById = useMemo(() => new Map(items.map((i) => [i.id, i])), [items])

  const updateLine = (key: number, patch: Partial<Line>) => {
    setLines((prev) => prev.map((l) => (l.key === key ? { ...l, ...patch } : l)))
    setDirty(true)
  }

  const total = lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (Number(l.unit_price) || 0), 0)

  const canSave =
    customerId !== '' &&
    lines.length > 0 &&
    lines.every((l) => l.item_id !== '' && Number(l.quantity) > 0 && Number(l.unit_price) >= 0) &&
    !saving

  const handleSave = async () => {
    setSaving(true)
    setError('')
    const payload: SalesOrderInput = {
      customer_id: customerId,
      required_date: requiredDate || null,
      notes: notes || null,
      items: lines.map((l) => ({
        item_id: l.item_id,
        quantity: Number(l.quantity),
        unit_price: Number(l.unit_price),
      })),
    }
    try {
      const res = await createSalesOrder(payload)
      const created = unwrap<{ id: string }>(res)
      onNavigate('Sales Order Detail', created?.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create sales order')
      setSaving(false)
    }
  }

  const handleCancel = () => {
    if (dirty && !window.confirm('Discard the unsaved changes?')) return
    onNavigate('Sales orders')
  }

  return (
    <>
      <PageHeader
        title={<>New Sales Order</>}
        subtitle="Creating an order reserves the finished goods from stock."
        crumbs={[
          { label: 'Sales orders', onClick: () => onNavigate('Sales orders') },
          { label: 'New order' },
        ]}
      />

      <Toolbar
        actions={[
          { label: 'Save', icon: 'spark', tone: 'success', onClick: () => void handleSave(), disabled: !canSave },
          { label: 'Cancel', icon: 'close', tone: 'danger', onClick: handleCancel },
          { label: 'Back', icon: 'arrow', onClick: () => onNavigate('Sales orders') },
        ]}
      />

      {error && <ErrorAlert message={error} />}

      <Card title="Order Details" bordered>
        <div className="ui-form-grid is-3">
          <Field
            as="select"
            label="Customer"
            required
            value={customerId}
            onChange={(e) => {
              setCustomerId(e.target.value)
              setDirty(true)
            }}
          >
            <option value="">Select customer…</option>
            {customers
              .filter((c) => c.is_active)
              .map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
          </Field>

          <Field
            label="Required Date"
            type="date"
            value={requiredDate}
            onChange={(e) => {
              setRequiredDate(e.target.value)
              setDirty(true)
            }}
          />

          <Field
            label="Notes"
            value={notes}
            placeholder="Optional notes"
            onChange={(e) => {
              setNotes(e.target.value)
              setDirty(true)
            }}
          />
        </div>
      </Card>

      <Card title="Order Lines" subtitle="Quantities are reserved when the order is saved." bordered>
        {lines.length === 0 ? (
          <EmptyState title="No lines yet" description="Add the finished goods being sold." icon="sales" />
        ) : (
          <table className="ui-lines">
            <thead>
              <tr>
                <th style={{ width: '40%' }}>Item</th>
                <th style={{ width: '16%' }}>Quantity</th>
                <th style={{ width: '16%' }}>Available</th>
                <th style={{ width: '18%' }}>Unit Price</th>
                <th style={{ width: '10%' }}>Amount</th>
                <th style={{ width: 44 }} />
              </tr>
            </thead>
            <tbody>
              {lines.map((line) => {
                const avail = available[line.item_id]
                const item = itemById.get(line.item_id)
                const short = line.item_id !== '' && Number.isFinite(avail) && Number(line.quantity) > avail
                return (
                  <tr key={line.key}>
                    <td>
                      <select
                        value={line.item_id}
                        onChange={(e) => updateLine(line.key, { item_id: e.target.value })}
                        aria-label="Item"
                      >
                        <option value="">Select item…</option>
                        {items
                          .filter((i) => i.is_active)
                          .map((i) => (
                            <option key={i.id} value={i.id}>
                              {i.code} — {i.name}
                            </option>
                          ))}
                      </select>
                    </td>
                    <td>
                      <input
                        type="number"
                        min="0"
                        step="0.0001"
                        value={line.quantity}
                        onChange={(e) => updateLine(line.key, { quantity: e.target.value })}
                        aria-label="Quantity"
                      />
                    </td>
                    <td style={{ fontSize: 12, color: short ? 'var(--color-danger)' : 'var(--color-text-muted)' }}>
                      {line.item_id === ''
                        ? '—'
                        : Number.isFinite(avail)
                          ? `${fmtQty(avail)}${item ? ` ${item.unit_of_measure}` : ''}`
                          : 'Checking…'}
                      {short && ' — insufficient'}
                    </td>
                    <td>
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={line.unit_price}
                        onChange={(e) => updateLine(line.key, { unit_price: e.target.value })}
                        aria-label="Unit price"
                      />
                    </td>
                    <td className="num">{fmtMoney((Number(line.quantity) || 0) * (Number(line.unit_price) || 0))}</td>
                    <td>
                      <button
                        type="button"
                        className="ui-line-remove"
                        onClick={() => {
                          setLines((prev) => prev.filter((l) => l.key !== line.key))
                          setDirty(true)
                        }}
                        aria-label="Remove line"
                      >
                        ×
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}

        <button type="button" className="ui-add-line" onClick={() => setLines((prev) => [...prev, newLine()])}>
          + Add line
        </button>
      </Card>

      <div className="ui-totals">
        <div className="ui-totals-item">
          <small>Total amount</small>
          {fmtMoney(total)}
        </div>
      </div>
    </>
  )
}
