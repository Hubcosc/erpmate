import { useEffect, useState } from 'react'
import { createUser, updateUser, listRoles, type Role, type User, type CreateUserInput } from '../../../api/users'
import { listDepartments, type Department } from '../../../api/master'

type Props = {
  user?: User | null        
  onSuccess: () => void
  onCancel: () => void
}

export default function AddUserForm({ user, onSuccess, onCancel }: Props) {
  const isEdit = Boolean(user)
  const [roles, setRoles] = useState<Role[]>([])
  const [departments, setDepartments] = useState<Department[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

    const [form, setForm] = useState<CreateUserInput>({
    full_name: user?.full_name ?? '',
    email: user?.email ?? '',
    password: '',                 
    employee_id: user?.employee_id ?? '',
    phone: user?.phone ?? '',
    role_id: user?.role_id ?? '',
    department_id: user?.department_id ?? '',
    })

useEffect(() => {
  void listRoles()
    .then((res) => setRoles(Array.isArray(res) ? res : []))
    .catch(() => setRoles([]))
  void listDepartments()
    .then((res) => {
      const list = Array.isArray(res) ? res : (res as any)?.data ?? []
      setDepartments(list)
    })
    .catch(() => setDepartments([]))
}, [])

  const update = <K extends keyof CreateUserInput>(key: K, value: CreateUserInput[K]) => {
    setForm((f) => ({ ...f, [key]: value }))
  }

  const submit = async (e: React.FormEvent) => {
  e.preventDefault()
  setError('')

  if (!form.full_name.trim()) return setError('Full name is required')
  if (!form.email.trim()) return setError('Email is required')
  if (!form.role_id) return setError('Role is required')

  if (!isEdit) {
    if (!form.password || form.password.length < 6) {
      return setError('Password must be at least 6 characters')
    }
  }

  setSubmitting(true)
  try {
    if (isEdit && user) {
      // EDIT — send only changed fields (backend accepts partial updates)
      await updateUser(user.id, {
        full_name: form.full_name.trim(),
        employee_id: form.employee_id?.trim() || undefined,
        phone: form.phone?.trim() || undefined,
        role_id: form.role_id,
        department_id: form.department_id || undefined,
      })
    } else {
      // CREATE
      const payload: CreateUserInput = {
        full_name: form.full_name.trim(),
        email: form.email.trim(),
        password: form.password,
        role_id: form.role_id,
      }
      if (form.employee_id?.trim()) payload.employee_id = form.employee_id.trim()
      if (form.phone?.trim()) payload.phone = form.phone.trim()
      if (form.department_id) payload.department_id = form.department_id

      await createUser(payload)
    }
    onSuccess()
  } catch (err) {
    setError(err instanceof Error ? err.message : `Failed to ${isEdit ? 'update' : 'create'} user`)
  } finally {
    setSubmitting(false)
  }
}

  return (
    <form className="user-form" onSubmit={submit}>
      {error && <div className="user-form-error">{error}</div>}

      <div className="user-form-grid">
        <label className="user-form-field">
          <span>Full name <i>*</i></span>
          <input
            type="text"
            value={form.full_name}
            onChange={(e) => update('full_name', e.target.value)}
            placeholder="e.g. Abebe Bekele"
            required
          />
        </label>

        <label className="user-form-field">
          <span>Email <i>*</i></span>
            <input
            type="email"
            value={form.email}
            onChange={(e) => update('email', e.target.value)}
            placeholder="name@company.com"
            disabled={isEdit}
            required
            />
        </label>

        {!isEdit && (
        <label className="user-form-field">
            <span>Password <i>*</i></span>
            <input
            type="password"
            value={form.password}
            onChange={(e) => update('password', e.target.value)}
            placeholder="At least 6 characters"
            required
            />
        </label>
        )}

        <label className="user-form-field">
          <span>Employee ID</span>
          <input
            type="text"
            value={form.employee_id}
            onChange={(e) => update('employee_id', e.target.value)}
            placeholder="e.g. EMP-001"
          />
        </label>

        <label className="user-form-field">
          <span>Phone</span>
          <input
            type="tel"
            value={form.phone}
            onChange={(e) => update('phone', e.target.value)}
            placeholder="+251..."
          />
        </label>

        <label className="user-form-field">
          <span>Role <i>*</i></span>
          <select value={form.role_id} onChange={(e) => update('role_id', e.target.value)} required>
            <option value="">Select a role…</option>
            {Array.isArray(roles) && roles.map((r) => (
            <option key={r.id} value={r.id}>{r.name}</option>
            ))}
          </select>
        </label>

        <label className="user-form-field user-form-field-wide">
          <span>Department</span>
          <select value={form.department_id} onChange={(e) => update('department_id', e.target.value)}>
            <option value="">No department</option>
            {departments.map((d) => (
              <option key={d.id} value={d.id}>{d.name}</option>
            ))}
          </select>
        </label>
      </div>

      <div className="user-form-actions">
        <button type="button" className="user-form-btn user-form-btn-ghost" onClick={onCancel} disabled={submitting}>
          Cancel
        </button>
        <button type="submit" className="user-form-btn user-form-btn-primary" disabled={submitting}>
        {submitting ? (isEdit ? 'Saving…' : 'Creating…') : (isEdit ? 'Save changes' : 'Create user')}
        </button>
      </div>
    </form>
  )
}