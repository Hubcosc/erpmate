import { useMemo, useState } from 'react'
import { setApiToken } from './api/client'
import { clearSession, loadSession, saveSession, type AuthSession } from './store/authStore'
import { Icon, type IconName } from './shared/components/Icon'
import { NotificationMenu } from './shared/components/NotificationMenu'
import { ROLE_LABELS, ROLE_MENUS } from './shared/constants/roles'
import LoginPage from './modules/auth/pages/LoginPage'
import Dashboard from './modules/dashboard/pages/Dashboard'
import DepartmentsPage from './modules/master/departments/DepartmentsPage'
import './styles/tokens.css'
import './AppShell.css'
import UsersPage from './modules/users/pages/UsersPage'
import WarehousesPage from './modules/master/warehouses/WarehousesPage'
import SuppliersPage from './modules/master/suppliers/SuppliersPage'
import CustomersPage from './modules/master/customers/CustomersPage'
import CategoriesPage from './modules/master/categories/CategoriesPage'
import ItemsPage from './modules/master/items/ItemsPage'
import RequisitionsListPage from './modules/purchase/RequisitionsListPage'
import CreateRequisitionPage from './modules/purchase/CreateRequisitionPage'
import RequisitionDetailPage from './modules/purchase/RequisitionDetailPage'
import PurchaseOrdersListPage from './modules/procurement/PurchaseOrdersListPage'
import PurchaseOrderDetailPage from './modules/procurement/PurchaseOrderDetailPage'
import GRNListPage from './modules/procurement/GRNListPage'
import CreateGRNPage from './modules/procurement/CreateGRNPage'
import GRNDetailPage from './modules/procurement/GRNDetailPage'
import StockDashboardPage from './modules/inventory/StockDashboardPage'
import MaterialRequestsListPage from './modules/material/MaterialRequestsListPage'
import CreateMaterialRequestPage from './modules/material/CreateMaterialRequestPage'
import MaterialRequestDetailPage from './modules/material/MaterialRequestDetailPage'
import ProductionReceiptsListPage from './modules/production/ProductionReceiptsListPage'
import CreateProductionReceiptPage from './modules/production/CreateProductionReceiptPage'
import ProductionReceiptDetailPage from './modules/production/ProductionReceiptDetailPage'
import SalesOrdersListPage from './modules/sales/SalesOrdersListPage'
import CreateSalesOrderPage from './modules/sales/CreateSalesOrderPage'
import SalesOrderDetailPage from './modules/sales/SalesOrderDetailPage'
import DeliveryNotesListPage from './modules/sales/DeliveryNotesListPage'
import DeliveryNoteDetailPage from './modules/sales/DeliveryNoteDetailPage'
import ReportsPage from './modules/reports/ReportsPage'
import StockMovementsPage from './modules/inventory/StockMovementsPage'
import StoreViewPage from './modules/inventory/StoreViewPage'
import StoresOverviewPage from './modules/inventory/StoresOverviewPage'
import RolesPermissionsPage from './modules/admin/RolesPermissionsPage'
import AuditLogPage from './modules/admin/AuditLogPage'
// Sidebar groups. Swap the icon names for any other names your Icon component supports.
type NavItem = { label: string; icon: IconName }
const navGroups: { title: string; items: NavItem[] }[] = [
  {
    title: 'Workspace',
    items: [
      { label: 'Overview', icon: 'overview' },
      { label: 'Purchase requisitions', icon: 'purchase' },
      { label: 'Material requests', icon: 'materials' },
    ],
  },
  {
    title: 'Stores',
    items: [
      { label: 'Main Store', icon: 'inventory' },
      { label: 'Production Store', icon: 'materials' },
      { label: 'Finished Goods Store', icon: 'inventory' },
      { label: 'Stores Overview', icon: 'overview' },
    ],
  },
  {
    title: 'Operations',
    items: [
      { label: 'Purchase orders', icon: 'purchase' },
      { label: 'Goods receipts', icon: 'inventory' },
      { label: 'Inventory', icon: 'inventory' },
      { label: 'Production receipts', icon: 'materials' },
      { label: 'Sales orders', icon: 'purchase' },
      { label: 'Delivery notes', icon: 'inventory' },
    ],
  },
  {
    title: 'Insights',
    items: [{ label: 'Reports', icon: 'reports' }],
  },
  {
    title: 'Master Data',
    items: [
      { label: 'Departments', icon: 'overview' },
      { label: 'Categories', icon: 'overview' },
      { label: 'Items', icon: 'materials' },
      { label: 'Suppliers', icon: 'purchase' },
      { label: 'Customers', icon: 'sales' },
      { label: 'Warehouses', icon: 'inventory' },
    ],
  },
  {
    title: 'Administration',
    items: [
      { label: 'Users', icon: 'settings' },
      { label: 'Roles & Permissions', icon: 'settings' },
      { label: 'Audit Log', icon: 'reports' },
    ],
  },
]

