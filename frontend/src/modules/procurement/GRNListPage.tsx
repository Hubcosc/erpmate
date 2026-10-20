import { useEffect, useState } from 'react'
import { listGRNs, type GoodsReceipt } from '../../api/goodsReceipts'
import { Icon } from '../../shared/components/Icon'
import './PurchaseOrdersListPage.css' // Reuse the PO list CSS

type Props = {
  onNavigate: (page: string, id?: string) => void
}

export default function GRNListPage({ onNavigate }: Props) {
  const [grns, setGRNs] = useState<GoodsReceipt[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadData = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await listGRNs({ limit: 50 })
      const list = (res as any).data ? (res as any).data : res
      setGRNs(Array.isArray(list) ? list : [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load receipts')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [])

  return (
    <div className="po-page">
      <header className="po-header">
        <div>
          <p className="po-eyebrow">Procurement</p>
          <h1>Goods Receipts</h1>
          <p className="po-subtitle">Record incoming deliveries against Purchase Orders.</p>
        </div>
        <button className="po-btn po-btn-primary" onClick={() => onNavigate('Create GRN')}>
          <Icon name="settings" size={16} /> Receive Items
        </button>
      </header>

      {loading && <p className="po-state">Loading receipts…</p>}
      {error && <p className="po-state po-state-error">{error}</p>}
      {!loading && !error && grns.length === 0 && (
        <p className="po-state">No Goods Receipts found.</p>
      )}

      {!loading && !error && grns.length > 0 && (
        <div className="po-table-wrap">
          <table className="po-table">
            <thead>
              <tr>
                <th>GRN Number</th>
                <th>PO Number</th>
                <th>Warehouse</th>
                <th>Received By</th>
                <th>Date</th>
                <th>Invoice Ref</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {grns.map((grn) => (
                <tr key={grn.id} onClick={() => onNavigate('GRN Detail', grn.id)} style={{ cursor: 'pointer' }}>
                  <td><strong>{grn.grn_number}</strong></td>
                  <td>{grn.po_number}</td>
                  <td>{grn.warehouse_name}</td>
                  <td>{grn.received_by_name}</td>
                  <td>{new Date(grn.receipt_date).toLocaleDateString()}</td>
                  <td>{grn.supplier_invoice_reference || '—'}</td>
                  <td style={{ textAlign: 'right' }}><Icon name="chevron" size={16} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}