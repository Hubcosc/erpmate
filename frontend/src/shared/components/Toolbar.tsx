import type { ReactNode } from 'react'
import { Icon, type IconName } from './Icon'
import './ui.css'

export type ToolbarAction = {
  label: string
  icon?: IconName
  onClick: () => void
  disabled?: boolean
  tone?: 'success' | 'danger'
}

type Props = {
  actions: ToolbarAction[]
  recordNav?: ReactNode
}

/** UI Design Specification 5.3 — Page header toolbar on form screens.
 *  A row of icon buttons with a small label under each icon.
 *  Save uses the success color, Cancel uses the danger color.
 *  Only actions that make sense for the current state are enabled. */
export default function Toolbar({ actions, recordNav }: Props) {
  return (
    <div className="ui-toolbar">
      {actions.map((action) => (
        <button
          key={action.label}
          type="button"
          className={`ui-toolbar-action ${action.tone ? `is-${action.tone}` : ''}`.trim()}
          onClick={action.onClick}
          disabled={action.disabled}
        >
          {action.icon ? <Icon name={action.icon} size={18} /> : <span style={{ height: 18 }} />}
          <span>{action.label}</span>
        </button>
      ))}
      {recordNav && recordNav}
    </div>
  )
}
