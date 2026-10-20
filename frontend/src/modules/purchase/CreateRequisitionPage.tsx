import { useEffect, useRef, useState, type FormEvent } from 'react'
import {
  createPR,
  updatePR,
  getPR,
  type CreatePRInput,
  type CreatePRItemInput,
} from '../../api/purchaseRequisitions'
import { listDepartments, listSuppliers, listItems } from '../../api/master'
import './CreateRequisitionPage.css'

type Props = {
  onNavigate: (page: string, id?: string) => void
  editId?: string
}

type FormItem = CreatePRItemInput & { tempId: string; notes?: string }

type NamedOption = { id: string; name: string }
type CatalogItem = { id: string; code: string; name: string; unit_of_measure?: string }

/** Accepts either the raw payload or a { data: payload } envelope. */
function unwrap<T>(res: unknown): T {
  if (res && typeof res === 'object' && 'data' in (res as Record<string, unknown>)) {
    return (res as { data: T }).data
  }
  return res as T
}

function toArray<T>(res: unknown): T[] {
  const value = unwrap<unknown>(res)
  return Array.isArray(value) ? (value as T[]) : []
}

/** Local YYYY-MM-DD (toISOString() would use UTC and can be off by one day). */
function localDate(d = new Date()): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export default function CreateRequisitionPage({ onNavigate, editId }: Props) {
  const isEdit = Boolean(editId)

  // Dropdown data
  const [departments, setDepartments] = useState<NamedOption[]>([])
  const [suppliers, setSuppliers] = useState<NamedOption[]>([])
  const [items, setItems] = useState<CatalogItem[]>([])

  // Form state
  const [departmentId, setDepartmentId] = useState('')
  const [supplierId, setSupplierId] = useState('')
  const [requiredDate, setRequiredDate] = useState('')
  const [originalDate, setOriginalDate] = useState('') // date stored on the requisition being edited
  const [justification, setJustification] = useState('')
  const [lines, setLines] = useState<FormItem[]>([])

  // UI state
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [locked, setLocked] = useState(false) // requisition can no longer be edited
  const errorRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      try {
        const [deptRes, supRes, itemRes, prRes] = await Promise.all([
          listDepartments(),
          listSuppliers(1, 100),
          listItems(undefined, 1, 100),
          editId ? getPR(editId) : Promise.resolve(null),
        ])
        if (cancelled) return

        setDepartments(toArray<NamedOption>(deptRes))
        setSuppliers(toArray<NamedOption>(supRes))
        setItems(toArray<CatalogItem>(itemRes))

        if (prRes) {
          const pr = unwrap<any>(prRes)

          // The backend tells us whether the current user may still edit it.
          if (Array.isArray(pr.available_actions) && !pr.available_actions.includes('edit')) {
            setLocked(true)
            setError('This requisition can no longer be edited.')
            return
          }

          const date = pr.required_date ? String(pr.required_date).slice(0, 10) : ''
          setDepartmentId(pr.department_id ?? '')
          setSupplierId(pr.preferred_supplier_id ?? '')
          setRequiredDate(date)
          setOriginalDate(date)
          setJustification(pr.justification ?? '')
          setLines(
            (pr.items ?? []).map((i: any) => ({
              tempId: i.id || crypto.randomUUID(),
              item_id: i.item_id ?? '',
              description: i.description ?? '',
              quantity: Number(i.quantity),
              unit_of_measure: i.unit_of_measure ?? '',
              estimated_unit_price: i.estimated_unit_price ?? undefined,
              notes: i.notes ?? '',
            })),
          )
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load form data')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [editId])

  // Bring the error banner into view (the form can be long).
  useEffect(() => {
    if (error) errorRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [error])

  // ---- line helpers (functional updates so several patches never overwrite each other) ----
  const addLine = () =>
    setLines((prev) => [
      ...prev,
      { tempId: crypto.randomUUID(), item_id: '', description: '', quantity: 1, unit_of_measure: '', estimated_unit_price: undefined, notes: '' },
    ])

  const removeLine = (tempId: string) => setLines((prev) => prev.filter((l) => l.tempId !== tempId))

  const patchLine = (tempId: string, patch: Partial<FormItem>) =>
    setLines((prev) => prev.map((l) => (l.tempId === tempId ? { ...l, ...patch } : l)))

  const handleItemSelect = (tempId: string, itemId: string) => {
    const selected = items.find((i) => i.id === itemId)
    if (selected) {
      patchLine(tempId, {
        item_id: itemId,
        unit_of_measure: selected.unit_of_measure ?? '',
        description: selected.name,
      })
    } else {
      patchLine(tempId, { item_id: '' })
    }
  }

  const totalEstimated = lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (l.estimated_unit_price ?? 0), 0)

  // ---- submit ----
  const validate = (): string => {
    if (!departmentId) return 'Please select a department'
    if (lines.length === 0) return 'Please add at least one item'
    if (requiredDate && requiredDate < localDate() && requiredDate !== originalDate) {
      return 'Required date cannot be in the past'
    }
    for (let i = 0; i < lines.length; i++) {
      const l = lines[i]
      const n = i + 1
      if (!l.item_id && !(l.description ?? '').trim()) return `Line ${n}: select an item or enter a description`
      if (!(Number(l.quantity) > 0)) return `Line ${n}: quantity must be greater than 0`
      if (l.estimated_unit_price !== undefined && l.estimated_unit_price < 0) return `Line ${n}: price cannot be negative`
    }
    return ''
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (submitting || locked) return
    setError('')

    const problem = validate()
    if (problem) {
      setError(problem)
      return
    }

    setSubmitting(true)
    try {
      const payload: CreatePRInput = {
        department_id: departmentId,
        preferred_supplier_id: supplierId || undefined,
        required_date: requiredDate || undefined,
        justification: justification.trim() || undefined,
        items: lines.map(({ tempId: _tempId, ...rest }) => ({
          ...rest,
          item_id: rest.item_id || undefined,
          description: rest.description?.trim() || undefined,
          unit_of_measure: rest.unit_of_measure?.trim() || undefined,
          notes: rest.notes?.trim() || undefined,
          quantity: Number(rest.quantity),
          estimated_unit_price: rest.estimated_unit_price ?? undefined,
        })),
      }

      if (isEdit && editId) {
        await updatePR(editId, payload)
        onNavigate('PR Detail', editId)
      } else {
        const res = await createPR(payload)
        onNavigate('PR Detail', unwrap<{ id: string }>(res).id)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to ${isEdit ? 'update' : 'create'} requisition`)
    } finally {
      setSubmitting(false)
    }
  }

  const cancel = () => onNavigate(isEdit ? 'PR Detail' : 'Purchase requisitions', editId)

  if (loading) return <div className="pr-page"><p className="pr-state">Loading form data...</p></div>

  // Date picker: allow today onward, but keep an existing (older) date selectable when editing.
  const today = localDate()
  const minDate = originalDate && originalDate < today ? originalDate : today

  return (
    <div className="pr-page">
      <header className="pr-header">
        <div>
          <p className="pr-eyebrow">Purchase Requisitions</p>
          <h1>{isEdit ? 'Edit Requisition' : 'New Requisition'}</h1>
          <p className="pr-subtitle">
            {isEdit ? 'Update the details of your requisition.' : 'Create a new request for procurement.'}
          </p>
        </div>
        <button className="pr-btn pr-btn-ghost" type="button" onClick={cancel}>
          {locked ? 'Back' : 'Cancel'}
        </button>
      </header>

      {error && <div className="pr-error" role="alert" ref={errorRef}>{error}</div>}

      {!locked && (
        <form onSubmit={submit} className="pr-form" noValidate>
          <section className="pr-card">
            <h2 className="pr-card-title">Request Details</h2>
            <div className="pr-grid">
              <label className="pr-field">
                <span>Department <i>*</i></span>
                <select value={departmentId} onChange={(e) => setDepartmentId(e.target.value)} required>
                  <option value="">Select Department...</option>
                  {departments.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
                </select>
              </label>

              <label className="pr-field">
                <span>Preferred Supplier (Optional)</span>
                <select value={supplierId} onChange={(e) => setSupplierId(e.target.value)}>
                  <option value="">None</option>
                  {suppliers.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
                </select>
              </label>

              <label className="pr-field">
                <span>Required Date</span>
                <input type="date" value={requiredDate} onChange={(e) => setRequiredDate(e.target.value)} min={minDate} />
              </label>

              <label className="pr-field pr-field-wide">
                <span>Justification / Purpose</span>
                <textarea
                  value={justification}
                  onChange={(e) => setJustification(e.target.value)}
                  rows={2}
                  maxLength={2000}
                  placeholder="Why is this needed?"
                />
              </label>
            </div>
          </section>

          <section className="pr-card">
            <div className="pr-card-header">
              <h2 className="pr-card-title">Line Items</h2>
              <button type="button" className="pr-btn pr-btn-small" onClick={addLine}>
                + Add Line
              </button>
            </div>

            {lines.length === 0 ? (
              <p className="pr-empty-lines">No items added yet. Click "+ Add Line" to start.</p>
            ) : (
              <div className="pr-line-table-wrap">
                <table className="pr-line-table">
                  <thead>
                    <tr>
                      <th style={{ width: '30%' }}>Item</th>
                      <th style={{ width: '25%' }}>Description</th>
                      <th style={{ width: '10%' }}>Qty</th>
                      <th style={{ width: '15%' }}>Unit</th>
                      <th style={{ width: '15%' }}>Est. Price</th>
                      <th style={{ width: '5%' }}></th>
                    </tr>
                  </thead>
                  <tbody>
                    {lines.map((line, index) => (
                      <tr key={line.tempId}>
                        <td>
                          <select
                            aria-label={`Item for line ${index + 1}`}
                            value={line.item_id || ''}
                            onChange={(e) => handleItemSelect(line.tempId, e.target.value)}
                          >
                            <option value="">Select item...</option>
                            {items.map((i) => <option key={i.id} value={i.id}>{i.code} - {i.name}</option>)}
                          </select>
                        </td>
                        <td>
                          <input
                            type="text"
                            aria-label={`Description for line ${index + 1}`}
                            value={line.description || ''}
                            maxLength={500}
                            onChange={(e) => patchLine(line.tempId, { description: e.target.value })}
                            placeholder="Description"
                          />
                        </td>
                        <td>
                          <input
                            type="number"
                            aria-label={`Quantity for line ${index + 1}`}
                            min="0.01"
                            step="0.01"
                            value={line.quantity}
                            onChange={(e) => patchLine(line.tempId, { quantity: Number(e.target.value) })}
                          />
                        </td>
                        <td>
                          <input
                            type="text"
                            aria-label={`Unit for line ${index + 1}`}
                            value={line.unit_of_measure || ''}
                            maxLength={50}
                            onChange={(e) => patchLine(line.tempId, { unit_of_measure: e.target.value })}
                            placeholder="e.g. kg"
                          />
                        </td>
                        <td>
                          <input
                            type="number"
                            aria-label={`Estimated price for line ${index + 1}`}
                            min="0"
                            step="0.01"
                            value={line.estimated_unit_price ?? ''}
                            onChange={(e) =>
                              patchLine(line.tempId, {
                                estimated_unit_price: e.target.value === '' ? undefined : Number(e.target.value),
                              })
                            }
                            placeholder="0.00"
                          />
                        </td>
                        <td>
                          <button
                            type="button"
                            className="pr-row-btn-danger"
                            aria-label={`Remove line ${index + 1}`}
                            onClick={() => removeLine(line.tempId)}
                          >
                            ✕
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <div className="pr-form-footer">
            <div className="pr-total">
              <span>Estimated Total:</span>
              <strong>{totalEstimated.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</strong>
            </div>
            <button type="submit" className="pr-btn pr-btn-primary" disabled={submitting}>
              {submitting ? 'Saving...' : isEdit ? 'Save Changes' : 'Create Requisition'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}