import { useEffect, useState } from 'react'
import { downloadDeliveryNotePDF, listDeliveryNotes, type DeliveryNote } from '../../api/sales'
import Button from '../../shared/components/Button'
import DataTable, { type Column } from '../../shared/components/DataTable'
import EmptyState, { ErrorAlert } from '../../shared/components/EmptyState'
import PageHeader from '../../shared/components/PageHeader'
import { fmtDate, unwrapPaged } from '../../shared/utils/format'

type Props = {
  onNavigate: (page: string, id?: string) => void
}

export default function DeliveryNotesListPage({ onNavigate }: Props) {
  const [items, setItems] = useState<DeliveryNote[]>([])
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listDeliveryNotes(1, 100)
      const { items: list, total: count } = unwrapPaged<DeliveryNote>(res)
      setItems(list)
      setTotal(count)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load delivery notes')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [])

  const visible = items.filter((note) => {
    if (!search.trim()) return true
    const q = search.trim().toLowerCase()
    return (
      note.delivery_number?.toLowerCase().includes(q) || note.order_number?.toLowerCase().includes(q)
    )
  })

  const handleDownload = async (note: DeliveryNote) => {
    try {
      const blob = await downloadDeliveryNotePDF(note.id)
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `${note.delivery_number}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to download the delivery note PDF')
    }
  }

  const columns: Column<DeliveryNote>[] = [
    { key: 'no', header: 'Delivery Number', render: (r) => <strong>{r.delivery_number}</strong> },
    { key: 'order', header: 'Sales Order', render: (r) => r.order_number || '—' },
    { key: 'by', header: 'Delivered By', render: (r) => r.delivered_by_name || '—' },
    { key: 'to', header: 'Recipient', render: (r) => r.recipient_name || '—' },
    { key: 'date', header: 'Delivery Date', render: (r) => fmtDate(r.delivery_date) },
    { key: 'lines', header: 'Lines', align: 'right', render: (r) => r.items?.length ?? 0 },
    {
      key: 'pdf',
      header: '',
      render: (r) => (
        <Button size="sm" onClick={() => void handleDownload(r)}>
          PDF
        </Button>
      ),
    },
  ]

  return (
    <>
      <PageHeader
        title={<>Delivery Notes</>}
        subtitle="Dispatch documents for sales orders. Each note can be downloaded as a PDF."
      />

      <div className="ui-table-toolbar">
        <div className="ui-table-filters">
          <label className="ui-table-search">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
            <input
              type="search"
              placeholder="Search by delivery number or order…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
        </div>
        <div className="ui-table-filters">
          <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>
            {loading ? 'Loading…' : `${visible.length} of ${total} note${total === 1 ? '' : 's'}`}
          </span>
          <Button size="sm" onClick={() => void load()}>
            Refresh
          </Button>
        </div>
      </div>

      {error && <ErrorAlert message={error} onRetry={() => void load()} />}

      <div className="ui-card" style={{ padding: 0, overflow: 'hidden' }}>
        {loading ? (
          <div className="ui-state">Loading delivery notes…</div>
        ) : visible.length === 0 ? (
          <EmptyState
            title="No delivery notes found"
            description="Delivery notes are created from a sales order."
            icon="sales"
          />
        ) : (
          <DataTable
            columns={columns}
            rows={visible}
            rowKey={(r) => r.id}
            onRowClick={(r) => onNavigate('Delivery Note Detail', r.id)}
          />
        )}
      </div>
    </>
  )
}
