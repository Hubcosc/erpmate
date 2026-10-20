package master

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ====================== HELPERS ======================

func newResult[T any](items []T, total int64, p ListParams) ListResult[T] {
	return ListResult[T]{Items: items, Page: p.Page, Limit: p.Limit, Total: total}
}

// setPtr adds col to fields when the pointer was provided.
func setPtr[T any](f map[string]interface{}, col string, p *T) {
	if p != nil {
		f[col] = *p
	}
}

// setOpt handles absent / null / value for non-string nullable columns.
func setOpt[T any](f map[string]interface{}, col string, o Optional[T]) {
	if !o.Set {
		return
	}
	if o.Null {
		f[col] = nil
		return
	}
	f[col] = o.Value
}

// setOptString is setOpt for strings: trims, and treats "" as NULL.
func setOptString(f map[string]interface{}, col string, o Optional[string]) {
	if !o.Set {
		return
	}
	if o.Null {
		f[col] = nil
		return
	}
	if t := strings.TrimSpace(o.Value); t != "" {
		f[col] = t
	} else {
		f[col] = nil
	}
}

// setName validates and sets a required-when-present name field.
func setName(v *validator, f map[string]interface{}, p *string) {
	if p == nil {
		return
	}
	name := strings.TrimSpace(*p)
	if name == "" {
		v.add("name", "cannot be empty")
		return
	}
	f["name"] = name
}

func checkEmail(v *validator, email *string) {
	if e := trimPtr(email); e != nil && !validEmail(*e) {
		v.add("email", "is not a valid email address")
	}
}

func checkOptEmail(v *validator, o Optional[string]) {
	if !o.Set || o.Null {
		return
	}
	if e := strings.TrimSpace(o.Value); e != "" && !validEmail(e) {
		v.add("email", "is not a valid email address")
	}
}

// categoryExists returns false (not an error) when the category is missing.
func (s *Service) categoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	_, err := s.repo.GetCategoryByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// ====================== DEPARTMENTS ======================

func (s *Service) CreateDepartment(ctx context.Context, req CreateDepartmentRequest) (*Department, error) {
	v := &validator{}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		v.add("name", "is required")
	}
	if err := v.err(); err != nil {
		return nil, err
	}

	d := &Department{ID: uuid.New(), Name: name, Code: trimPtr(req.Code), IsActive: true}
	if err := s.repo.CreateDepartment(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Service) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}

func (s *Service) ListDepartments(ctx context.Context, p ListParams) (ListResult[Department], error) {
	p.normalize()
	list, total, err := s.repo.ListDepartments(ctx, p)
	if err != nil {
		return ListResult[Department]{}, err
	}
	return newResult(list, total, p), nil
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (*Department, error) {
	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)
	setOptString(fields, "code", req.Code)
	setPtr(fields, "is_active", req.IsActive)
	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateDepartment(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetDepartmentByID(ctx, id)
}

func (s *Service) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDepartment(ctx, id)
}

// ====================== CATEGORIES ======================

