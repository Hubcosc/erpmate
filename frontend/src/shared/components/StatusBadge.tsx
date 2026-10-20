import './ui.css'

export type BadgeTone = 'positive' | 'warning' | 'negative' | 'accent' | 'neutral' | 'default'

type Props = {
  label: string
  tone?: BadgeTone
}

/** Status pill. Colour means something, but never rely on colour alone —
 *  the label text always states the status (spec 10, accessibility). */
export default function StatusBadge({ label, tone = 'default' }: Props) {
  return (
    <span className={`ui-badge ${tone !== 'default' ? `is-${tone}` : ''}`.trim()}>
      <i />
      {label}
    </span>
  )
}

/** Map a workflow status string to a badge tone. */
export function statusTone(status: string): BadgeTone {
  switch (status) {
    case 'approved':
    case 'received':
    case 'delivered':
    case 'fully_issued':
    case 'accepted':
    case 'in_stock':
    case 'confirmed':
    case 'completed':
      return 'positive'
    case 'suspended':
    case 'pending_pm':
    case 'pending_finance':
    case 'partially_issued':
    case 'partially_delivered':
    case 'partially_received':
    case 'issued':
    case 'low_stock':
    case 'near_expiry':
    case 'pending':
      return 'warning'
    case 'rejected':
    case 'out_of_stock':
    case 'expired':
    case 'cancelled':
    case 'critical':
      return 'negative'
    case 'draft':
      return 'neutral'
    default:
      return 'default'
  }
}

/** Human-readable label for a workflow status. */
export function statusLabel(status: string): string {
  return status
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}
