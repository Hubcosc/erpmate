import { useEffect, useState } from 'react'
import { listPurchaseRequisitions } from '../../../api/purchase'
import { listMaterialRequisitions } from '../../../api/material'
import { listSalesOrders } from '../../../api/sales'
import { Icon, type IconName } from '../../../shared/components/Icon'
import { listDepartments, listItems, listSuppliers, listCustomers } from '../../../api/master' 
import './Dashboard.css'

type DashboardProps = {
  dateLabel: string
  userName: string
  role: string
  onNavigate: (page: string) => void
}

const roleLabels: Record<string, string> = {
  department_head: 'Department Head',
  production_manager: 'Production Manager',
  finance: 'Finance',
  purchasing: 'Purchasing Team',
  store_keeper: 'Store Keeper',
  production_store_keeper: 'Production Store Keeper',
  sales: 'Sales Staff',
  sales_head: 'Sales Head',
  admin: 'System Administrator',
}

const roleMenus: Record<string, string[]> = {
  department_head: ['Overview', 'Purchase requisitions', 'Material requests'],
  production_manager: ['Overview', 'Purchase requisitions', 'Material requests', 'Production receipts', 'Inventory', 'Reports'],
  finance: ['Overview', 'Purchase requisitions', 'Purchase orders', 'Goods receipts', 'Reports'],
  purchasing: ['Overview', 'Purchase requisitions', 'Purchase orders', 'Goods receipts', 'Inventory'],
  store_keeper: ['Overview', 'Material requests', 'Goods receipts', 'Inventory', 'Reports'],
  production_store_keeper: ['Overview', 'Production receipts', 'Inventory', 'Reports'],
  sales: ['Overview', 'Sales orders', 'Delivery notes', 'Inventory'],
  sales_head: ['Overview', 'Sales orders', 'Delivery notes', 'Reports'],
  admin: ['Overview', 'Purchase requisitions', 'Material requests', 'Purchase orders', 'Goods receipts', 'Inventory', 'Production receipts', 'Sales orders', 'Delivery notes', 'Reports'],
}

const roleFocus: Record<string, { intro: string; focus: string; focusDetail: string }> = {
  department_head: { intro: 'Track your purchase and material requests from one place.', focus: 'Your requests', focusDetail: 'Create and follow requests for your department' },
  production_manager: { intro: 'Review requests and keep production supplied.', focus: 'Approval workspace', focusDetail: 'Purchase and material requests needing review' },
  finance: { intro: 'Review budget decisions and purchase commitments.', focus: 'Finance approvals', focusDetail: 'Purchase requisitions awaiting a finance decision' },
  purchasing: { intro: 'Follow approved purchase requests through procurement.', focus: 'Purchasing workspace', focusDetail: 'Approved requests ready for purchasing follow-up' },
  store_keeper: { intro: 'Keep raw-material requests and store operations moving.', focus: 'Store workspace', focusDetail: 'Approved material requests and stock activity' },
  production_store_keeper: { intro: 'Keep finished goods and production receipts in view.', focus: 'Production store', focusDetail: 'Finished goods receipts and availability' },
  sales: { intro: 'Keep orders and finished-goods availability close at hand.', focus: 'Sales workspace', focusDetail: 'Orders, delivery preparation, and availability' },
  sales_head: { intro: 'See sales activity and performance across your team.', focus: 'Sales overview', focusDetail: 'Sales performance and operational reports' },
  admin: { intro: 'Manage users, roles, and the complete ERP workspace.', focus: 'Administration', focusDetail: 'System access and workspace configuration' },
}

