// Package admin provides a Django-style administration site for the super
// administrator.
//
// Instead of hand-writing a page per entity, the admin is driven by a registry
// of models. Each model declares its table, the columns to show in a list, the
// fields to expose on a form, and the joins needed to resolve related names.
// The same registry is served to the frontend at GET /admin/models, so the UI
// renders exactly the fields the backend accepts.
//
// Every identifier used in SQL comes from this file — never from a request — and
// every value is passed as a bind parameter.
package admin

// FieldType tells the frontend which input to render.
type FieldType string

const (
	FieldText     FieldType = "text"
	FieldTextarea FieldType = "textarea"
	FieldNumber   FieldType = "number"
	FieldBoolean  FieldType = "boolean"
	FieldDate     FieldType = "date"
	FieldDateTime FieldType = "datetime"
	FieldSelect   FieldType = "select"
	FieldPassword FieldType = "password"
	FieldReadOnly FieldType = "readonly"
)

// Option is one choice of a select field.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Field is one editable attribute of a model.
type Field struct {
	Name     string    `json:"name"`
	Label    string    `json:"label"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	ReadOnly bool      `json:"read_only"`
	Options  []Option  `json:"options,omitempty"`
	Help     string    `json:"help,omitempty"`
	Min      float64   `json:"min,omitempty"`
	Step     float64   `json:"step,omitempty"`
	// Lookup names the reference list the frontend should populate the choices
	// from, e.g. "roles" or "departments".
	Lookup string `json:"lookup,omitempty"`

	// SQL is the column expression used by the service. It is never sent to
	// the client.
	SQL string `json:"-"`
	// DBType is the PostgreSQL type, used to cast bind parameters safely.
	DBType string `json:"-"`
	// WriteOnly fields are never read back, e.g. a password.
	WriteOnly bool `json:"-"`
}

// Column is one column of a list view.
type Column struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Align string `json:"align,omitempty"`
	Type  string `json:"type,omitempty"`

	// SQL is the column expression used by the service.
	SQL string `json:"-"`
}

// Model is one registered entity.
type Model struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	LabelPlural string   `json:"label_plural"`
	Group       string   `json:"group"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Columns     []Column `json:"columns"`
	Fields      []Field  `json:"fields"`
	Search      []string `json:"search,omitempty"`
	Editable    bool     `json:"editable"`

	// Table is the physical table. Joins are appended to the FROM clause.
	Table string `json:"-"`
	Joins []string `json:"-"`
	// SoftDelete marks tables that have a deleted_at column.
	SoftDelete bool `json:"-"`
	// OrderBy is the default ordering column.
	OrderBy string `json:"-"`
	// OrderDesc orders newest first.
	OrderDesc bool `json:"-"`
	// ForceRole restricts a model to a role other than admin.
	ForceRole string `json:"-"`
}

// ---------------------------------------------------------------------------
// Shared building blocks
// ---------------------------------------------------------------------------

func text(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldText, SQL: sql, DBType: "text"}
}

func textarea(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldTextarea, SQL: sql, DBType: "text"}
}

func number(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldNumber, Step: 0.0001, SQL: sql, DBType: "numeric"}
}

func boolean(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldBoolean, SQL: sql, DBType: "boolean"}
}

func date(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldDate, SQL: sql, DBType: "date"}
}

func dateTime(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldDateTime, SQL: sql, DBType: "timestamptz"}
}

func selectOf(name, label, sql, dbType string, options ...Option) Field {
	return Field{Name: name, Label: label, Type: FieldSelect, Options: options, SQL: sql, DBType: dbType}
}

func password(name, label, sql string) Field {
	return Field{Name: name, Label: label, Type: FieldPassword, SQL: sql, DBType: "text", WriteOnly: true}
}

func col(key, label, sql, align, typ string) Column {
	return Column{Key: key, Label: label, SQL: sql, Align: align, Type: typ}
}

var itemTypes = []Option{
	{Value: "raw_material", Label: "Raw material"},
	{Value: "finished_good", Label: "Finished good"},
	{Value: "consumable", Label: "Consumable"},
	{Value: "spare_part", Label: "Spare part"},
}

