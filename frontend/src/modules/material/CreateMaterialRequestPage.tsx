import { useEffect, useMemo, useState } from 'react'
import { createMaterialRequisition, type CreateMRInput } from '../../api/material'
import { listDepartments, listItems, type CatalogItem, type Department } from '../../api/master'
import { inventoryApi } from '../../api/inventoryClient'
import Card from '../../shared/components/Card'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import Field from '../../shared/components/Field'
import PageHeader from '../../shared/components/PageHeader'
import Toolbar from '../../shared/components/Toolbar'
import { fmtQty, unwrap } from '../../shared/utils/format'

type Line = {
  key: number
  item_id: string
  quantity: string
  notes: string
}

type Props = {
  onNavigate: (page: string, id?: string) => void
}

let lineSeq = 0
const newLine = (): Line => ({ key: ++lineSeq, item_id: '', quantity: '', notes: '' })

export default function CreateMaterialRequestPage({ onNavigate }: Props) {
  const [departments, setDepartments] = useState<Department[]>([])
  const [items, setItems] = useState<CatalogItem[]>([])
  const [departmentId, setDepartmentId] = useState('')
  const [requiredDate, setRequiredDate] = useState('')
  const [purpose, setPurpose] = useState('')
  const [lines, setLines] = useState<Line[]>([newLine()])
  const [availability, setAvailability] = useState<Record<string, number>>({})
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)

  useEffect(() => {
    void (async () => {
      try {
        const [depts, catalog] = await Promise.all([listDepartments(), listItems(undefined, 1, 200)])
        setDepartments(Array.isArray(depts) ? depts : unwrap<Department[]>(depts))
        setItems(unwrap<CatalogItem[]>(catalog))
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load reference data')
      }
    })()
  }, [])

  // Live stock availability for the chosen item on each line.
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
      setAvailability((prev) => {
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

  const canSave =
    departmentId !== '' &&
    lines.length > 0 &&
    lines.every((l) => l.item_id !== '' && Number(l.quantity) > 0) &&
    !saving

  const handleSave = async () => {
    setSaving(true)
    setError('')
    const payload: CreateMRInput = {
      department_id: departmentId,
      required_date: requiredDate || null,
      purpose: purpose || null,
      items: lines.map((l) => ({
        item_id: l.item_id,
        quantity: Number(l.quantity),
        notes: l.notes || null,
      })),
    }
    try {
      const res = await createMaterialRequisition(payload)
      const created = unwrap<{ id: string }>(res)
      onNavigate('MR Detail', created?.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create material request')
      setSaving(false)
    }
  }

  const handleCancel = () => {
    if (dirty && !window.confirm('Discard the unsaved changes?')) return
    onNavigate('Material requests')
  }

  return (
    <>
      <PageHeader
        title={<>New Material Request</>}
        subtitle="Request raw material from the store. The store keeper issues against this request."
        crumbs={[
          { label: 'Material requests', onClick: () => onNavigate('Material requests') },
          { label: 'New request' },
        ]}
      />

      <Toolbar
        actions={[
          { label: 'Save', icon: 'spark', tone: 'success', onClick: () => void handleSave(), disabled: !canSave },
          { label: 'Cancel', icon: 'close', tone: 'danger', onClick: handleCancel },
          { label: 'Back', icon: 'arrow', onClick: () => onNavigate('Material requests') },
        ]}
      />

      {error && <ErrorAlert message={error} />}

      <Card title="Request Details" bordered>
        <div className="ui-form-grid is-3">
          <Field
            as="select"
            label="Department"
            required
            value={departmentId}
            onChange={(e) => {
              setDepartmentId(e.target.value)
              setDirty(true)
            }}
          >
            <option value="">Select department…</option>
            {departments
              .filter((d) => d.is_active)
              .map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
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
            label="Purpose"
            value={purpose}
            placeholder="What is the material needed for?"
            onChange={(e) => {
              setPurpose(e.target.value)
              setDirty(true)
            }}
          />
        </div>
      </Card>

      <Card title="Requested Items" subtitle="Add one line per material." bordered>
        {lines.length === 0 ? (
          <EmptyState title="No lines yet" description="Add the materials this department needs." icon="materials" />
        ) : (
          <table className="ui-lines">
            <thead>
              <tr>
                <th style={{ width: '34%' }}>Item</th>
                <th style={{ width: '16%' }}>Quantity</th>
                <th style={{ width: '20%' }}>Available</th>
                <th>Notes</th>
                <th style={{ width: 44 }} />
              </tr>
            </thead>
            <tbody>
              {lines.map((line) => {
                const available = availability[line.item_id]
                const item = itemById.get(line.item_id)
                const short =
                  line.item_id !== '' && Number.isFinite(available) && Number(line.quantity) > available
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
                        : Number.isFinite(available)
                          ? `${fmtQty(available)}${item ? ` ${item.unit_of_measure}` : ''}`
                          : 'Checking…'}
                      {short && ' — insufficient'}
                    </td>
                    <td>
                      <input
                        value={line.notes}
                        placeholder="Optional note"
                        onChange={(e) => updateLine(line.key, { notes: e.target.value })}
                        aria-label="Notes"
                      />
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
                )
              })}
            </tbody>
          </table>
        )}

        <button type="button" className="ui-add-line" onClick={() => setLines((prev) => [...prev, newLine()])}>
          + Add line
        </button>
      </Card>
    </>
  )
}
