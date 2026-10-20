import { useEffect, useState } from 'react'
import { createProductionReceipt, type ProductionReceiptInput } from '../../api/production'
import { listItems, listWarehouses, type CatalogItem, type Warehouse } from '../../api/master'
import Card from '../../shared/components/Card'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import Field from '../../shared/components/Field'
import PageHeader from '../../shared/components/PageHeader'
import Toolbar from '../../shared/components/Toolbar'
import { fmtQty, unwrap } from '../../shared/utils/format'

type Line = {
  key: number
  item_id: string
  quantity_produced: string
  batch_number: string
  manufacture_date: string
  expiry_date: string
  unit_cost: string
  quality_status: 'accepted' | 'rejected'
}

type Props = {
  onNavigate: (page: string, id?: string) => void
}

let seq = 0
const newLine = (): Line => ({
  key: ++seq,
  item_id: '',
  quantity_produced: '',
  batch_number: '',
  manufacture_date: '',
  expiry_date: '',
  unit_cost: '',
  quality_status: 'accepted',
})

export default function CreateProductionReceiptPage({ onNavigate }: Props) {
  const [warehouses, setWarehouses] = useState<Warehouse[]>([])
  const [items, setItems] = useState<CatalogItem[]>([])
  const [warehouseId, setWarehouseId] = useState('')
  const [notes, setNotes] = useState('')
  const [lines, setLines] = useState<Line[]>([newLine()])
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)

  useEffect(() => {
    void (async () => {
      try {
        const [whs, catalog] = await Promise.all([listWarehouses(), listItems(undefined, 1, 200)])
        setWarehouses(Array.isArray(whs) ? whs : unwrap<Warehouse[]>(whs))
        setItems(unwrap<CatalogItem[]>(catalog))
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load reference data')
      }
    })()
  }, [])

  // Default to the finished-goods warehouse once the list arrives.
  useEffect(() => {
    if (warehouseId || warehouses.length === 0) return
    const finished = warehouses.find((w) => w.warehouse_type === 'finished_goods')
    if (finished) setWarehouseId(finished.id)
  }, [warehouses, warehouseId])

  const updateLine = (key: number, patch: Partial<Line>) => {
    setLines((prev) => prev.map((l) => (l.key === key ? { ...l, ...patch } : l)))
    setDirty(true)
  }

  const canSave =
    warehouseId !== '' &&
    lines.length > 0 &&
    lines.every((l) => l.item_id !== '' && Number(l.quantity_produced) > 0) &&
    !saving

  const totalAccepted = lines
    .filter((l) => l.quality_status === 'accepted')
    .reduce((sum, l) => sum + (Number(l.quantity_produced) || 0), 0)

  const handleSave = async () => {
    setSaving(true)
    setError('')
    const payload: ProductionReceiptInput = {
      warehouse_id: warehouseId,
      notes: notes || null,
      items: lines.map((l) => ({
        item_id: l.item_id,
        quantity_produced: Number(l.quantity_produced),
        batch_number: l.batch_number || null,
        manufacture_date: l.manufacture_date || null,
        expiry_date: l.expiry_date || null,
        unit_cost: l.unit_cost === '' ? null : Number(l.unit_cost),
        quality_status: l.quality_status,
      })),
    }
    try {
      const res = await createProductionReceipt(payload)
      const created = unwrap<{ id: string }>(res)
      onNavigate('Production Receipt Detail', created?.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create production receipt')
      setSaving(false)
    }
  }

  const handleCancel = () => {
    if (dirty && !window.confirm('Discard the unsaved changes?')) return
    onNavigate('Production receipts')
  }

  return (
    <>
      <PageHeader
        title={<>New Production Receipt</>}
        subtitle="Record finished goods so they are added to the finished-goods store."
        crumbs={[
          { label: 'Production receipts', onClick: () => onNavigate('Production receipts') },
          { label: 'New receipt' },
        ]}
      />

      <Toolbar
        actions={[
          { label: 'Save', icon: 'spark', tone: 'success', onClick: () => void handleSave(), disabled: !canSave },
          { label: 'Cancel', icon: 'close', tone: 'danger', onClick: handleCancel },
          { label: 'Back', icon: 'arrow', onClick: () => onNavigate('Production receipts') },
        ]}
      />

      {error && <ErrorAlert message={error} />}

      <Card title="Receipt Details" bordered>
        <div className="ui-form-grid">
          <Field
            as="select"
            label="Warehouse"
            required
            value={warehouseId}
            onChange={(e) => {
              setWarehouseId(e.target.value)
              setDirty(true)
            }}
          >
            <option value="">Select warehouse…</option>
            {warehouses
              .filter((w) => w.is_active)
              .map((w) => (
                <option key={w.id} value={w.id}>
                  {w.code} — {w.name} ({w.warehouse_type.replace(/_/g, ' ')})
                </option>
              ))}
          </Field>

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

      <Card title="Produced Items" subtitle="Only accepted lines add to stock." bordered>
        {lines.length === 0 ? (
          <EmptyState title="No lines yet" description="Add the finished goods produced." icon="production" />
        ) : (
          <table className="ui-lines">
            <thead>
              <tr>
                <th style={{ width: '26%' }}>Item</th>
                <th style={{ width: '11%' }}>Qty</th>
                <th style={{ width: '13%' }}>Batch</th>
                <th style={{ width: '12%' }}>Mfg Date</th>
                <th style={{ width: '12%' }}>Expiry</th>
                <th style={{ width: '10%' }}>Unit Cost</th>
                <th style={{ width: '12%' }}>Quality</th>
                <th style={{ width: 44 }} />
              </tr>
            </thead>
            <tbody>
              {lines.map((line) => (
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
                      value={line.quantity_produced}
                      onChange={(e) => updateLine(line.key, { quantity_produced: e.target.value })}
                      aria-label="Quantity produced"
                    />
                  </td>
                  <td>
                    <input
                      value={line.batch_number}
                      placeholder="Batch"
                      onChange={(e) => updateLine(line.key, { batch_number: e.target.value })}
                      aria-label="Batch number"
                    />
                  </td>
                  <td>
                    <input
                      type="date"
                      value={line.manufacture_date}
                      onChange={(e) => updateLine(line.key, { manufacture_date: e.target.value })}
                      aria-label="Manufacture date"
                    />
                  </td>
                  <td>
                    <input
                      type="date"
                      value={line.expiry_date}
                      onChange={(e) => updateLine(line.key, { expiry_date: e.target.value })}
                      aria-label="Expiry date"
                    />
                  </td>
                  <td>
                    <input
                      type="number"
                      min="0"
                      step="0.01"
                      value={line.unit_cost}
                      placeholder="—"
                      onChange={(e) => updateLine(line.key, { unit_cost: e.target.value })}
                      aria-label="Unit cost"
                    />
                  </td>
                  <td>
                    <select
                      value={line.quality_status}
                      onChange={(e) => updateLine(line.key, { quality_status: e.target.value as Line['quality_status'] })}
                      aria-label="Quality status"
                    >
                      <option value="accepted">Accepted</option>
                      <option value="rejected">Rejected</option>
                    </select>
                  </td>
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
              ))}
            </tbody>
          </table>
        )}

        <button type="button" className="ui-add-line" onClick={() => setLines((prev) => [...prev, newLine()])}>
          + Add line
        </button>
      </Card>

      <div className="ui-totals">
        <div className="ui-totals-item">
          <small>Accepted quantity to be added to stock</small>
          {fmtQty(totalAccepted)}
        </div>
      </div>
    </>
  )
}
