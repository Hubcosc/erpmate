import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'
import './ui.css'

type Props = {
  title: string
  description?: string
  icon?: IconName
  action?: ReactNode
}

/** Empty table / empty page state: one centred line of muted text and an action. */
export default function EmptyState({ title, description, icon = 'overview', action }: Props) {
  return (
    <div className="ui-state">
      <span className="ui-state-mark">
        <Icon name={icon} size={20} />
      </span>
      <strong>{title}</strong>
      {description && <p>{description}</p>}
      {action}
    </div>
  )
}

/** Inline error banner. */
export function ErrorAlert({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return (
    <div className="ui-alert is-error" role="alert">
      <Icon name="close" size={15} />
      <span style={{ flex: 1 }}>{message}</span>
      {onRetry && (
        <button type="button" className="ui-btn is-sm" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  )
}

/** Inline info banner. */
export function InfoAlert({ children }: { children: ReactNode }) {
  return (
    <div className="ui-alert is-info">
      <Icon name="spark" size={15} />
      <span>{children}</span>
    </div>
  )
}