func (s *Service) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*ItemCategory, error) {
	v := &validator{}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		v.add("name", "is required")
	}
	if req.ParentID != nil {
		ok, err := s.categoryExists(ctx, *req.ParentID)
		if err != nil {
			return nil, err
		}
		if !ok {
			v.add("parent_id", "parent category not found")
		}
	}
	if err := v.err(); err != nil {
		return nil, err
	}

	c := &ItemCategory{ID: uuid.New(), Name: name, ParentID: req.ParentID}
	if err := s.repo.CreateCategory(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetCategory(ctx context.Context, id uuid.UUID) (*ItemCategory, error) {
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) ListCategories(ctx context.Context, p ListParams) (ListResult[ItemCategory], error) {
	p.normalize()
	list, total, err := s.repo.ListCategories(ctx, p)
	if err != nil {
		return ListResult[ItemCategory]{}, err
	}
	return newResult(list, total, p), nil
}

func (s *Service) UpdateCategory(ctx context.Context, id uuid.UUID, req UpdateCategoryRequest) (*ItemCategory, error) {
	if _, err := s.repo.GetCategoryByID(ctx, id); err != nil {
		return nil, err
	}

	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)

	if req.ParentID.Set {
		if req.ParentID.Null {
			fields["parent_id"] = nil
		} else {
			parentID := req.ParentID.Value
			ok, err := s.categoryExists(ctx, parentID)
			if err != nil {
				return nil, err
			}
			if !ok {
				v.add("parent_id", "parent category not found")
			} else if cycle, err := s.repo.WouldCreateCycle(ctx, id, parentID); err != nil {
				return nil, err
			} else if cycle {
				v.add("parent_id", "a category cannot be its own parent or a descendant's child")
			} else {
				fields["parent_id"] = parentID
			}
		}
	}

	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCategory(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetCategoryByID(ctx, id)
}

func (s *Service) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	if _, err := s.repo.GetCategoryByID(ctx, id); err != nil {
		return err
	}
	hasChildren, err := s.repo.CategoryHasChildren(ctx, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return conflict("category has sub-categories; move or delete them first")
	}
	hasItems, err := s.repo.CategoryHasItems(ctx, id)
	if err != nil {
		return err
	}
	if hasItems {
		return conflict("category is used by items; reassign those items first")
	}
	return s.repo.DeleteCategory(ctx, id)
}

// ====================== ITEMS ======================

func (s *Service) CreateItem(ctx context.Context, req CreateItemRequest) (*Item, error) {
	v := &validator{}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	itemType := strings.TrimSpace(req.ItemType)
	uom := strings.TrimSpace(req.UnitOfMeasure)

	if code == "" {
		v.add("code", "is required")
	}
	if name == "" {
		v.add("name", "is required")
	}
	if uom == "" {
		v.add("unit_of_measure", "is required")
	}
	if itemType == "" {
		v.add("item_type", "is required")
	} else if !contains(ItemTypes, itemType) {
		v.add("item_type", "must be one of: "+strings.Join(ItemTypes, ", "))
	}
	if req.MinStockLevel < 0 {
		v.add("min_stock_level", "cannot be negative")
	}
	if req.MaxStockLevel != nil && *req.MaxStockLevel < req.MinStockLevel {
		v.add("max_stock_level", "cannot be less than min_stock_level")
	}
	if req.ReorderLevel != nil && *req.ReorderLevel < 0 {
		v.add("reorder_level", "cannot be negative")
	}
	if req.CategoryID != nil {
		ok, err := s.categoryExists(ctx, *req.CategoryID)
		if err != nil {
			return nil, err
		}
		if !ok {
			v.add("category_id", "category not found")
		}
	}
	if err := v.err(); err != nil {
		return nil, err
	}

	item := &Item{
		ID:              uuid.New(),
		Code:            code,
		Name:            name,
		Description:     trimPtr(req.Description),
		CategoryID:      req.CategoryID,
		ItemType:        itemType,
		UnitOfMeasure:   uom,
		MinStockLevel:   req.MinStockLevel,
		MaxStockLevel:   req.MaxStockLevel,
		ReorderLevel:    req.ReorderLevel,
		IsBatchTracked:  req.IsBatchTracked,
		IsExpiryTracked: req.IsExpiryTracked,
		IsActive:        true,
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetItemByID(ctx, item.ID) // re-read to include category_name
}

func (s *Service) GetItem(ctx context.Context, id uuid.UUID) (*Item, error) {
	return s.repo.GetItemByID(ctx, id)
}

func (s *Service) ListItems(ctx context.Context, p ItemListParams) (ListResult[Item], error) {
	p.normalize()
	p.ItemType = strings.TrimSpace(p.ItemType)
	list, total, err := s.repo.ListItems(ctx, p)
	if err != nil {
		return ListResult[Item]{}, err
	}
	return newResult(list, total, p.ListParams), nil
}

func (s *Service) UpdateItem(ctx context.Context, id uuid.UUID, req UpdateItemRequest) (*Item, error) {
	existing, err := s.repo.GetItemByID(ctx, id)
	if err != nil {
		return nil, err
	}

	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)
	setOptString(fields, "description", req.Description)
	setPtr(fields, "is_batch_tracked", req.IsBatchTracked)
	setPtr(fields, "is_expiry_tracked", req.IsExpiryTracked)
	setPtr(fields, "is_active", req.IsActive)

	if req.UnitOfMeasure != nil {
		uom := strings.TrimSpace(*req.UnitOfMeasure)
		if uom == "" {
			v.add("unit_of_measure", "cannot be empty")
		} else {
			fields["unit_of_measure"] = uom
		}
	}

	// Validate stock levels against the values the row will have after this update.
	minLevel := existing.MinStockLevel
	if req.MinStockLevel != nil {
		minLevel = *req.MinStockLevel
		if minLevel < 0 {
			v.add("min_stock_level", "cannot be negative")
		}
		fields["min_stock_level"] = minLevel
	}
	maxLevel := existing.MaxStockLevel
	if req.MaxStockLevel.Set {
		if req.MaxStockLevel.Null {
			maxLevel = nil
		} else {
			val := req.MaxStockLevel.Value
			maxLevel = &val
		}
		setOpt(fields, "max_stock_level", req.MaxStockLevel)
	}
	if maxLevel != nil && *maxLevel < minLevel {
		v.add("max_stock_level", "cannot be less than min_stock_level")
	}
	if req.ReorderLevel.Set && !req.ReorderLevel.Null && req.ReorderLevel.Value < 0 {
		v.add("reorder_level", "cannot be negative")
	}
	setOpt(fields, "reorder_level", req.ReorderLevel)

	if req.CategoryID.Set && !req.CategoryID.Null {
		ok, err := s.categoryExists(ctx, req.CategoryID.Value)
		if err != nil {
			return nil, err
		}
		if !ok {
			v.add("category_id", "category not found")
		}
	}
	setOpt(fields, "category_id", req.CategoryID)

	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetItemByID(ctx, id)
}

