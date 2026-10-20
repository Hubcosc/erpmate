/** Role definitions — the single source of truth for what each role can open.
 *  The backend enforces the same rules server-side on every request. */

export const ROLE_LABELS: Record<string, string> = {
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

export const ROLE_MENUS: Record<string, string[]> = {
  department_head: ['Overview', 'Purchase requisitions', 'Material requests'],
  production_manager: ['Overview', 'Purchase requisitions', 'Material requests', 'Stores Overview', 'Production receipts', 'Inventory', 'Reports'],
  finance: ['Overview', 'Purchase requisitions', 'Purchase orders', 'Goods receipts', 'Reports'],
  purchasing: ['Overview', 'Purchase requisitions', 'Purchase orders', 'Goods receipts', 'Inventory'],
  store_keeper: ['Overview', 'Material requests', 'Main Store', 'Stores Overview', 'Goods receipts', 'Inventory', 'Reports'],
  production_store_keeper: ['Overview', 'Production Store', 'Finished Goods Store', 'Stores Overview', 'Production receipts', 'Inventory', 'Reports'],
  sales: ['Overview', 'Finished Goods Store', 'Sales orders', 'Delivery notes', 'Inventory'],
  sales_head: ['Overview', 'Finished Goods Store', 'Sales orders', 'Delivery notes', 'Reports'],
  admin: [
    'Overview', 'Purchase requisitions', 'Material requests',
    'Main Store', 'Production Store', 'Finished Goods Store', 'Stores Overview',
    'Purchase orders', 'Goods receipts', 'Inventory', 'Production receipts', 'Sales orders', 'Delivery notes',
    'Reports',
    'Departments', 'Categories', 'Items', 'Suppliers', 'Customers', 'Warehouses',
    'Users', 'Roles & Permissions', 'Audit Log',
  ],
}

/** The capability a role grants, grouped for the permissions matrix. */
export const ROLE_CAPABILITIES: Record<string, string[]> = {
  department_head: ['Raise purchase requisitions', 'Raise material requests', 'Acknowledge material receipts'],
  production_manager: ['Approve or return purchase requisitions', 'Approve or reject material requests', 'Record production receipts', 'View inventory and reports'],
  finance: ['Approve purchase requisitions', 'Suspend or reject requisitions', 'View purchase orders and receipts', 'View reports and valuation'],
  purchasing: ['Create purchase orders from approved requisitions', 'Set suppliers, prices and terms', 'Issue purchase orders'],
  store_keeper: ['Post goods receipt notes', 'Issue material from the raw-material store', 'View stock and alerts'],
  production_store_keeper: ['Record production receipts', 'Manage the finished-goods store', 'View stock and alerts'],
  sales: ['Create sales orders', 'Post delivery notes', 'Download delivery notes as PDF'],
  sales_head: ['Create sales orders', 'Post delivery notes', 'View sales reports'],
  admin: ['Full access to every module', 'Manage users and roles', 'Manage master data'],
}