export default function Dashboard({ dateLabel, userName, role, onNavigate }: DashboardProps) {
  const [metrics, setMetrics] = useState({ purchase: '—', approvals: '—', materials: '—', sales: '—' })
  const [adminMetrics, setAdminMetrics] = useState({ departments: '—', items: '—', suppliers: '—', customers: '—' })

  useEffect(() => {
    let active = true
    void Promise.allSettled([
      listPurchaseRequisitions(1, 100),
      listMaterialRequisitions(1, 100),
      listSalesOrders(1, 100),
    ]).then(([purchaseResult, materialResult, salesResult]) => {
      if (!active) return
      const purchase = purchaseResult.status === 'fulfilled' ? (purchaseResult.value.data ?? []) : []
      const materials = materialResult.status === 'fulfilled' ? (materialResult.value.data ?? []) : []
      const sales = salesResult.status === 'fulfilled' ? (salesResult.value.data ?? []) : []
      
      const openApprovals = [...purchase, ...materials].filter((item) =>
        ['pending_pm', 'pending_finance'].includes(item.status)
      ).length
      
      setMetrics({
        purchase: String(purchase.length),
        approvals: String(openApprovals),
        materials: String(materials.length),
        sales: String(sales.length),
      })
    })
    return () => { active = false }
  }, [role])

  useEffect(() => {
    if (role !== 'admin') return
    let active = true
    void Promise.allSettled([
      listDepartments(),
      listItems(undefined, 1, 1),
      listSuppliers(1, 1),
      listCustomers(1, 1),
    ]).then(([deptRes, itemRes, supplierRes, customerRes]) => {
      if (!active) return
      const departments = deptRes.status === 'fulfilled'? (Array.isArray(deptRes.value) ? deptRes.value.length : (deptRes.value as any)?.data?.length ?? 0): 0
      const items = itemRes.status === 'fulfilled' ? itemRes.value.meta?.total ?? 0 : 0
      const suppliers = supplierRes.status === 'fulfilled' ? supplierRes.value.meta?.total ?? 0 : 0
      const customers = customerRes.status === 'fulfilled' ? customerRes.value.meta?.total ?? 0 : 0
      setAdminMetrics({
        departments: String(departments),
        items: String(items),
        suppliers: String(suppliers),
        customers: String(customers),
      })
    })
    return () => { active = false }
  }, [role])

  const firstName = userName.trim().split(/\s+/)[0] || 'there'

  const context = roleFocus[role] ?? {
    intro: "Here's what's happening across your workspace today.",
    focus: 'Your workspace',
    focusDetail: 'Tools and updates for your role',
  }

  const stats: { label: string; icon: IconName; tone: string; note: string; value: string; page: string }[] = role === 'admin'
    ? [
        { label: 'Departments', icon: 'overview', tone: 'tone-1', note: 'Org units', value: adminMetrics.departments, page: 'Departments' },
        { label: 'Items', icon: 'materials', tone: 'tone-2', note: 'Catalog entries', value: adminMetrics.items, page: 'Items' },
        { label: 'Suppliers', icon: 'purchase', tone: 'tone-3', note: 'Vendor records', value: adminMetrics.suppliers, page: 'Suppliers' },
        { label: 'Customers', icon: 'sales', tone: 'tone-4', note: 'Client records', value: adminMetrics.customers, page: 'Customers' },
      ]
    : [
        { label: 'Purchase requisitions', icon: 'purchase', tone: 'tone-1', note: 'Request workflow', value: metrics.purchase, page: 'Purchase requisitions' },
        { label: 'Awaiting approvals', icon: 'clock', tone: 'tone-2', note: 'PM & Finance', value: metrics.approvals, page: 'Purchase requisitions' },
        { label: 'Material requests', icon: 'materials', tone: 'tone-3', note: 'Store requests', value: metrics.materials, page: 'Material requests' },
        { label: 'Sales orders', icon: 'sales', tone: 'tone-4', note: 'Pending delivery', value: metrics.sales, page: 'Sales orders' },
      ]

  return (
    <div className="db-page">
      {/* Header */}
      <header className="db-header">
        <div>
          <p className="db-role">{roleLabels[role] ?? 'Your workspace'}</p>
          <h1>Welcome, <span>{firstName}</span></h1>
          <p className="db-intro">{context.intro}</p>
        </div>
        <div className="db-header-side">
          <span className="db-date"><Icon name="clock" size={15} />{dateLabel}</span>
          <button
            className="db-btn"
            type="button"
            onClick={() => onNavigate(roleMenus[role]?.[1] ?? 'Overview')}
          >
            Open your workspace
          </button>
        </div>
      </header>

      {/* Metrics */}
      <div className="db-section-head">
        <h2>{context.focus}</h2>
        <p>{context.focusDetail}</p>
      </div>

      <section className="db-stats" aria-label="Workspace metrics">
        {stats.map((stat) => (
          <article className={`db-stat ${stat.tone}`} key={stat.label}>
            <span className="db-stat-icon"><Icon name={stat.icon} size={18} /></span>
            <h3>{stat.label}</h3>
            <p className="db-stat-value">{stat.value}</p>
            <span className="db-stat-bar" aria-hidden="true" />
            <p className="db-stat-note">{stat.note}</p>
            <button className="db-btn db-btn-small" type="button" onClick={() => onNavigate(stat.page)}>
              Open
            </button>
          </article>
        ))}
      </section>

      {/* Lower panels */}
      <div className="db-grid">
        <section className="db-panel">
          <div className="db-panel-head">
            <div>
              <h2>Phase 1 foundation</h2>
              <p>Your ERP workspace is being set up</p>
            </div>
            <span className="db-badge">In progress</span>
          </div>

          <div className="db-list">
            <WorkflowItem icon="purchase" title="Purchase requisitions" detail="Request & approval workflow" state="Ready to connect" onClick={() => onNavigate('Purchase requisitions')} />
            <WorkflowItem icon="materials" title="Material requests" detail="Internal store requests" state="Ready to connect" onClick={() => onNavigate('Material requests')} />
            <WorkflowItem icon="inventory" title="Raw material store" detail="Stock, receipts & issues" state="In progress" onClick={() => onNavigate('Inventory')} />
            <WorkflowItem icon="bell" title="Team notifications" detail="Keep everyone up to date" state="Coming next" onClick={() => onNavigate('Settings')} />
          </div>
          <p className="db-note">Dashboard numbers appear after connecting live ERP data.</p>
        </section>

        <section className="db-panel">
          <div className="db-panel-head">
            <div>
              <h2>Quick access</h2>
              <p>Jump into a workspace</p>
            </div>
          </div>

          <div className="db-list">
            <QuickLink icon="purchase" title="Purchase requisitions" detail="Create, review, and track requests" onClick={() => onNavigate('Purchase requisitions')} />
            <QuickLink icon="materials" title="Material requests" detail="Request items from your store" onClick={() => onNavigate('Material requests')} />
            <QuickLink icon="reports" title="Reports & insights" detail="Explore operational reporting" onClick={() => onNavigate('Reports')} />
          </div>
        </section>
      </div>
    </div>
  )
}

function WorkflowItem({ icon, title, detail, state, onClick }: { icon: IconName; title: string; detail: string; state: string; onClick: () => void }) {
  return (
    <button className="db-row" type="button" onClick={onClick}>
      <span className={`db-row-icon ic-${icon}`}><Icon name={icon} size={16} /></span>
      <span className="db-row-copy"><strong>{title}</strong><small>{detail}</small></span>
      <span className={`db-state ${state === 'In progress' ? 'is-progress' : ''}`}>{state}</span>
      <Icon name="chevron" size={15} />
    </button>
  )
}

function QuickLink({ icon, title, detail, onClick }: { icon: IconName; title: string; detail: string; onClick: () => void }) {
  return (
    <button className="db-row" type="button" onClick={onClick}>
      <span className={`db-row-icon ic-${icon}`}><Icon name={icon} size={16} /></span>
      <span className="db-row-copy"><strong>{title}</strong><small>{detail}</small></span>
      <Icon name="chevron" size={15} />
    </button>
  )
}