func (s *Service) DeleteItem(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteItem(ctx, id)
}

// ====================== SUPPLIERS ======================

func (s *Service) CreateSupplier(ctx context.Context, req CreateSupplierRequest) (*Supplier, error) {
	v := &validator{}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		v.add("name", "is required")
	}
	checkEmail(v, req.Email)
	if err := v.err(); err != nil {
		return nil, err
	}

	sup := &Supplier{
		ID:            uuid.New(),
		Code:          trimPtr(req.Code),
		Name:          name,
		ContactPerson: trimPtr(req.ContactPerson),
		Phone:         trimPtr(req.Phone),
		Email:         trimPtr(req.Email),
		Address:       trimPtr(req.Address),
		TinNumber:     trimPtr(req.TinNumber),
		IsActive:      true,
	}
	if err := s.repo.CreateSupplier(ctx, sup); err != nil {
		return nil, err
	}
	return sup, nil
}

func (s *Service) GetSupplier(ctx context.Context, id uuid.UUID) (*Supplier, error) {
	return s.repo.GetSupplierByID(ctx, id)
}

func (s *Service) ListSuppliers(ctx context.Context, p ListParams) (ListResult[Supplier], error) {
	p.normalize()
	list, total, err := s.repo.ListSuppliers(ctx, p)
	if err != nil {
		return ListResult[Supplier]{}, err
	}
	return newResult(list, total, p), nil
}

func (s *Service) UpdateSupplier(ctx context.Context, id uuid.UUID, req UpdateSupplierRequest) (*Supplier, error) {
	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)
	setOptString(fields, "code", req.Code)
	setOptString(fields, "contact_person", req.ContactPerson)
	setOptString(fields, "phone", req.Phone)
	setOptString(fields, "email", req.Email)
	setOptString(fields, "address", req.Address)
	setOptString(fields, "tin_number", req.TinNumber)
	setPtr(fields, "is_active", req.IsActive)
	checkOptEmail(v, req.Email)
	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSupplier(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetSupplierByID(ctx, id)
}

func (s *Service) DeleteSupplier(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSupplier(ctx, id)
}

// ====================== CUSTOMERS ======================

