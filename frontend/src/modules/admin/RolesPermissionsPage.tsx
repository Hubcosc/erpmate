import { useEffect, useState } from 'react'
import { listRoles, type Role } from '../../api/users'
import Card from '../../shared/components/Card'
import { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import StatusBadge from '../../shared/components/StatusBadge'
import { ROLE_CAPABILITIES, ROLE_LABELS, ROLE_MENUS } from '../../shared/constants/roles'
import { unwrap } from '../../shared/utils/format'

/** Roles & Permissions — the roles seeded in the database and what each one can do.
 *  The backend enforces the same role rules on every request. */
export default function RolesPermissionsPage() {
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    void (async () => {
      setLoading(true)
      setError('')
      try {
        const res = await listRoles()
        setRoles(Array.isArray(res) ? res : unwrap<Role[]>(res) ?? [])
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load roles')
      } finally {
        setLoading(false)
      }
    })()
  }, [])

  const ordered = [...roles].sort((a, b) => {
    const order = Object.keys(ROLE_LABELS)
    return order.indexOf(a.name) - order.indexOf(b.name)
  })

  return (
    <>
      <PageHeader
        title={<>Roles &amp; Permissions</>}
        subtitle="What each role can open and do. The backend enforces the same rules on every request."
      />

      {error && <ErrorAlert message={error} />}

      {loading ? (
        <div className="ui-state">Loading roles…</div>
      ) : ordered.length === 0 ? (
        <Card>
          <p style={{ margin: 0, fontSize: 13, color: 'var(--color-text-muted)' }}>No roles found.</p>
        </Card>
      ) : (
        ordered.map((role) => {
          const menus = ROLE_MENUS[role.name] ?? []
          const capabilities = ROLE_CAPABILITIES[role.name] ?? []
          return (
            <Card
              key={role.id}
              title={ROLE_LABELS[role.name] ?? role.name}
              subtitle={role.description ?? undefined}
              bordered
              tools={<StatusBadge label={role.name} tone="accent" />}
            >
              <div className="ui-form-grid is-3">
                <div>
                  <span className="ui-field-label">Role key</span>
                  <p style={{ margin: '2px 0 0', fontSize: 13 }}>{role.name}</p>
                </div>
                <div style={{ gridColumn: 'span 2' }}>
                  <span className="ui-field-label">Capabilities</span>
                  <ul style={{ margin: '2px 0 0', paddingLeft: 18, fontSize: 13, color: 'var(--color-text)', lineHeight: 1.7 }}>
                    {capabilities.length === 0 ? (
                      <li style={{ color: 'var(--color-text-muted)' }}>No specific capabilities recorded.</li>
                    ) : (
                      capabilities.map((cap) => <li key={cap}>{cap}</li>)
                    )}
                  </ul>
                </div>
              </div>

              <span className="ui-field-label" style={{ display: 'block', marginTop: 16 }}>Accessible modules</span>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 8 }}>
                {menus.length === 0 ? (
                  <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>None</span>
                ) : (
                  menus.map((menu) => (
                    <span
                      key={menu}
                      style={{
                        fontSize: 11,
                        fontWeight: 600,
                        color: 'var(--color-accent)',
                        background: 'var(--color-accent-tint)',
                        borderRadius: 99,
                        padding: '3px 9px',
                      }}
                    >
                      {menu}
                    </span>
                  ))
                )}
              </div>
            </Card>
          )
        })
      )}
    </>
  )
}
