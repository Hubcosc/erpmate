import { useEffect, useState } from 'react'
import { listUsers, listRoles, deleteUser, type User, type Role } from '../../../api/users'
import { Icon } from '../../../shared/components/Icon'
import './UsersPage.css'
import Modal from '../../../shared/components/Modal'
import AddUserForm from './AddUserForm'

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [search, setSearch] = useState('')
  const [total, setTotal] = useState(0)
  const [showAddUser, setShowAddUser] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [deletingUser, setDeletingUser] = useState<User | null>(null)
  const [deleting, setDeleting] = useState(false)

  const loadUsers = async (searchValue = '') => {
    setLoading(true)
    setError('')
    try {
      const res = await listUsers({ search: searchValue, page: 1, limit: 100 })
      setUsers(res.data)
      setTotal(res.meta?.total ?? res.data.length)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load users')
    } finally {
      setLoading(false)
    }
  }

  const loadRoles = async () => {
    try {
      const res = await listRoles()
      setRoles(res)
    } catch {
      // roles are optional — page still works without them
    }
  }

  useEffect(() => {
    void loadUsers()
    void loadRoles()
  }, [])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    void loadUsers(search)
  }

  const confirmDelete = async () => {
  if (!deletingUser) return
  setDeleting(true)
  try {
    await deleteUser(deletingUser.id)
    setDeletingUser(null)
    void loadUsers(search)
  } catch (err) {
    alert(err instanceof Error ? err.message : 'Failed to delete user')
  } finally {
    setDeleting(false)
  }
}

  return (
    <div className="users-page">
      <header className="users-header">
        <div>
          <p className="users-eyebrow">Administration</p>
          <h1>Users</h1>
          <p className="users-subtitle">
            {total} {total === 1 ? 'user' : 'users'} in the system
          </p>
        </div>
        <button
          className="users-btn users-btn-primary"
          type="button"
          onClick={() => setShowAddUser(true)}
        >
          <Icon name="settings" size={16} /> Add user
        </button>
      </header>

      <form className="users-toolbar" onSubmit={handleSearch}>
        <label className="users-search">
          <Icon name="search" size={16} />
          <input
            type="search"
            placeholder="Search by name, email, or employee ID"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </label>
        <button className="users-btn" type="submit">Search</button>
        {search && (
          <button
            className="users-btn users-btn-ghost"
            type="button"
            onClick={() => { setSearch(''); void loadUsers('') }}
          >
            Clear
          </button>
        )}
      </form>

      {loading && <p className="users-state">Loading users…</p>}
      {error && <p className="users-state users-state-error">{error}</p>}

      {!loading && !error && users.length === 0 && (
        <p className="users-state">No users found.</p>
      )}

      {!loading && !error && users.length > 0 && (
        <div className="users-table-wrap">
          <table className="users-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Employee ID</th>
                <th>Role</th>
                <th>Department</th>
                <th>Status</th>
                <th>Last login</th>
                <th aria-label="Actions"></th> 
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <div className="users-name-cell">
                      <span className="users-avatar" aria-hidden="true">
                        {initials(u.full_name)}
                      </span>
                      <span>{u.full_name}</span>
                    </div>
                  </td>
                  <td>{u.email}</td>
                  <td>{u.employee_id ?? '—'}</td>
                  <td>{u.role_name}</td>
                  <td>{u.department_name ?? '—'}</td>
                  <td>
                    <span className={`users-pill ${u.is_active ? 'is-active' : 'is-inactive'}`}>
                      {u.is_active ? 'Active' : 'Inactive'}
                    </span>
                  </td>
                  <td>{u.last_login_at ? new Date(u.last_login_at).toLocaleString() : '—'}</td>
                  <td className="users-actions-cell">
                    <div className="users-row-actions">
                      <button
                        className="users-row-btn"
                        type="button"
                        onClick={() => setEditingUser(u)}
                        title="Edit user"
                      >
                        Edit
                      </button>
                      <button
                        className="users-row-btn users-row-btn-danger"
                        type="button"
                        onClick={() => setDeletingUser(u)}
                        title="Delete user"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {/* Add user modal */}
      <Modal
        open={showAddUser}
        title="Add user"
        onClose={() => setShowAddUser(false)}
      >
        <AddUserForm
          onSuccess={() => {
            setShowAddUser(false)
            void loadUsers(search)
          }}
          onCancel={() => setShowAddUser(false)}
        />
      </Modal>

      {/* Edit user modal */}
      <Modal
        open={editingUser !== null}
        title={editingUser ? `Edit user: ${editingUser.full_name}` : 'Edit user'}
        onClose={() => setEditingUser(null)}
      >
        {editingUser && (
          <AddUserForm
            user={editingUser}
            onSuccess={() => {
              setEditingUser(null)
              void loadUsers(search)
            }}
            onCancel={() => setEditingUser(null)}
          />
        )}
      </Modal>

      {/* Delete confirmation modal */}
      <Modal
        open={deletingUser !== null}
        title="Delete user"
        onClose={() => !deleting && setDeletingUser(null)}
      >
        {deletingUser && (
          <div className="user-delete-confirm">
            <p>
              Are you sure you want to delete <strong>{deletingUser.full_name}</strong>?
            </p>
            <p className="user-delete-note">
              This will deactivate the user's account. This action is reversible only by an administrator.
            </p>
            <div className="user-form-actions">
              <button
                type="button"
                className="user-form-btn user-form-btn-ghost"
                onClick={() => setDeletingUser(null)}
                disabled={deleting}
              >
                Cancel
              </button>
              <button
                type="button"
                className="user-form-btn user-form-btn-danger"
                onClick={() => void confirmDelete()}
                disabled={deleting}
              >
                {deleting ? 'Deleting…' : 'Delete user'}
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('') || 'U'
}