const roleMenus = ROLE_MENUS
const roleLabels = ROLE_LABELS

// Pages that are not in the sidebar belong to a menu item (used for access checks
// and for keeping that menu item highlighted).
const PAGE_PARENT: Record<string, string> = {
  'Create PR': 'Purchase requisitions',
  'PR Detail': 'Purchase requisitions',
  'Create MR': 'Material requests',
  'MR Detail': 'Material requests',
  'Create Sales Order': 'Sales orders',
  'Sales Order Detail': 'Sales orders',
  'Create Delivery Note': 'Delivery notes',
  'Delivery Note Detail': 'Delivery notes',
  'Create Production Receipt': 'Production receipts',
  'Production Receipt Detail': 'Production receipts',
  'PO Detail': 'Purchase orders',
  'Create GRN': 'Goods receipts',
  'GRN Detail': 'Goods receipts',
}

function initials(name: string): string {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0].toUpperCase()).join('') || 'U'
}

function ModulePlaceholder({ page, onBack }: { page: string; onBack: () => void }) {
  return (
    <section className="sh-placeholder">
      <h1>{page}</h1>
      <p>This workspace is part of the ErpMate foundation and will open as its module UI is built.</p>
      <p className="sh-placeholder-note">Module interface is not connected yet</p>
      <button className="sh-btn" type="button" onClick={onBack}>Back to overview</button>
    </section>
  )
}

function NoAccess({ onBack }: { onBack: () => void }) {
  return (
    <section className="sh-placeholder">
      <h1>No access</h1>
      <p>Your role does not have permission to open this page.</p>
      <button className="sh-btn" type="button" onClick={onBack}>Back to overview</button>
    </section>
  )
}

