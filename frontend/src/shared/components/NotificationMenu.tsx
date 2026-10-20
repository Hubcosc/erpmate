import { useEffect, useState } from 'react'
import {
  listNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification,
} from '../../api/notifications'
import { Icon } from './Icon'

export function NotificationMenu() {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<Notification[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const unread = items.filter((item) => !item.read_at).length

  const refresh = async () => {
    setLoading(true)
    setError('')
    try {
      setItems((await listNotifications(1, 10)).data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Notifications could not be loaded.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open) void refresh()
  }, [open])

  const markRead = async (id: string) => {
    try {
      await markNotificationRead(id)
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update notification.')
    }
  }

  const markAll = async () => {
    try {
      await markAllNotificationsRead()
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not update notifications.')
    }
  }

  return (
    <div className="notification-wrap">
      <button
        className={`icon-button notification-button ${open ? 'selected' : ''}`}
        aria-label={`Notifications${unread ? `, ${unread} unread` : ''}`}
        aria-expanded={open}
        onClick={() => setOpen(!open)}
      >
        <Icon name="bell" size={18} />
        {unread > 0 && <span className="notification-count">{unread > 9 ? '9+' : unread}</span>}
      </button>

      {open && (
        <div className="notification-popover">
          <div className="popover-heading">
            <div>
              <strong>Notifications</strong>
              <span>{unread ? `${unread} unread` : 'All caught up'}</span>
            </div>
            {unread > 0 && <button onClick={() => void markAll()}>Mark all read</button>}
          </div>

          {error && <p className="notification-error">{error}</p>}

          {loading ? (
            <div className="popover-loading">Loading updates…</div>
          ) : items.length === 0 ? (
            <div className="popover-empty">
              <span><Icon name="bell" size={19} /></span>
              <p>No notifications yet</p>
              <small>Workflow updates will appear here.</small>
            </div>
          ) : (
            <div className="notification-list">
              {items.map((item) => (
                <button
                  className={`notification-entry ${item.read_at ? 'is-read' : ''}`}
                  key={item.id}
                  onClick={() => !item.read_at && void markRead(item.id)}
                >
                  <span className="notification-entry-icon">
                    <Icon
                      name={
                        item.entity_type === 'purchase_requisition'
                          ? 'purchase'
                          : item.entity_type === 'material_requisition'
                          ? 'materials'
                          : 'bell'
                      }
                      size={15}
                    />
                  </span>
                  <span className="notification-entry-copy">
                    <strong>{item.title}</strong>
                    <small>{item.message}</small>
                    <time>{new Date(item.created_at).toLocaleString()}</time>
                  </span>
                  {!item.read_at && <i className="unread-marker" />}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}