var warehouseTypes = []Option{
	{Value: "raw_material", Label: "Raw material store"},
	{Value: "finished_goods", Label: "Finished goods store"},
	{Value: "quarantine", Label: "Quarantine"},
}

var customerTypes = []Option{
	{Value: "local", Label: "Local"},
	{Value: "international", Label: "International"},
}

// ---------------------------------------------------------------------------
// The registry
// ---------------------------------------------------------------------------

// models is the ordered registry. Order here is the order shown in the admin
// navigation.
var models = []Model{
	// ---------------- Master data ----------------
	{
		Key: "items", Label: "Item", LabelPlural: "Items", Group: "Master data",
		Description: "Everything that can be bought, stocked or sold.",
		Icon:        "inventory", Table: "items", Editable: true, SoftDelete: true,
		OrderBy: "m.code",
		Search: []string{"m.code", "m.name", "m.description"},
		Columns: []Column{
			col("code", "Code", "m.code", "", "text"),
			col("name", "Name", "m.name", "", "text"),
			col("category", "Category", "c.name", "", "text"),
			col("item_type", "Type", "m.item_type", "", "badge"),
			col("unit", "Unit", "m.unit_of_measure", "", "text"),
			col("min_stock", "Min level", "m.min_stock_level", "right", "number"),
			col("reorder_level", "Reorder level", "m.reorder_level", "right", "number"),
			col("tracked", "Tracked", "m.is_batch_tracked", "", "boolean"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
		},
		Fields: []Field{
			{Name: "code", Label: "Code", Type: FieldText, Required: true, SQL: "m.code"},
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			textarea("description", "Description", "m.description"),
			{Name: "category_id", Label: "Category", Type: FieldSelect, SQL: "m.category_id", DBType: "uuid", Lookup: "item_categories", Help: "Optional category"},
			{Name: "item_type", Label: "Item type", Type: FieldSelect, Required: true, Options: itemTypes, SQL: "m.item_type", DBType: "varchar"},
			{Name: "unit_of_measure", Label: "Unit of measure", Type: FieldText, Required: true, SQL: "m.unit_of_measure"},
			number("min_stock_level", "Minimum stock level", "m.min_stock_level"),
			number("max_stock_level", "Maximum stock level", "m.max_stock_level"),
			number("reorder_level", "Reorder level", "m.reorder_level"),
			boolean("is_batch_tracked", "Batch tracked", "m.is_batch_tracked"),
			boolean("is_expiry_tracked", "Expiry tracked", "m.is_expiry_tracked"),
			boolean("is_active", "Active", "m.is_active"),
		},
		Joins: []string{"LEFT JOIN item_categories c ON c.id = m.category_id"},
	},
	{
		Key: "item_categories", Label: "Item category", LabelPlural: "Item categories", Group: "Master data",
		Description: "Groups used to organise the item catalogue.",
		Icon:        "categories", Table: "item_categories", Editable: true, SoftDelete: true,
		OrderBy: "m.name",
		Search: []string{"m.name"},
		Columns: []Column{
			col("name", "Name", "m.name", "", "text"),
			col("parent", "Parent", "p.name", "", "text"),
		},
		Fields: []Field{
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			{Name: "parent_id", Label: "Parent category", Type: FieldSelect, SQL: "m.parent_id", DBType: "uuid", Lookup: "item_categories"},
		},
		Joins: []string{"LEFT JOIN item_categories p ON p.id = m.parent_id"},
	},
	{
		Key: "suppliers", Label: "Supplier", LabelPlural: "Suppliers", Group: "Master data",
		Description: "Companies that goods are purchased from.",
		Icon:        "purchase", Table: "suppliers", Editable: true, SoftDelete: true,
		OrderBy: "m.name",
		Search: []string{"m.code", "m.name", "m.contact_person", "m.email"},
		Columns: []Column{
			col("code", "Code", "m.code", "", "text"),
			col("name", "Name", "m.name", "", "text"),
			col("contact", "Contact", "m.contact_person", "", "text"),
			col("phone", "Phone", "m.phone", "", "text"),
			col("email", "Email", "m.email", "", "text"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
		},
		Fields: []Field{
			{Name: "code", Label: "Code", Type: FieldText, SQL: "m.code"},
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			text("contact_person", "Contact person", "m.contact_person"),
			text("phone", "Phone", "m.phone"),
			text("email", "Email", "m.email"),
			textarea("address", "Address", "m.address"),
			text("tin_number", "TIN number", "m.tin_number"),
			boolean("is_active", "Active", "m.is_active"),
		},
	},
	{
		Key: "customers", Label: "Customer", LabelPlural: "Customers", Group: "Master data",
		Description: "Companies that finished goods are sold to.",
		Icon:        "sales", Table: "customers", Editable: true, SoftDelete: true,
		OrderBy: "m.name",
		Search: []string{"m.code", "m.name", "m.contact_person", "m.email"},
		Columns: []Column{
			col("code", "Code", "m.code", "", "text"),
			col("name", "Name", "m.name", "", "text"),
			col("type", "Type", "m.customer_type", "", "badge"),
			col("contact", "Contact", "m.contact_person", "", "text"),
			col("email", "Email", "m.email", "", "text"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
		},
		Fields: []Field{
			{Name: "code", Label: "Code", Type: FieldText, SQL: "m.code"},
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			{Name: "customer_type", Label: "Customer type", Type: FieldSelect, Required: true, Options: customerTypes, SQL: "m.customer_type", DBType: "varchar"},
			text("contact_person", "Contact person", "m.contact_person"),
			text("phone", "Phone", "m.phone"),
			text("email", "Email", "m.email"),
			textarea("address", "Address", "m.address"),
			text("tin_number", "TIN number", "m.tin_number"),
			boolean("is_active", "Active", "m.is_active"),
		},
	},
	{
		Key: "warehouses", Label: "Warehouse", LabelPlural: "Warehouses", Group: "Master data",
		Description: "Physical stores that hold stock.",
		Icon:        "inventory", Table: "warehouses", Editable: true, SoftDelete: true,
		OrderBy: "m.code",
		Search: []string{"m.code", "m.name", "m.location"},
		Columns: []Column{
			col("code", "Code", "m.code", "", "text"),
			col("name", "Name", "m.name", "", "text"),
			col("type", "Type", "m.warehouse_type", "", "badge"),
			col("location", "Location", "m.location", "", "text"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
		},
		Fields: []Field{
			{Name: "code", Label: "Code", Type: FieldText, Required: true, SQL: "m.code"},
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			{Name: "warehouse_type", Label: "Type", Type: FieldSelect, Required: true, Options: warehouseTypes, SQL: "m.warehouse_type", DBType: "varchar"},
			textarea("location", "Location", "m.location"),
			boolean("is_active", "Active", "m.is_active"),
		},
	},
	{
		Key: "departments", Label: "Department", LabelPlural: "Departments", Group: "Master data",
		Description: "Teams that raise requisitions.",
		Icon:        "overview", Table: "departments", Editable: true, SoftDelete: true,
		OrderBy: "m.name",
		Search: []string{"m.name", "m.code"},
		Columns: []Column{
			col("name", "Name", "m.name", "", "text"),
			col("code", "Code", "m.code", "", "text"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
		},
		Fields: []Field{
			{Name: "name", Label: "Name", Type: FieldText, Required: true, SQL: "m.name"},
			{Name: "code", Label: "Code", Type: FieldText, SQL: "m.code"},
			boolean("is_active", "Active", "m.is_active"),
		},
	},

	// ---------------- Access ----------------
	{
		Key: "users", Label: "User", LabelPlural: "Users", Group: "Access",
		Description: "Everyone who can sign in, and the role they hold.",
		Icon:        "settings", Table: "users", Editable: true, SoftDelete: true,
		OrderBy: "m.full_name",
		Search: []string{"m.full_name", "m.email", "m.employee_id"},
		Columns: []Column{
			col("full_name", "Name", "m.full_name", "", "text"),
			col("email", "Email", "m.email", "", "text"),
			col("employee_id", "Employee ID", "m.employee_id", "", "text"),
			col("role", "Role", "r.name", "", "badge"),
			col("department", "Department", "d.name", "", "text"),
			col("is_active", "Active", "m.is_active", "", "boolean"),
			col("last_login_at", "Last login", "m.last_login_at", "", "datetime"),
		},
		Fields: []Field{
			{Name: "employee_id", Label: "Employee ID", Type: FieldText, SQL: "m.employee_id"},
			{Name: "full_name", Label: "Full name", Type: FieldText, Required: true, SQL: "m.full_name"},
			{Name: "email", Label: "Email", Type: FieldText, Required: true, SQL: "m.email"},
			text("phone", "Phone", "m.phone"),
			{Name: "role_id", Label: "Role", Type: FieldSelect, Required: true, SQL: "m.role_id", DBType: "uuid", Lookup: "roles"},
			{Name: "department_id", Label: "Department", Type: FieldSelect, SQL: "m.department_id", DBType: "uuid", Lookup: "departments"},
			boolean("is_active", "Active", "m.is_active"),
			{Name: "password", Label: "Password", Type: FieldPassword, SQL: "m.password_hash", DBType: "text", WriteOnly: true, Help: "Leave blank to keep the current password. Minimum 12 characters."},
		},
		Joins: []string{"LEFT JOIN roles r ON r.id = m.role_id", "LEFT JOIN departments d ON d.id = m.department_id"},
	},
	{
		Key: "roles", Label: "Role", LabelPlural: "Roles", Group: "Access",
		Description: "The roles the application checks. Role names are referenced directly in the code, so they are read-only.",
		Icon:        "settings", Table: "roles", Editable: false,
		OrderBy: "m.name",
		Search: []string{"m.name", "m.description"},
		Columns: []Column{
			col("name", "Name", "m.name", "", "text"),
			col("description", "Description", "m.description", "", "text"),
		},
		Fields: []Field{
			{Name: "name", Label: "Name", Type: FieldText, ReadOnly: true, SQL: "m.name"},
			textarea("description", "Description", "m.description"),
		},
	},

	// ---------------- Procurement ----------------
	{
		Key: "purchase_requisitions", Label: "Purchase requisition", LabelPlural: "Purchase requisitions", Group: "Procurement",
		Description: "Requests raised by departments and their approval state.",
		Icon:        "purchase", Table: "purchase_requisitions", Editable: false, SoftDelete: true,
		OrderBy: "m.pr_number", OrderDesc: true,
		Search: []string{"m.pr_number", "m.justification"},
		Columns: []Column{
			col("pr_number", "PR number", "m.pr_number", "", "text"),
			col("department", "Department", "d.name", "", "text"),
			col("requester", "Requester", "u.full_name", "", "text"),
			col("status", "Status", "m.status", "", "badge"),
			col("required_date", "Required", "m.required_date", "", "date"),
			col("total_estimated", "Estimated", "m.total_estimated", "right", "number"),
			col("created_at", "Created", "m.created_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN departments d ON d.id = m.department_id", "LEFT JOIN users u ON u.id = m.requested_by"},
	},
	{
		Key: "purchase_orders", Label: "Purchase order", LabelPlural: "Purchase orders", Group: "Procurement",
		Description: "Orders issued to suppliers.",
		Icon:        "purchase", Table: "purchase_orders", Editable: false, SoftDelete: true,
		OrderBy: "m.po_number", OrderDesc: true,
		Search: []string{"m.po_number"},
		Columns: []Column{
			col("po_number", "PO number", "m.po_number", "", "text"),
			col("supplier", "Supplier", "s.name", "", "text"),
			col("status", "Status", "m.status", "", "badge"),
			col("order_date", "Order date", "m.order_date", "", "date"),
			col("expected_delivery_date", "Expected", "m.expected_delivery_date", "", "date"),
			col("total_amount", "Total", "m.total_amount", "right", "number"),
		},
		Joins: []string{"LEFT JOIN suppliers s ON s.id = m.supplier_id"},
	},
	{
		Key: "goods_receipt_notes", Label: "Goods receipt note", LabelPlural: "Goods receipt notes", Group: "Procurement",
		Description: "Goods received against purchase orders.",
		Icon:        "inventory", Table: "goods_receipt_notes", Editable: false, SoftDelete: true,
		OrderBy: "m.grn_number", OrderDesc: true,
		Search: []string{"m.grn_number", "m.supplier_invoice_reference"},
		Columns: []Column{
			col("grn_number", "GRN number", "m.grn_number", "", "text"),
			col("po_number", "PO number", "po.po_number", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("receipt_date", "Received", "m.receipt_date", "", "date"),
			col("supplier_invoice_reference", "Supplier invoice", "m.supplier_invoice_reference", "", "text"),
		},
		Joins: []string{"LEFT JOIN purchase_orders po ON po.id = m.purchase_order_id", "LEFT JOIN warehouses w ON w.id = m.warehouse_id"},
	},
	{
		Key: "purchase_order_items", Label: "Purchase order line", LabelPlural: "Purchase order lines", Group: "Procurement",
		Description: "The individual lines of a purchase order.",
		Icon:        "purchase", Table: "purchase_order_items", Editable: false,
		OrderBy: "m.created_at", OrderDesc: true,
		Columns: []Column{
			col("po_number", "PO number", "po.po_number", "", "text"),
			col("item", "Item", "i.name", "", "text"),
			col("quantity", "Quantity", "m.quantity", "right", "number"),
			col("received", "Received", "m.received_quantity", "right", "number"),
			col("unit_price", "Unit price", "m.unit_price", "right", "number"),
			col("line_total", "Total", "m.line_total", "right", "number"),
		},
		Joins: []string{"LEFT JOIN purchase_orders po ON po.id = m.purchase_order_id", "LEFT JOIN items i ON i.id = m.item_id"},
	},

	// ---------------- Production & materials ----------------
	{
		Key: "material_requisitions", Label: "Material requisition", LabelPlural: "Material requisitions", Group: "Production & materials",
		Description: "Department requests for raw material.",
		Icon:        "materials", Table: "material_requisitions", Editable: false, SoftDelete: true,
		OrderBy: "m.mr_number", OrderDesc: true,
		Search: []string{"m.mr_number", "m.purpose"},
		Columns: []Column{
			col("mr_number", "MR number", "m.mr_number", "", "text"),
			col("department", "Department", "d.name", "", "text"),
			col("requester", "Requester", "u.full_name", "", "text"),
			col("status", "Status", "m.status", "", "badge"),
			col("required_date", "Required", "m.required_date", "", "date"),
			col("purpose", "Purpose", "m.purpose", "", "text"),
		},
		Joins: []string{"LEFT JOIN departments d ON d.id = m.department_id", "LEFT JOIN users u ON u.id = m.requested_by"},
	},
	{
		Key: "material_issues", Label: "Material issue", LabelPlural: "Material issues", Group: "Production & materials",
		Description: "Material issued from the raw material store.",
		Icon:        "materials", Table: "material_issues", Editable: false,
		OrderBy: "m.issued_date", OrderDesc: true,
		Search: []string{"m.issue_number"},
		Columns: []Column{
			col("issue_number", "Issue number", "m.issue_number", "", "text"),
			col("mr_number", "MR number", "mr.mr_number", "", "text"),
			col("issued_by", "Issued by", "u.full_name", "", "text"),
			col("issued_to", "Issued to", "u2.full_name", "", "text"),
			col("issued_date", "Issued", "m.issued_date", "", "date"),
			col("acknowledged", "Acknowledged", "m.acknowledged_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN material_requisitions mr ON mr.id = m.mr_id", "LEFT JOIN users u ON u.id = m.issued_by", "LEFT JOIN users u2 ON u2.id = m.issued_to"},
	},
	{
		Key: "production_receipts", Label: "Production receipt", LabelPlural: "Production receipts", Group: "Production & materials",
		Description: "Finished goods received from production.",
		Icon:        "production", Table: "production_receipts", Editable: false, SoftDelete: true,
		OrderBy: "m.receipt_number", OrderDesc: true,
		Search: []string{"m.receipt_number"},
		Columns: []Column{
			col("receipt_number", "Receipt number", "m.receipt_number", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("recorded_by", "Recorded by", "u.full_name", "", "text"),
			col("receipt_date", "Received", "m.receipt_date", "", "date"),
		},
		Joins: []string{"LEFT JOIN warehouses w ON w.id = m.warehouse_id", "LEFT JOIN users u ON u.id = m.recorded_by"},
	},

	// ---------------- Sales ----------------
	{
		Key: "sales_orders", Label: "Sales order", LabelPlural: "Sales orders", Group: "Sales",
		Description: "Customer orders and their dispatch state.",
		Icon:        "sales", Table: "sales_orders", Editable: false, SoftDelete: true,
		OrderBy: "m.order_number", OrderDesc: true,
		Search: []string{"m.order_number"},
		Columns: []Column{
			col("order_number", "Order number", "m.order_number", "", "text"),
			col("customer", "Customer", "cu.name", "", "text"),
			col("status", "Status", "m.status", "", "badge"),
			col("order_date", "Order date", "m.order_date", "", "date"),
			col("required_date", "Required", "m.required_date", "", "date"),
			col("total_amount", "Total", "m.total_amount", "right", "number"),
		},
		Joins: []string{"LEFT JOIN customers cu ON cu.id = m.customer_id"},
	},
	{
		Key: "delivery_notes", Label: "Delivery note", LabelPlural: "Delivery notes", Group: "Sales",
		Description: "Dispatch documents for sales orders.",
		Icon:        "sales", Table: "delivery_notes", Editable: false, SoftDelete: true,
		OrderBy: "m.delivery_number", OrderDesc: true,
		Search: []string{"m.delivery_number"},
		Columns: []Column{
			col("delivery_number", "Delivery number", "m.delivery_number", "", "text"),
			col("order_number", "Order number", "so.order_number", "", "text"),
			col("delivered_by", "Delivered by", "u.full_name", "", "text"),
			col("recipient_name", "Recipient", "m.recipient_name", "", "text"),
			col("delivery_date", "Delivered", "m.delivery_date", "", "date"),
		},
		Joins: []string{"LEFT JOIN sales_orders so ON so.id = m.sales_order_id", "LEFT JOIN users u ON u.id = m.delivered_by"},
	},

	// ---------------- Inventory ----------------
	{
		Key: "stock_balances", Label: "Stock balance", LabelPlural: "Stock balances", Group: "Inventory",
		Description: "The live quantity of every item in every warehouse.",
		Icon:        "inventory", Table: "stock_balances", Editable: false,
		OrderBy: "i.name",
		Search: []string{"i.code", "i.name"},
		Columns: []Column{
			col("item", "Item", "i.name", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("quantity", "On hand", "m.quantity", "right", "number"),
			col("reserved_qty", "Reserved", "m.reserved_qty", "right", "number"),
			col("available_qty", "Available", "m.available_qty", "right", "number"),
			col("updated_at", "Updated", "m.updated_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN items i ON i.id = m.item_id", "LEFT JOIN warehouses w ON w.id = m.warehouse_id"},
	},
	{
		Key: "stock_batches", Label: "Stock batch", LabelPlural: "Stock batches", Group: "Inventory",
		Description: "Batch, expiry and cost of each lot on hand.",
		Icon:        "inventory", Table: "stock_batches", Editable: false,
		OrderBy: "m.created_at", OrderDesc: true,
		Search: []string{"m.batch_number"},
		Columns: []Column{
			col("batch_number", "Batch", "m.batch_number", "", "text"),
			col("item", "Item", "i.name", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("quantity", "Quantity", "m.quantity", "right", "number"),
			col("expiry_date", "Expiry", "m.expiry_date", "", "date"),
			col("unit_cost", "Unit cost", "m.unit_cost", "right", "number"),
		},
		Joins: []string{"LEFT JOIN items i ON i.id = m.item_id", "LEFT JOIN warehouses w ON w.id = m.warehouse_id"},
	},
	{
		Key: "stock_movements", Label: "Stock movement", LabelPlural: "Stock movements", Group: "Inventory",
		Description: "Every receipt, issue and movement. This is the audit trail.",
		Icon:        "clock", Table: "stock_movements", Editable: false,
		OrderBy: "m.movement_date", OrderDesc: true,
		Search: []string{"m.movement_type", "m.notes"},
		Columns: []Column{
			col("movement_type", "Movement", "m.movement_type", "", "badge"),
			col("item", "Item", "i.name", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("batch", "Batch", "b.batch_number", "", "text"),
			col("quantity", "Quantity", "m.quantity", "right", "number"),
			col("unit_cost", "Unit cost", "m.unit_cost", "right", "number"),
			col("movement_date", "Date", "m.movement_date", "", "date"),
		},
		Joins: []string{
			"LEFT JOIN items i ON i.id = m.item_id",
			"LEFT JOIN warehouses w ON w.id = m.warehouse_id",
			"LEFT JOIN stock_batches b ON b.id = m.batch_id",
		},
	},
	{
		Key: "inventory_alerts", Label: "Inventory alert", LabelPlural: "Inventory alerts", Group: "Inventory",
		Description: "Low-stock and expiry alerts raised by the background scanner.",
		Icon:        "bell", Table: "inventory_alerts", Editable: false,
		OrderBy: "m.created_at", OrderDesc: true,
		Search: []string{"m.message"},
		Columns: []Column{
			col("alert_type", "Type", "m.alert_type", "", "badge"),
			col("item", "Item", "i.name", "", "text"),
			col("warehouse", "Warehouse", "w.name", "", "text"),
			col("message", "Message", "m.message", "", "text"),
			col("active", "Active", "m.active", "", "boolean"),
			col("created_at", "Raised", "m.created_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN items i ON i.id = m.item_id", "LEFT JOIN warehouses w ON w.id = m.warehouse_id"},
	},

	// ---------------- System ----------------
	{
		Key: "notifications", Label: "Notification", LabelPlural: "Notifications", Group: "System",
		Description: "In-app notifications raised for each user.",
		Icon:        "bell", Table: "notifications", Editable: false, SoftDelete: true,
		OrderBy: "m.created_at", OrderDesc: true,
		Search: []string{"m.title", "m.message"},
		Columns: []Column{
			col("title", "Title", "m.title", "", "text"),
			col("recipient", "Recipient", "u.full_name", "", "text"),
			col("entity_type", "Entity", "m.entity_type", "", "badge"),
			col("read_at", "Read", "m.read_at", "", "datetime"),
			col("created_at", "Created", "m.created_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN users u ON u.id = m.user_id"},
	},
	{
		Key: "status_history", Label: "Status history", LabelPlural: "Status history", Group: "System",
		Description: "Every workflow status change, with who made it and why.",
		Icon:        "clock", Table: "status_history", Editable: false,
		OrderBy: "m.created_at", OrderDesc: true,
		Search: []string{"m.entity_type", "m.new_status", "m.reason"},
		Columns: []Column{
			col("entity_type", "Entity", "m.entity_type", "", "badge"),
			col("old_status", "From", "m.old_status", "", "text"),
			col("new_status", "To", "m.new_status", "", "text"),
			col("changed_by", "Changed by", "u.full_name", "", "text"),
			col("reason", "Reason", "m.reason", "", "text"),
			col("created_at", "When", "m.created_at", "", "datetime"),
		},
		Joins: []string{"LEFT JOIN users u ON u.id = m.changed_by"},
	},
}

// Registry returns every registered model.
func Registry() []Model {
	out := make([]Model, len(models))
	copy(out, models)
	return out
}

// Find returns the model registered under key.
func Find(key string) (Model, bool) {
	for _, m := range models {
		if m.Key == key {
			return m, true
		}
	}
	return Model{}, false
}

// Groups returns the registry grouped for navigation, preserving the order in
// which groups first appear.
func Groups() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range models {
		if !seen[m.Group] {
			seen[m.Group] = true
			out = append(out, m.Group)
		}
	}
	return out
}