function App() {
  // The API token must be set BEFORE any page renders. Child effects run before
  // parent effects, so setting it in a useEffect here would let the first page
  // fire its requests without a token after a browser refresh.
  const [session, setSession] = useState<AuthSession | null>(() => {
    const stored = loadSession()
    setApiToken(stored?.token ?? null)
    return stored
  })
  const [activePage, setActivePage] = useState('Overview')
  const [activeId, setActiveId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [closedGroups, setClosedGroups] = useState<Record<string, boolean>>({})

  const dateLabel = useMemo(
    () => new Intl.DateTimeFormat('en', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' }).format(new Date()),
    [],
  )

  const role = session?.user.role_name.toLowerCase() ?? ''
  const allowedPages = roleMenus[role] ?? ['Overview']
  const query = search.trim().toLowerCase()
  const visibleGroups = navGroups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => allowedPages.includes(item.label) && item.label.toLowerCase().includes(query)),
    }))
    .filter((group) => group.items.length > 0)

  // Highlight the parent menu item while on Create PR / PR Detail.
  const navActive = PAGE_PARENT[activePage] ?? activePage
  const isDetailPage = [
    'PR Detail', 'PO Detail', 'Create PR', 'Edit PR', 'Create GRN', 'GRN Detail',
    'MR Detail', 'Create MR', 'Sales Order Detail', 'Create Sales Order',
    'Delivery Note Detail', 'Create Delivery Note', 'Production Receipt Detail',
    'Create Production Receipt',
  ].includes(activePage)
  const canOpen = isDetailPage || roleMenus[role]?.includes(activePage)
  const canManageRequisitions = role === 'department_head' || role === 'admin'

  const handleLogin = (nextSession: AuthSession, remember: boolean) => {
    saveSession(nextSession, remember)
    setApiToken(nextSession.token)
    setSession(nextSession)
    setActivePage('Overview')
    setActiveId(null)
  }

  const handleLogout = () => {
    clearSession()
    setApiToken(null)
    setSession(null)
    setActivePage('Overview')
    setActiveId(null)
  }

  const toggleGroup = (title: string) => {
    setClosedGroups((current) => ({ ...current, [title]: !current[title] }))
  }

  const navigate = (page: string, id?: string) => {
    setActivePage(page)
    setActiveId(id || null)
    setSidebarOpen(false)
  }

  if (!session) return <LoginPage onLogin={handleLogin} />

  const fullName = session.user.full_name
  const goOverview = () => navigate('Overview')

  const renderPage = () => {
    if (!canOpen) return <NoAccess onBack={goOverview} />

    switch (activePage) {
      case 'Overview':
        return <Dashboard dateLabel={dateLabel} userName={fullName} role={role} onNavigate={navigate} />
      case 'Users':
        return <UsersPage />
      case 'Departments':
        return <DepartmentsPage />
      case 'Warehouses':
        return <WarehousesPage />
      case 'Suppliers':
        return <SuppliersPage />
      case 'Customers':
        return <CustomersPage />
      case 'Categories':
        return <CategoriesPage />
      case 'Items':
        return <ItemsPage />
      case 'Purchase requisitions':
        return <RequisitionsListPage role={role} onNavigate={navigate} />
      case 'Create PR':
        if (!canManageRequisitions) return <NoAccess onBack={() => navigate('Purchase requisitions')} />
        return <CreateRequisitionPage key={activeId ?? 'new'} onNavigate={navigate} editId={activeId ?? undefined} />
      case 'PR Detail':
        if (!activeId) return <RequisitionsListPage role={role} onNavigate={navigate} />
        return <RequisitionDetailPage key={activeId} id={activeId} role={role} onNavigate={navigate} />
      case 'Purchase orders':
        return <PurchaseOrdersListPage onNavigate={navigate} />
      case 'PO Detail':
        if (!activeId) return <PurchaseOrdersListPage onNavigate={navigate} />
        return <PurchaseOrderDetailPage id={activeId} role={role} onNavigate={navigate} />
      case 'Goods receipts':
        return <GRNListPage onNavigate={navigate} />
      case 'Create GRN':
        return <CreateGRNPage onNavigate={navigate} />
      case 'GRN Detail':
        if (!activeId) return <GRNListPage onNavigate={navigate} />
        return <GRNDetailPage id={activeId} onNavigate={navigate} />
      case 'Inventory':
        return <StockDashboardPage onNavigate={navigate} />
      case 'Material requests':
        return <MaterialRequestsListPage role={role} onNavigate={navigate} />
      case 'Create MR':
        if (!canManageRequisitions) return <NoAccess onBack={() => navigate('Material requests')} />
        return <CreateMaterialRequestPage onNavigate={navigate} />
      case 'MR Detail':
        if (!activeId) return <MaterialRequestsListPage role={role} onNavigate={navigate} />
        return <MaterialRequestDetailPage key={activeId} id={activeId} role={role} onNavigate={navigate} />
      case 'Production receipts':
        return <ProductionReceiptsListPage role={role} onNavigate={navigate} />
      case 'Create Production Receipt':
        if (role !== 'production_store_keeper' && role !== 'admin') {
          return <NoAccess onBack={() => navigate('Production receipts')} />
        }
        return <CreateProductionReceiptPage onNavigate={navigate} />
      case 'Production Receipt Detail':
        if (!activeId) return <ProductionReceiptsListPage role={role} onNavigate={navigate} />
        return <ProductionReceiptDetailPage key={activeId} id={activeId} onNavigate={navigate} />
      case 'Sales orders':
        return <SalesOrdersListPage role={role} onNavigate={navigate} />
      case 'Create Sales Order':
        if (role !== 'sales' && role !== 'sales_head' && role !== 'admin') {
          return <NoAccess onBack={() => navigate('Sales orders')} />
        }
        return <CreateSalesOrderPage onNavigate={navigate} />
      case 'Sales Order Detail':
        if (!activeId) return <SalesOrdersListPage role={role} onNavigate={navigate} />
        return <SalesOrderDetailPage key={activeId} id={activeId} role={role} onNavigate={navigate} />
      case 'Delivery notes':
        return <DeliveryNotesListPage onNavigate={navigate} />
      case 'Delivery Note Detail':
        if (!activeId) return <DeliveryNotesListPage onNavigate={navigate} />
        return <DeliveryNoteDetailPage key={activeId} id={activeId} onNavigate={navigate} />
      case 'Reports':
        return <ReportsPage />
      case 'Stock Movements':
        return <StockMovementsPage onNavigate={navigate} />
      case 'Main Store':
        return (
          <StoreViewPage
            warehouseType="raw_material"
            title="Main Store"
            description="The raw-material store. Goods receipts land here and material issues are posted from it."
            moduleColor="#14663F"
            onNavigate={navigate}
          />
        )
      case 'Production Store':
        return (
          <StoreViewPage
            warehouseType="raw_material"
            itemType="raw_material"
            title="Production Store"
            description="Raw material available to production. Material issues move stock from the main store to the production floor."
            moduleColor="#0F4A33"
            onNavigate={navigate}
          />
        )
      case 'Finished Goods Store':
        return (
          <StoreViewPage
            warehouseType="finished_goods"
            title="Finished Goods Store"
            description="Finished goods received from production, reserved and dispatched to customers."
            moduleColor="#2E9E5B"
            onNavigate={navigate}
          />
        )
      case 'Stores Overview':
        return <StoresOverviewPage onNavigate={navigate} />
      case 'Roles & Permissions':
        return <RolesPermissionsPage />
      case 'Audit Log':
        return <AuditLogPage onNavigate={navigate} />
      default:
        return <ModulePlaceholder page={activePage} onBack={goOverview} />
    }
  }

  return (
    <div className="sh-shell">
      {/* Top bar */}
      <header className="sh-topbar">
        <button
          className="sh-menu"
          type="button"
          aria-label="Toggle navigation"
          onClick={() => setSidebarOpen(!sidebarOpen)}
        >
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
            <path d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>

        <span className="sh-brand">erpmate<span>.</span></span>

        <label className="sh-search">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
            <circle cx="11" cy="11" r="7" />
            <path d="m20 20-3.5-3.5" />
          </svg>
          <input
            type="search"
            placeholder="Search menu..."
            aria-label="Search menu"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
        </label>

        {/* Notification bell with the unread count badge (spec 5.1) */}
        <div className="sh-topbar-right">
          <NotificationMenu />
          <span className="sh-user-name">{fullName}</span>
          <span className="sh-avatar sh-avatar-sm" aria-hidden="true">{initials(fullName)}</span>
        </div>
      </header>

      {sidebarOpen && (
        <button className="sh-scrim" type="button" aria-label="Close navigation" onClick={() => setSidebarOpen(false)} />
      )}

      {/* Sidebar */}
      <aside className={`sh-sidebar ${sidebarOpen ? 'is-open' : ''}`}>
        <div className="sh-profile">
          <span className="sh-avatar" aria-hidden="true">{initials(fullName)}</span>
          <div className="sh-profile-copy">
            <strong>{fullName}</strong>
            <span><span className="sh-status-dot" /> {roleLabels[role] ?? 'Online'}</span>
          </div>
        </div>

        <nav className="sh-nav" aria-label="Main navigation">
          {visibleGroups.map((group) => {
            const open = query !== '' || !closedGroups[group.title]
            const hasActive = group.items.some((item) => item.label === navActive)
            return (
              <div className="sh-group" key={group.title}>
                <button
                  className={`sh-group-head ${hasActive ? 'has-active' : ''}`}
                  type="button"
                  aria-expanded={open}
                  onClick={() => toggleGroup(group.title)}
                >
                  <span>{group.title}</span>
                  <svg className={`sh-chevron ${open ? 'is-open' : ''}`} width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </button>

                {open && (
                  <div className="sh-group-items">
                    {group.items.map((item) => (
                      <button
                        key={item.label}
                        type="button"
                        className={`sh-nav-item ${navActive === item.label ? 'is-active' : ''}`}
                        aria-current={navActive === item.label ? 'page' : undefined}
                        onClick={() => navigate(item.label)}
                      >
                        <Icon name={item.icon} size={17} />
                        <span>{item.label}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
          {visibleGroups.length === 0 && <p className="sh-nav-empty">No matching menu</p>}
        </nav>

        <button className="sh-logout" type="button" onClick={handleLogout}>
          <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
            <path d="m16 17 5-5-5-5M21 12H9" />
          </svg>
          <span>Log out</span>
        </button>
      </aside>

      {/* Content */}
      <main className="sh-main">
        <div className="sh-content">{renderPage()}</div>

        <footer className="sh-footer">
          <span>© 2026 ErpMate</span>
          <span>Workspace foundation · v0.1.0</span>
        </footer>
      </main>
    </div>
  )
}

export default App