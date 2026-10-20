import type { ReactNode } from 'react'
import './ui.css'

type Crumb = { label: string; onClick?: () => void }

type Props = {
  title: ReactNode
  subtitle?: ReactNode
  crumbs?: Crumb[]
  actions?: ReactNode
}

/** UI Design Specification 5.3 — breadcrumb under the toolbar, and the
 *  page title block. Greeting style: name in accent color. */
export default function PageHeader({ title, subtitle, crumbs, actions }: Props) {
  return (
    <>
      {crumbs && crumbs.length > 0 && (
        <nav className="ui-breadcrumb" aria-label="Breadcrumb">
          {crumbs.map((crumb, index) => (
            <span key={`${crumb.label}-${index}`} style={{ display: 'contents' }}>
              {index > 0 && <span className="sep">&gt;</span>}
              {crumb.onClick ? (
                <button type="button" onClick={crumb.onClick}>
                  {crumb.label}
                </button>
              ) : (
                <span>{crumb.label}</span>
              )}
            </span>
          ))}
        </nav>
      )}
      <header className="ui-page-head">
        <div>
          <h1 className="ui-page-title">{title}</h1>
          {subtitle && <p className="ui-page-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className="ui-page-actions">{actions}</div>}
      </header>
    </>
  )
}
