import type { ReactNode } from 'react'

export type IconName =
  | 'overview' | 'purchase' | 'materials' | 'inventory' | 'production'
  | 'sales' | 'reports' | 'settings' | 'search' | 'bell' | 'menu'
  | 'arrow' | 'clock' | 'spark' | 'chevron' | 'close' | 'categories'

export function Icon({ name, size = 18 }: { name: IconName; size?: number }) {
  const common = {
    width: size,
    height: size,
    viewBox: '0 0 24 24',
    fill: 'none',
    stroke: 'currentColor',
    strokeWidth: 1.7,
    strokeLinecap: 'round' as const,
    strokeLinejoin: 'round' as const,
    'aria-hidden': true as const,
  }

  const paths: Record<IconName, ReactNode> = {
    overview: <><rect x="3.5" y="3.5" width="7" height="7" rx="1.5"/><rect x="13.5" y="3.5" width="7" height="4" rx="1.5"/><rect x="13.5" y="10.5" width="7" height="10" rx="1.5"/><rect x="3.5" y="13.5" width="7" height="7" rx="1.5"/></>,
    purchase: <><path d="M7 3.5h7l4 4v13H7a2 2 0 0 1-2-2v-13a2 2 0 0 1 2-2Z"/><path d="M14 3.5v5h5M8.5 13h7M8.5 16.5h7"/></>,
    materials: <><path d="m4 7 8-4 8 4-8 4-8-4Z"/><path d="m4 12 8 4 8-4M4 17l8 4 8-4"/></>,
    inventory: <><path d="m3.5 7 8.5-4 8.5 4v10L12 21l-8.5-4V7Z"/><path d="m3.8 7.2 8.2 4 8.2-4M12 11.5V21M7.5 5l8.4 4"/></>,
    production: <><rect x="3.5" y="4" width="17" height="16" rx="2"/><path d="M8 4v5h8V4M8 15h2M14 15h2"/></>,
    sales: <><path d="M4 19V9l5 3V8l5 4V6l6-3v16H4Z"/><path d="M8 16h.01M12 16h.01M16 16h.01"/></>,
    reports: <><path d="M4 19.5h16M6.5 16V10M12 16V5M17.5 16v-4"/><circle cx="6.5" cy="8" r="1"/><circle cx="12" cy="3" r="1"/><circle cx="17.5" cy="10" r="1"/></>,
    settings: <><circle cx="12" cy="12" r="3"/><path d="m19.4 15 .1.1 1.3 1-1.3 2.3-1.6-.6a8 8 0 0 1-1.6.9l-.3 1.7h-2.7l-.3-1.7a8 8 0 0 1-1.6-.9l-1.6.6-1.3-2.3 1.3-1a7 7 0 0 1 0-1.9l-1.3-1 1.3-2.3 1.6.6a8 8 0 0 1 1.6-.9l.3-1.7h2.7l.3 1.7a8 8 0 0 1 1.6.9l1.6-.6 1.3 2.3-1.3 1a7 7 0 0 1-.1 1.8Z"/></>,
    search: <><circle cx="10.8" cy="10.8" r="6.8"/><path d="m16 16 4.5 4.5"/></>,
    bell: <><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9ZM10 21h4"/></>,
    menu: <><path d="M4 6h16M4 12h16M4 18h16"/></>,
    arrow: <><path d="M5 12h14M13 6l6 6-6 6"/></>,
    clock: <><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></>,
    spark: <><path d="m12 3 1.9 5.8L20 11l-6.1 2.2L12 19l-1.9-5.8L4 11l6.1-2.2L12 3Z"/><path d="m19 15 .8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8L19 15Z"/></>,
    chevron: <path d="m9 18 6-6-6-6"/>,
    categories: <><path d="M3.5 5.5h6l1.5 2h9.5v11a1 1 0 0 1-1 1h-16a1 1 0 0 1-1-1V6.5a1 1 0 0 1 1-1Z"/><path d="M3.5 9.5h17"/></>,
    close: <><path d="m6 6 12 12M18 6 6 18"/></>,
  }

  return <svg {...common}>{paths[name]}</svg>
}