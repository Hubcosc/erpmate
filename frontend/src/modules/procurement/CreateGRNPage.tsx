import { useEffect, useState } from 'react'
import { listPOs, getPO, type PurchaseOrder, type POSummary } from '../../api/purchaseOrders'
import { listWarehouses } from '../../api/master'
import { createGRN } from '../../api/goodsReceipts'
import './CreateGRNPage.css'

type Props = {
  onNavigate: (page: string, id?: string) => void
}

export default function CreateGRNPage({ onNavigate }: Props) {
  const [issuedPOs, setIssuedPOs] = useState<POSummary[]>([])
  const [warehouses, setWarehouses] = useState<any[]>([])
  const [selectedPO, setSelectedPO] = useState<PurchaseOrder | null>(null)

  const [poId, setPoId] = useState('')
  const [warehouseId, setWarehouseId] = useState('')
  const [invoiceRef, setInvoiceRef] = useState('')
  const [notes, setNotes] = useState('')

  // State for the items being received
  const [lines, setLines] = useState<Record<string, { qty: number; batch: string }>>({})

  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  // Load POs and Warehouses
  useEffect(() => {
    void Promise.all([
      listPOs({ status: 'issued', limit: 100 }).then(res => {
        const list = (res as any).data ? (res as any).data : res
        setIssuedPOs(Array.isArray(list) ? list : [])
      }),
        listWarehouses().then(res => {
        // Handle envelope vs raw array safely
        const list = (res as any).data ? (res as any).data : res
        const arrayList = Array.isArray(list) ? list : []
        setWarehouses(arrayList.filter((w: any) => w.warehouse_type === 'raw_material'))
      })
    ]).finally(() => setLoading(false))
  }, [])

  // When the user selects a PO, load its full details
  useEffect(() => {
    if (!poId) {
      setSelectedPO(null)
      setLines({})
      return
    }
    setLoading(true)
    void getPO(poId).then(res => {
      const po = (res as any).data ? (res as any).data : res
      setSelectedPO(po)
      
      const initialLines: Record<string, { qty: number; batch: string }> = {}
      po.items.forEach((item: any) => {
        // Default to receiving the remaining quantity
        initialLines[item.id] = { qty: item.remaining_quantity, batch: '' } 
      })
      setLines(initialLines)
    }).finally(() => setLoading(false))
  }, [poId])

  const updateLine = (itemId: string, field: 'qty' | 'batch', value: any) => {
    setLines(prev => ({
      ...prev,
      [itemId]: { ...prev[itemId], [field]: value }
    }))
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!poId) return setError('Please select a Purchase Order')
    if (!warehouseId) return setError('Please select a Warehouse')
    if (!selectedPO) return

    // Validate lines
    const payloadItems = []
    let hasItems = false

    for (const item of selectedPO.items) {
      const line = lines[item.id]
      if (!line || line.qty <= 0) continue // Skip items with 0 received

      if (line.qty > item.remaining_quantity) {
        return setError(`Cannot receive more than remaining quantity for ${item.item_name}`)
      }
      if (item.is_batch_tracked && !line.batch.trim()) {
        return setError(`Batch number is required for ${item.item_name}`)
      }

      hasItems = true
      payloadItems.push({
        purchase_order_item_id: item.id,
        quantity_received: Number(line.qty),
        batch_number: line.batch.trim() || undefined,
      })
    }

    if (!hasItems) return setError('Please enter a received quantity for at least one item')

    setSubmitting(true)
    try {
      const payload = {
        purchase_order_id: poId,
        warehouse_id: warehouseId,
        supplier_invoice_reference: invoiceRef.trim() || undefined,
        notes: notes.trim() || undefined,
        items: payloadItems,
      }

      const res = await createGRN(payload)
      const grn = (res as any).data ? (res as any).data : res
      onNavigate('Goods receipts', grn.id) // Send back to list
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create receipt')
    } finally {
      setSubmitting(false)
    }
  }

  if (loading && !selectedPO) return <div className="grn-page"><p>Loading...</p></div>

  return (
    <div className="grn-page">
      <header className="grn-header">
        <div>
          <p className="grn-eyebrow">Procurement</p>
          <h1>Receive Items</h1>
          <p className="grn-subtitle">Log incoming deliveries against an Issued Purchase Order.</p>
        </div>
        <button className="grn-btn grn-btn-ghost" type="button" onClick={() => onNavigate('Goods receipts')}>
          Cancel
        </button>
      </header>

      {error && <div className="grn-error">{error}</div>}

      <form onSubmit={submit} className="grn-form">
        <section className="grn-card">
          <h2 className="grn-card-title">Receipt Details</h2>
          <div className="grn-grid">
            <label className="grn-field">
              <span>Purchase Order <i>*</i></span>
              <select value={poId} onChange={e => setPoId(e.target.value)} required>
                <option value="">Select an Issued PO...</option>
                {issuedPOs.map(po => (
                  <option key={po.id} value={po.id}>{po.po_number} — {po.supplier_name || 'No Supplier'}</option>
                ))}
              </select>
            </label>

            <label className="grn-field">
              <span>Receiving Warehouse <i>*</i></span>
              <select value={warehouseId} onChange={e => setWarehouseId(e.target.value)} required>
                <option value="">Select a Raw Material Store...</option>
                {warehouses.map(w => (
                  <option key={w.id} value={w.id}>{w.name}</option>
                ))}
              </select>
            </label>

            <label className="grn-field">
              <span>Supplier Invoice Ref</span>
              <input type="text" value={invoiceRef} onChange={e => setInvoiceRef(e.target.value)} placeholder="e.g. INV-1001" />
            </label>

            <label className="grn-field">
              <span>Notes</span>
              <input type="text" value={notes} onChange={e => setNotes(e.target.value)} placeholder="Optional notes" />
            </label>
          </div>
        </section>

        {selectedPO && (
          <section className="grn-card">
            <h2 className="grn-card-title">Items to Receive</h2>
            <div className="grn-table-wrap">
              <table className="grn-table">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th style={{ textAlign: 'right' }}>Ordered</th>
                    <th style={{ textAlign: 'right' }}>Remaining</th>
                    <th style={{ width: '120px' }}>Receiving Qty</th>
                    <th>Batch Number</th>
                  </tr>
                </thead>
                <tbody>
                  {selectedPO.items.map(item => {
                    const line = lines[item.id] || { qty: 0, batch: '' }
                    const isFullyReceived = item.remaining_quantity <= 0
                    return (
                      <tr key={item.id} style={{ opacity: isFullyReceived ? 0.5 : 1 }}>
                        <td>
                          <strong>{item.item_code ? `${item.item_code} - ` : ''}{item.item_name || item.description}</strong>
                          {isFullyReceived && <span style={{ marginLeft: 8, fontSize: 11, color: 'green' }}>(Fully Received)</span>}
                        </td>
                        <td style={{ textAlign: 'right' }}>{item.quantity} {item.unit_of_measure}</td>
                        <td style={{ textAlign: 'right', fontWeight: 600 }}>{item.remaining_quantity}</td>
                        <td>
                          <input
                            type="number"
                            min="0"
                            max={item.remaining_quantity}
                            step="0.01"
                            value={line.qty || ''}
                            onChange={e => updateLine(item.id, 'qty', Number(e.target.value))}
                            disabled={isFullyReceived}
                            placeholder="0"
                          />
                        </td>
                        <td>
                          {item.is_batch_tracked ? (
                            <input
                              type="text"
                              value={line.batch}
                              onChange={e => updateLine(item.id, 'batch', e.target.value)}
                              placeholder="Required *"
                              disabled={isFullyReceived}
                            />
                          ) : (
                            <span style={{ color: '#9ca3af', fontSize: 12 }}>N/A</span>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </section>
        )}

        <div className="grn-form-footer">
          <button type="submit" className="grn-btn grn-btn-primary" disabled={submitting || !selectedPO}>
            {submitting ? 'Submitting...' : 'Confirm Receipt'}
          </button>
        </div>
      </form>
    </div>
  )
}