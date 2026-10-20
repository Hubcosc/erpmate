import type { ReactNode } from 'react'
import './ui.css'

type Props = {
  title?: ReactNode
  subtitle?: ReactNode
  tools?: ReactNode
  bordered?: boolean
  className?: string
  children: ReactNode
}

/** UI Design Specification 8.4 — Card.
 *  White, 8px radius, shared shadow, 16–24px padding.
 *  A card has a title and optionally a right-aligned toolbar.
 *  Do not nest cards inside cards. */
export default function Card({ title, subtitle, tools, bordered, className = '', children }: Props) {
  return (
    <section className={`ui-card ${className}`.trim()}>
      {(title || tools) && (
        <header className={`ui-card-head ${bordered ? 'is-bordered' : ''}`.trim()}>
          <div>
            {title && <h2 className="ui-card-title">{title}</h2>}
            {subtitle && <p className="ui-card-subtitle">{subtitle}</p>}
          </div>
          {tools && <div className="ui-card-tools">{tools}</div>}
        </header>
      )}
      {children}
    </section>
  )
}
