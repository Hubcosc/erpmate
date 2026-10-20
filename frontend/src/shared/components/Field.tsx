import type { ReactNode, SelectHTMLAttributes, InputHTMLAttributes, TextareaHTMLAttributes } from 'react'
import './ui.css'

type BaseProps = {
  label?: string
  required?: boolean
  error?: string
  hint?: string
  wide?: boolean
  readOnly?: boolean
  className?: string
}

type InputProps = BaseProps &
  Omit<InputHTMLAttributes<HTMLInputElement>, 'className'> & { as?: never }
type SelectProps = BaseProps &
  Omit<SelectHTMLAttributes<HTMLSelectElement>, 'className'> & { as: 'select'; options?: Array<{ value: string; label: string }> }
type TextareaProps = BaseProps &
  Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className'> & { as: 'textarea' }

export type FieldProps = InputProps | SelectProps | TextareaProps

/** UI Design Specification 8.1 — Input (filled field).
 *  40px high, --color-field-bg, no border except a 2px bottom line,
 *  4px radius on the top corners only, 12px horizontal padding, 13px text.
 *  Focus: the bottom line turns accent color — no glow, no outer border.
 *  Error: bottom line in danger color and an 11px message below. */
export default function Field(props: FieldProps) {
  const {
    label,
    required,
    error,
    hint,
    wide,
    readOnly,
    className = '',
    as,
    ...rest
  } = props

  const isSelect = as === 'select'
  const isTextarea = as === 'textarea'

  const control = isSelect ? (
    <select {...(rest as SelectHTMLAttributes<HTMLSelectElement>)} />
  ) : isTextarea ? (
    <textarea {...(rest as TextareaHTMLAttributes<HTMLTextAreaElement>)} />
  ) : (
    <input {...(rest as InputHTMLAttributes<HTMLInputElement>)} readOnly={readOnly} />
  )

  return (
    <label
      className={[
        'ui-field',
        isSelect ? 'has-select' : '',
        error ? 'is-error' : '',
        readOnly ? 'is-readonly' : '',
        wide ? 'is-wide' : '',
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {label && (
        <span className="ui-field-label">
          {label}
          {required && <span className="req">*</span>}
        </span>
      )}
      <span className="ui-field-control">{control}</span>
      {error && <span className="ui-field-error">{error}</span>}
      {!error && hint && <span className="ui-field-hint">{hint}</span>}
    </label>
  )
}

/** Field wrapper for a lookup (search icon on the right). */
export function LookupField({
  label,
  required,
  error,
  hint,
  wide,
  children,
  onLookup,
}: BaseProps & { children: ReactNode; onLookup?: () => void }) {
  return (
    <label className={['ui-field', 'has-lookup', error ? 'is-error' : '', wide ? 'is-wide' : ''].filter(Boolean).join(' ')}>
      {label && (
        <span className="ui-field-label">
          {label}
          {required && <span className="req">*</span>}
        </span>
      )}
      <span className="ui-field-control">
        {children}
        <button type="button" className="ui-field-lookup" onClick={onLookup} aria-label={`Search ${label ?? 'record'}`}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
        </button>
      </span>
      {error && <span className="ui-field-error">{error}</span>}
      {!error && hint && <span className="ui-field-hint">{hint}</span>}
    </label>
  )
}
