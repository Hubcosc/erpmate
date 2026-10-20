import './ui.css'

type Props = {
  tabs: Array<{ key: string; label: string; count?: number }>
  active: string
  onChange: (key: string) => void
}

/** UI Design Specification 8.6 — Tabs.
 *  Horizontal, equal width inside a bordered, rounded container.
 *  Active: bold text plus a 2px accent underline. */
export default function Tabs({ tabs, active, onChange }: Props) {
  return (
    <div className="ui-tabs" role="tablist">
      {tabs.map((tab) => (
        <button
          key={tab.key}
          type="button"
          role="tab"
          aria-selected={active === tab.key}
          className={`ui-tab ${active === tab.key ? 'is-active' : ''}`}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
          {tab.count !== undefined && ` (${tab.count})`}
        </button>
      ))}
    </div>
  )
}
