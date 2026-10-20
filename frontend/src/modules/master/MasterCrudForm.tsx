import { useEffect, useState, type ReactNode } from 'react'

export type FieldType = 'text' | 'email' | 'tel' | 'number' | 'textarea' | 'select' | 'checkbox'

export interface FieldConfig<T = any> {
  key: string
  label: string
  type?: FieldType
  required?: boolean
  placeholder?: string
  /** For select fields: options to pick from */
  options?: { value: string; label: string }[]
  /** For select fields: load options from API */
  loadOptions?: () => Promise<{ value: string; label: string }[]>
  /** Help text below the field */
  hint?: string
  /** Only show in create mode (e.g. password) */
  createOnly?: boolean
  /** Only show in edit mode */
  editOnly?: boolean
  /** Read-only in edit mode (e.g. email) */
  readOnlyOnEdit?: boolean
  /** Grid span: 1 (default) or 2 (full width) */
  span?: 1 | 2
}

interface Props<T> {
  /** Fields to render */
  fields: FieldConfig[]
  /** Existing entity when editing; null/undefined = creating */
  entity?: T | null
  /** Called when user submits */
  onSubmit: (values: Record<string, any>) => Promise<void>
  /** Called when user cancels */
  onCancel: () => void
}

export default function MasterCrudForm<T extends Record<string, any>>({
  fields,
  entity,
  onSubmit,
  onCancel,
}: Props<T>) {
  const isEdit = Boolean(entity)
  const [values, setValues] = useState<Record<string, any>>({})
  const [loadingOptions, setLoadingOptions] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [optionsMap, setOptionsMap] = useState<Record<string, { value: string; label: string }[]>>({})

  // Load options for select fields that have loadOptions
  useEffect(() => {
    const loaders = fields.filter((f) => f.loadOptions)
    if (loaders.length === 0) return
    setLoadingOptions(true)
    void Promise.all(
      loaders.map(async (f) => {
        try {
          const opts = await f.loadOptions!()
          return { key: f.key, opts }
        } catch {
          return { key: f.key, opts: [] }
        }
      }),
    ).then((results) => {
      const map: Record<string, { value: string; label: string }[]> = {}
      results.forEach((r) => (map[r.key] = r.opts))
      setOptionsMap(map)
      setLoadingOptions(false)
    })
  }, [fields])

  // Initialize values from entity (edit) or empty (create)
  useEffect(() => {
    const initial: Record<string, any> = {}
    fields.forEach((f) => {
      const raw = entity ? entity[f.key] : undefined
      if (f.type === 'checkbox') {
        initial[f.key] = Boolean(raw ?? true)
      } else {
        initial[f.key] = raw ?? ''
      }
    })
    setValues(initial)
    setError('')
  }, [entity, fields])

  const update = (key: string, value: any) => {
    setValues((v) => ({ ...v, [key]: value }))
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    // Validate required
    for (const f of fields) {
      if (f.createOnly && isEdit) continue
      if (f.editOnly && !isEdit) continue
      if (f.required && !String(values[f.key] ?? '').trim()) {
        return setError(`${f.label} is required`)
      }
    }

    setSubmitting(true)
    try {
      await onSubmit(values)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed')
    } finally {
      setSubmitting(false)
    }
  }

  const renderField = (f: FieldConfig) => {
    if (f.createOnly && isEdit) return null
    if (f.editOnly && !isEdit) return null

    const value = values[f.key] ?? ''
    const disabled = isEdit && f.readOnlyOnEdit
    const options = f.options ?? optionsMap[f.key] ?? []

    const fieldClass = f.span === 2 ? 'mcf-field mcf-field-wide' : 'mcf-field'

    if (f.type === 'checkbox') {
      return (
        <label className={`${fieldClass} mcf-field-checkbox`} key={f.key}>
          <input
            type="checkbox"
            checked={Boolean(values[f.key])}
            onChange={(e) => update(f.key, e.target.checked)}
          />
          <span>{f.label}</span>
        </label>
      )
    }

    return (
      <label className={fieldClass} key={f.key}>
        <span>
          {f.label} {f.required && <i>*</i>}
        </span>
        {f.type === 'textarea' ? (
          <textarea
            value={value}
            onChange={(e) => update(f.key, e.target.value)}
            placeholder={f.placeholder}
            rows={3}
          />
        ) : f.type === 'select' ? (
          <select
            value={value}
            onChange={(e) => update(f.key, e.target.value)}
            disabled={disabled}
          >
            <option value="">{f.placeholder ?? 'Select…'}</option>
            {options.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        ) : (
          <input
            type={f.type ?? 'text'}
            value={value}
            onChange={(e) => update(f.key, e.target.value)}
            placeholder={f.placeholder}
            disabled={disabled}
          />
        )}
        {f.hint && <small className="mcf-hint">{f.hint}</small>}
      </label>
    )
  }

  return (
    <form className="mcf-form" onSubmit={submit}>
      {error && <div className="mcf-error">{error}</div>}

      <div className="mcf-grid">
        {fields.map(renderField)}
        {loadingOptions && <p className="mcf-loading">Loading options…</p>}
      </div>

      <div className="mcf-actions">
        <button type="button" className="mcf-btn mcf-btn-ghost" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button type="submit" className="mcf-btn mcf-btn-primary" disabled={submitting}>
          {submitting ? 'Saving…' : isEdit ? 'Save changes' : 'Create'}
        </button>
      </div>
    </form>
  )
}