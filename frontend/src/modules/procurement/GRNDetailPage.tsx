import { useEffect, useState } from 'react'
import { getGRN, type GoodsReceipt } from '../../api/goodsReceipts'
import './PurchaseOrderDetailPage.css' // Reuse the PO detail CSS!

type Props = {
  id: string
  onNavigate: (page: string) => void
}

export default function GRNDetailPage({ id, onNavigate }: Props) {
  const [grn, setGrn] = useState<GoodsReceipt | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    void getGRN(id)
      .then((res: any) => setGrn(res.data ? res.data : res))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load GRN'))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) return <div className="po-detail-page"><p>Loading...</p></div>
  if (error) return <div className="po-detail-page"><p className="po-error">{error}</p></div>
  if (!grn) return <div className="po-detail-page"><p>Not found.</p></div>

  return (
    <div className="po-detail-page">
      <header className="po-detail-header">
        <div>
          <p className="po-eyebrow">Goods Receipt</p>
          <div className="po-title-row">
            <h1>{grn.grn_number}</h1>
            <span className="po-pill status-received">Received</span>
          </div>
          <p className="po-subtitle">
            Received on {new Date(grn.receipt_date).toLocaleDateString()} by {grn.received_by_name}
          </p>
        </div>
        <div className="po-actions-bar">
          <button className="po-btn po-btn-ghost" onClick={() => onNavigate('Goods receipts')}>
            Back to List
          </button>
        </div>
      </header>

      <div className="po-detail-grid">
        <div className="po-detail-main">
          <section className="po-card">
            <h2 className="po-card-title">Receipt Details</h2>
            <div className="po-info-grid">
              <div><span>PO Number</span><strong>{grn.po_number}</strong></div>
              <div><span>Warehouse</span><strong>{grn.warehouse_name}</strong></div>
              <div><span>Invoice Ref</span><strong>{grn.supplier_invoice_reference || '—'}</strong></div>
              <div><span>Notes</span><strong>{grn.notes || '—'}</strong></div>
            </div>
          </section>

          <section className="po-card">
            <h2 className="po-card-title">Received Items</h2>
            <div className="po-line-table-wrap">
              <table className="po-line-table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Batch Number</th>
                    <th style={{ textAlign: 'right' }}>Qty Received</th>
                    <th style={{ textAlign: 'right' }}>Unit Cost</th>
                  </tr>
                </thead>
                <tbody>
                  {grn.items.map(item => (
                    <tr key={item.id}>
                      <td>{item.item_code ? `${item.item_code} - ${item.item_name}` : item.item_name}</td>
                      <td>{item.batch_number || '—'}</td>
                      <td style={{ textAlign: 'right' }}>{item.quantity_received}</td>
                      <td style={{ textAlign: 'right' }}>{item.unit_cost.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>
    </div>
  )
}