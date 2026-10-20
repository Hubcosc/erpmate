import { useEffect, useMemo, useState } from 'react'
import {
  createAdminRecord,
  getAdminRecord,
  updateAdminRecord,
  type AdminField,
  type AdminLookups,
  type AdminModel,
  type AdminOption,
  type AdminRow,
} from '../../api/admin'
import Button from '../../shared/components/Button'
import Field from '../../shared/components/Field'
import { ErrorAlert } from '../../shared/components/EmptyState'

type Props = {
  model: AdminModel
  lookups: AdminLookups | null
  recordId: string | null
  onCancel: () => void
  onSaved: () => void
}

export default function AdminRecordForm({ model, lookups, recordId, onCancel, onSaved }: Props) {
  const isNew = recordId === null
  const [values, setValues] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(!isNew)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (isNew) {
      // A new record starts from the defaults the form declares.
      const initial: Record<string, unknown> = {}
      for (const field of model.fields) {
        if (field.type === 'boolean') initial[field.name] = false
        else initial[field.name] = ''
      }
      setValues(initial)
      setLoading(false)
      return
    }
    void (async () => {
      setLoading(true)
      setError('')
      try {
        const res = await getAdminRecord(model.key, recordId)
        setValues((res.data ?? {}) as AdminRow)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Could not load the record')
      } finally {
        setLoading(false)
      }
    })()
  }, [model, recordId, isNew])

  const setValue = (name: string, value: unknown) => {
    setValues((prev) => ({ ...prev, [name]: value }))
  }

  const optionsFor = (field: AdminField): AdminOption[] => {
    if (field.options?.length) return field.options
    if (field.lookup && lookups) {
      const list = (lookups as unknown as Record<string, AdminOption[]>)[field.lookup]
      if (list?.length) return list
    }
    return []
  }

  const missing = useMemo(
    () => model.fields.filter((f) => f.required && f.type !== 'boolean' && String(values[f.name] ?? '').trim() === ''),
    [model.fields, values],
  )

  const handleSave = async () => {
    if (missing.length > 0) {
      setError(`${missing[0].label} is required.`)
      return
    }
    setSaving(true)
    setError('')

    // Only send the fields the registry declares, so nothing unexpected is
    // written to the database.
    const payload: Record<string, unknown> = {}
    for (const field of model.fields) {
      if (field.read_only) continue
      const value = values[field.name]
      if (value === undefined) continue
      if (typeof value === 'string' && value === '' && field.type !== 'password') {
        payload[field.name] = null
        continue
      }
      if (field.type === 'password' && value === '') continue
      payload[field.name] = value
    }

    try {
      if (isNew) await createAdminRecord(model.key, payload)
      else await updateAdminRecord(model.key, recordId, payload)
      onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save the record')
      setSaving(false)
    }
  }

  if (loading) return <div className="ui-state">Loading the record…</div>

  const title = isNew ? `Add ${model.label.toLowerCase()}` : `Change ${model.label.toLowerCase()}`

  return (
    <>
      <div className="admin-bar">
        <div>
          <h1 className="admin-bar-title">
            <strong>{title}</strong>
          </h1>
          <p className="admin-bar-caption">{model.description}</p>
        </div>
      </div>

      {error && <ErrorAlert message={error} />}

      {!model.editable && (
        <p className="admin-readonly-note">
          {model.label_plural} are created by the workflow, so they are read-only here. Use the module screen to act on
          them.
        </p>
      )}

      <div className="ui-card admin-form">
        <div className="admin-form-grid">
          {model.fields.map((field) => {
            const options = optionsFor(field)
            const wide = field.type === 'textarea' || field.type === 'password'
            const value = values[field.name]

            if (field.type === 'boolean') {
              return (
                <label className={`admin-field-checkbox ${wide ? 'is-wide' : ''}`} key={field.name}>
                  <input
                    type="checkbox"
                    checked={Boolean(value)}
                    disabled={field.read_only || !model.editable}
                    onChange={(e) => setValue(field.name, e.target.checked)}
                  />
                  {field.label}
                </label>
              )
            }

            if (field.type === 'select') {
              return (
                <Field
                  key={field.name}
                  as="select"
                  label={field.label}
                  required={field.required}
                  hint={field.help}
                  wide={wide}
                  value={value === null || value === undefined ? '' : String(value)}
                  disabled={field.read_only || !model.editable}
                  onChange={(e) => setValue(field.name, e.target.value)}
                >
                  <option value="">—</option>
                  {options.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Field>
              )
            }

            if (field.type === 'textarea') {
              return (
                <Field
                  key={field.name}
                  as="textarea"
                  label={field.label}
                  required={field.required}
                  hint={field.help}
                  wide
                  value={value === null || value === undefined ? '' : String(value)}
                  disabled={field.read_only || !model.editable}
                  onChange={(e) => setValue(field.name, e.target.value)}
                />
              )
            }

            if (field.type === 'password') {
              return (
                <Field
                  key={field.name}
                  type="password"
                  label={field.label}
                  hint={field.help}
                  wide
                  value={value === null || value === undefined ? '' : String(value)}
                  disabled={field.read_only || !model.editable}
                  onChange={(e) => setValue(field.name, e.target.value)}
                />
              )
            }

            const inputType =
              field.type === 'number' ? 'number' : field.type === 'date' ? 'date' : field.type === 'datetime' ? 'datetime-local' : 'text'

            return (
              <Field
                key={field.name}
                type={inputType}
                label={field.label}
                required={field.required}
                hint={field.help}
                wide={wide}
                min={field.min}
                step={field.step}
                value={value === null || value === undefined ? '' : String(value)}
                disabled={field.read_only || !model.editable}
                onChange={(e) => setValue(field.name, e.target.value)}
              />
            )
          })}
        </div>

        {model.editable && (
          <div className="admin-form-actions">
            <Button variant="primary" onClick={() => void handleSave()} disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </Button>
            <Button onClick={onCancel}>Cancel</Button>
          </div>
        )}
      </div>
    </>
  )
}
