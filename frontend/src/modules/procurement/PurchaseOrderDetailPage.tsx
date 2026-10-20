import { useEffect, useState } from 'react'
import {
  getPO,
  updatePO,
  issuePO,
  type PurchaseOrder,
} from '../../api/purchaseOrders'
import { listSuppliers } from '../../api/master'
import Modal from '../../shared/components/Modal'
import './PurchaseOrderDetailPage.css'

type Props = {
  id: string
  onNavigate: (page: string) => void
  role: string
}

const STATUS_LABELS: Record<string, string> = {
  draft: 'Draft',
  issued: 'Issued',
  partially_received: 'Partially Received',
  received: 'Received',
}

export default function PurchaseOrderDetailPage({ id, onNavigate, role }: Props) {
  const [po, setPo] = useState<PurchaseOrder | null>(null)
  const [suppliers, setSuppliers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // Edit Modal state
  const [isEditing, setIsEditing] = useState(false)
  const [editSupplier, setEditSupplier] = useState('')
  const [editExpectedDate, setEditExpectedDate] = useState('')
  const [editPaymentTerms, setEditPaymentTerms] = useState('')
  const [editDeliveryTerms, setEditDeliveryTerms] = useState('')
  const [editNotes, setEditNotes] = useState('')
  const [editPrices, setEditPrices] = useState<Record<string, number>>({})
  const [showIssueModal, setShowIssueModal] = useState(false)
  const [saving, setSaving] = useState(false)
  const [actionError, setActionError] = useState('')

  const loadData = async () => {
    setLoading(true)
    setError('')
    try {
      const [poRes, supRes] = await Promise.all([
        getPO(id),
        listSuppliers(1, 100),
      ])
      const poData = (poRes as any).data ? (poRes as any).data : poRes
      setPo(poData)
      setSuppliers((supRes as any).data ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load PO')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadData()
  }, [id])

  const openEditModal = () => {
    if (!po) return
    setEditSupplier(po.supplier_id || '')
    setEditExpectedDate(po.expected_delivery_date ? po.expected_delivery_date.split('T')[0] : '')
    setEditPaymentTerms(po.payment_terms || '')
    setEditDeliveryTerms(po.delivery_terms || '')
    setEditNotes(po.notes || '')
    
    const prices: Record<string, number> = {}
    po.items.forEach(item => {
      prices[item.id] = item.unit_price || 0
    })
    setEditPrices(prices)
    
    setActionError('')
    setIsEditing(true)
  }

  const handleSave = async () => {
    if (!po) return
    setSaving(true)
    setActionError('')
    try {
      const itemsPayload = po.items.map(item => ({
        id: item.id,
        unit_price: editPrices[item.id] || 0,
      }))

      const res = await updatePO(po.id, {
        supplier_id: editSupplier || undefined,
        expected_delivery_date: editExpectedDate || null,
        payment_terms: editPaymentTerms || null,
        delivery_terms: editDeliveryTerms || null,
        notes: editNotes || null,
        items: itemsPayload,
      })
      
      setPo((res as any).data ? (res as any).data : res)
      setIsEditing(false)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  const confirmIssue = async () => {
    if (!po) return
    setSaving(true)
    setActionError('') 
    try {
      const res = await issuePO(po.id)
      setPo((res as any).data ? (res as any).data : res)
      setShowIssueModal(false)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to issue PO')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="po-detail-page"><p className="po-state">Loading Purchase Order...</p></div>
  if (error) return <div className="po-detail-page"><p className="po-state po-state-error">{error}</p></div>
  if (!po) return <div className="po-detail-page"><p className="po-state">Not found.</p></div>

  const isDraft = po.status === 'draft'
  const canEdit = isDraft && (role === 'purchasing' || role === 'admin')
  const canIssue = isDraft && (role === 'purchasing' || role === 'admin')

  return (
    <div className="po-detail-page">
      <header className="po-detail-header">
        <div>
          <p className="po-eyebrow">Purchase Order</p>
          <div className="po-title-row">
            <h1>{po.po_number}</h1>
            <span className={`po-pill status-${po.status}`}>
              {STATUS_LABELS[po.status] || po.status}
            </span>
          </div>
          <p className="po-subtitle">
            Created on {new Date(po.created_at).toLocaleDateString()}
          </p>
        </div>
        <div className="po-actions-bar">
          <button className="po-btn po-btn-ghost" onClick={() => onNavigate('Purchase orders')}>
            Back to List
          </button>
          {canEdit && (
            <button className="po-btn po-btn-ghost" onClick={openEditModal}>
              Edit PO
            </button>
          )}
            {canIssue && (
            <button className="po-btn po-btn-primary" onClick={() => { setActionError(''); setShowIssueModal(true) }} disabled={saving}>
                Issue PO
            </button>
            )}
        </div>
      </header>

      <div className="po-detail-grid">
        <div className="po-detail-main">
          <section className="po-card">
            <h2 className="po-card-title">Order Details</h2>
            <div className="po-info-grid">
              <div><span>Supplier</span><strong>{po.supplier_name || 'Not selected'}</strong></div>
              <div><span>Expected Date</span><strong>{po.expected_delivery_date ? new Date(po.expected_delivery_date).toLocaleDateString() : '—'}</strong></div>
              <div><span>Payment Terms</span><strong>{po.payment_terms || '—'}</strong></div>
              <div><span>Delivery Terms</span><strong>{po.delivery_terms || '—'}</strong></div>
              <div className="po-info-wide"><span>Notes</span><p>{po.notes || '—'}</p></div>
            </div>
          </section>

          <section className="po-card">
            <h2 className="po-card-title">Line Items ({po.item_count})</h2>
            <div className="po-line-table-wrap">
              <table className="po-line-table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Description</th>
                    <th style={{ textAlign: 'right' }}>Ordered</th>
                    <th style={{ textAlign: 'right' }}>Received</th>
                    <th>Unit</th>
                    <th style={{ textAlign: 'right' }}>Unit Price</th>
                    <th style={{ textAlign: 'right' }}>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {po.items.map(item => (
                    <tr key={item.id}>
                      <td>{item.item_code ? `${item.item_code} - ${item.item_name}` : '—'}</td>
                      <td>{item.description || '—'}</td>
                      <td style={{ textAlign: 'right' }}>{item.quantity}</td>
                      <td style={{ textAlign: 'right' }}>{item.received_quantity}</td>
                      <td>{item.unit_of_measure || '—'}</td>
                      <td style={{ textAlign: 'right' }}>{item.unit_price.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                      <td style={{ textAlign: 'right' }}>{item.line_total.toLocaleString(undefined, { minimumFractionDigits: 2 })}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'right', fontWeight: 600 }}>Total Amount:</td>
                    <td style={{ textAlign: 'right', fontWeight: 600, color: 'var(--po-navy)' }}>
                      {po.total_amount.toLocaleString(undefined, { minimumFractionDigits: 2 })}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </section>
        </div>
      </div>

      {/* Edit Modal */}
      <Modal open={isEditing} title={`Edit ${po.po_number}`} onClose={() => setIsEditing(false)}>
        <div className="po-edit-form">
          {actionError && <div className="po-error">{actionError}</div>}
          
          <div className="po-grid">
            <label className="po-field">
              <span>Supplier <i>*</i></span>
              <select value={editSupplier} onChange={e => setEditSupplier(e.target.value)}>
                <option value="">Select Supplier...</option>
                {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </label>

            <label className="po-field">
              <span>Expected Delivery Date</span>
              <input type="date" value={editExpectedDate} onChange={e => setEditExpectedDate(e.target.value)} />
            </label>

            <label className="po-field">
              <span>Payment Terms</span>
              <input type="text" value={editPaymentTerms} onChange={e => setEditPaymentTerms(e.target.value)} placeholder="e.g. Net 30" />
            </label>

            <label className="po-field">
              <span>Delivery Terms</span>
              <input type="text" value={editDeliveryTerms} onChange={e => setEditDeliveryTerms(e.target.value)} placeholder="e.g. FOB" />
            </label>

            <label className="po-field po-field-wide">
              <span>Notes</span>
              <textarea value={editNotes} onChange={e => setEditNotes(e.target.value)} rows={2} />
            </label>
          </div>

          <h3 className="po-sub-title">Line Items Pricing</h3>
          <div className="po-line-table-wrap">
            <table className="po-line-table">
              <thead>
                <tr>
                  <th>Item</th>
                  <th style={{ width: '120px' }}>Qty</th>
                  <th style={{ width: '150px' }}>Unit Price</th>
                </tr>
              </thead>
              <tbody>
                {po.items.map(item => (
                  <tr key={item.id}>
                    <td>{item.item_code ? `${item.item_code} - ${item.item_name}` : item.description}</td>
                    <td>{item.quantity} {item.unit_of_measure}</td>
                    <td>
                      <input
                        type="number"
                        min="0"
                        step="0.01"
                        value={editPrices[item.id] || ''}
                        onChange={e => setEditPrices({ ...editPrices, [item.id]: Number(e.target.value) })}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="po-modal-actions">
            <button className="po-btn po-btn-ghost" onClick={() => setIsEditing(false)} disabled={saving}>
              Cancel
            </button>
            <button className="po-btn po-btn-primary" onClick={handleSave} disabled={saving}>
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </div>
      </Modal>

            {/* Issue Confirmation Modal */}
      <Modal open={showIssueModal} title="Issue Purchase Order" onClose={() => !saving && setShowIssueModal(false)}>
        <div className="po-edit-form">
          {actionError && <div className="po-error">{actionError}</div>}
          <p>Are you sure you want to issue <strong>{po.po_number}</strong>?</p>
          <p style={{ fontSize: '13px', color: 'var(--po-muted)' }}>
            Once issued, this Purchase Order is locked and cannot be edited. It will be sent to the supplier.
          </p>
          <div className="po-modal-actions">
            <button className="po-btn po-btn-ghost" onClick={() => setShowIssueModal(false)} disabled={saving}>
              Cancel
            </button>
            <button className="po-btn po-btn-primary" onClick={confirmIssue} disabled={saving}>
              {saving ? 'Issuing...' : 'Yes, Issue PO'}
            </button>
          </div>
        </div>
      </Modal>
      
    </div>
  )
}