func (s *Service) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*Customer, error) {
	v := &validator{}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		v.add("name", "is required")
	}
	customerType := strings.TrimSpace(req.CustomerType)
	if customerType == "" {
		customerType = "local"
	}
	checkEmail(v, req.Email)
	if err := v.err(); err != nil {
		return nil, err
	}

	c := &Customer{
		ID:            uuid.New(),
		Code:          trimPtr(req.Code),
		Name:          name,
		CustomerType:  customerType,
		ContactPerson: trimPtr(req.ContactPerson),
		Phone:         trimPtr(req.Phone),
		Email:         trimPtr(req.Email),
		Address:       trimPtr(req.Address),
		TinNumber:     trimPtr(req.TinNumber),
		IsActive:      true,
	}
	if err := s.repo.CreateCustomer(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) GetCustomer(ctx context.Context, id uuid.UUID) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, id)
}

func (s *Service) ListCustomers(ctx context.Context, p ListParams) (ListResult[Customer], error) {
	p.normalize()
	list, total, err := s.repo.ListCustomers(ctx, p)
	if err != nil {
		return ListResult[Customer]{}, err
	}
	return newResult(list, total, p), nil
}

func (s *Service) UpdateCustomer(ctx context.Context, id uuid.UUID, req UpdateCustomerRequest) (*Customer, error) {
	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)
	if req.CustomerType != nil {
		ct := strings.TrimSpace(*req.CustomerType)
		if ct == "" {
			v.add("customer_type", "cannot be empty")
		} else {
			fields["customer_type"] = ct
		}
	}
	setOptString(fields, "code", req.Code)
	setOptString(fields, "contact_person", req.ContactPerson)
	setOptString(fields, "phone", req.Phone)
	setOptString(fields, "email", req.Email)
	setOptString(fields, "address", req.Address)
	setOptString(fields, "tin_number", req.TinNumber)
	setPtr(fields, "is_active", req.IsActive)
	checkOptEmail(v, req.Email)
	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateCustomer(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetCustomerByID(ctx, id)
}

func (s *Service) DeleteCustomer(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteCustomer(ctx, id)
}

// ====================== WAREHOUSES ======================

func (s *Service) CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (*Warehouse, error) {
	v := &validator{}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	whType := strings.TrimSpace(req.WarehouseType)

	if code == "" {
		v.add("code", "is required")
	}
	if name == "" {
		v.add("name", "is required")
	}
	if whType == "" {
		v.add("warehouse_type", "is required")
	} else if !contains(WarehouseTypes, whType) {
		v.add("warehouse_type", "must be one of: "+strings.Join(WarehouseTypes, ", "))
	}
	if err := v.err(); err != nil {
		return nil, err
	}

	w := &Warehouse{
		ID:            uuid.New(),
		Code:          code,
		Name:          name,
		WarehouseType: whType,
		Location:      trimPtr(req.Location),
		IsActive:      true,
	}
	if err := s.repo.CreateWarehouse(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) GetWarehouse(ctx context.Context, id uuid.UUID) (*Warehouse, error) {
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *Service) ListWarehouses(ctx context.Context, p ListParams) (ListResult[Warehouse], error) {
	p.normalize()
	list, total, err := s.repo.ListWarehouses(ctx, p)
	if err != nil {
		return ListResult[Warehouse]{}, err
	}
	return newResult(list, total, p), nil
}

func (s *Service) UpdateWarehouse(ctx context.Context, id uuid.UUID, req UpdateWarehouseRequest) (*Warehouse, error) {
	v := &validator{}
	fields := map[string]interface{}{}
	setName(v, fields, req.Name)
	if req.WarehouseType != nil {
		t := strings.TrimSpace(*req.WarehouseType)
		if !contains(WarehouseTypes, t) {
			v.add("warehouse_type", "must be one of: "+strings.Join(WarehouseTypes, ", "))
		} else {
			fields["warehouse_type"] = t
		}
	}
	setOptString(fields, "location", req.Location)
	setPtr(fields, "is_active", req.IsActive)
	if err := v.err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateWarehouse(ctx, id, fields); err != nil {
		return nil, err
	}
	return s.repo.GetWarehouseByID(ctx, id)
}

func (s *Service) DeleteWarehouse(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteWarehouse(ctx, id)
}
