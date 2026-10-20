import type { ButtonHTMLAttributes, ReactNode } from 'react'
import './ui.css'

type Variant = 'default' | 'primary' | 'module' | 'danger' | 'success'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant
  size?: 'md' | 'sm'
  moduleColor?: string
  children: ReactNode
}

/** UI Design Specification 8.2 — Buttons.
 *  Primary: accent fill, white 12px/600 text, 4px radius.
 *  Module action: same shape, filled with the module color.
 *  Rules: one primary button per section. Disabled uses 50% opacity. */
export default function Button({
  variant = 'default',
  size = 'md',
  moduleColor,
  className = '',
  style,
  children,
  ...rest
}: Props) {
  const classes = [
    'ui-btn',
    variant !== 'default' ? `is-${variant}` : '',
    size === 'sm' ? 'is-sm' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <button
      type="button"
      className={classes}
      style={variant === 'module' && moduleColor ? { ...style, ['--module-color' as string]: moduleColor } : style}
      {...rest}
    >
      {children}
    </button>
  )
}
