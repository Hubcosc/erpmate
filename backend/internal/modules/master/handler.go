package master

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes mounts full CRUD for every master entity:
//
//	GET    /{entity}        list  (?search=&page=&limit=&is_active=)
//	POST   /{entity}        create
//	GET    /{entity}/{id}   read
//	PUT    /{entity}/{id}   update (partial updates allowed)
//	PATCH  /{entity}/{id}   update (same handler)
//	DELETE /{entity}/{id}   soft delete (204 No Content)
//	GET    /lookups         enum values for frontend dropdowns
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	s := h.service

	r.Get("/lookups", h.Lookups)

	r.Route("/departments", func(r chi.Router) {
		update := makeUpdate("Department", s.UpdateDepartment)
		r.Get("/", makeList(parseListParams, s.ListDepartments))
		r.Post("/", makeCreate("Department", s.CreateDepartment))
		r.Get("/{id}", makeGet("Department", s.GetDepartment))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Department", s.DeleteDepartment))
	})

	r.Route("/categories", func(r chi.Router) {
		update := makeUpdate("Category", s.UpdateCategory)
		r.Get("/", makeList(parseListParams, s.ListCategories))
		r.Post("/", makeCreate("Category", s.CreateCategory))
		r.Get("/{id}", makeGet("Category", s.GetCategory))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Category", s.DeleteCategory))
	})

	r.Route("/items", func(r chi.Router) {
		update := makeUpdate("Item", s.UpdateItem)
		r.Get("/", makeList(parseItemListParams, s.ListItems))
		r.Post("/", makeCreate("Item", s.CreateItem))
		r.Get("/{id}", makeGet("Item", s.GetItem))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Item", s.DeleteItem))
	})

	r.Route("/suppliers", func(r chi.Router) {
		update := makeUpdate("Supplier", s.UpdateSupplier)
		r.Get("/", makeList(parseListParams, s.ListSuppliers))
		r.Post("/", makeCreate("Supplier", s.CreateSupplier))
		r.Get("/{id}", makeGet("Supplier", s.GetSupplier))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Supplier", s.DeleteSupplier))
	})

	r.Route("/customers", func(r chi.Router) {
		update := makeUpdate("Customer", s.UpdateCustomer)
		r.Get("/", makeList(parseListParams, s.ListCustomers))
		r.Post("/", makeCreate("Customer", s.CreateCustomer))
		r.Get("/{id}", makeGet("Customer", s.GetCustomer))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Customer", s.DeleteCustomer))
	})

	r.Route("/warehouses", func(r chi.Router) {
		update := makeUpdate("Warehouse", s.UpdateWarehouse)
		r.Get("/", makeList(parseListParams, s.ListWarehouses))
		r.Post("/", makeCreate("Warehouse", s.CreateWarehouse))
		r.Get("/{id}", makeGet("Warehouse", s.GetWarehouse))
		r.Put("/{id}", update)
		r.Patch("/{id}", update)
		r.Delete("/{id}", makeDelete("Warehouse", s.DeleteWarehouse))
	})

	return r
}

// Lookups returns the allowed enum values so the frontend never hard-codes them.
func (h *Handler) Lookups(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string][]string{
		"item_types":      ItemTypes,
		"warehouse_types": WarehouseTypes,
	})
}

// ====================== GENERIC HANDLER FACTORIES ======================

func makeCreate[Req, Res any](entity string, fn func(context.Context, Req) (*Res, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if !decodeBody(w, r, &req) {
			return
		}
		res, err := fn(r.Context(), req)
		if err != nil {
			fail(w, err, entity)
			return
		}
		response.JSON(w, http.StatusCreated, res)
	}
}

func makeGet[Res any](entity string, fn func(context.Context, uuid.UUID) (*Res, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		res, err := fn(r.Context(), id)
		if err != nil {
			fail(w, err, entity)
			return
		}
		response.JSON(w, http.StatusOK, res)
	}
}

func makeUpdate[Req, Res any](entity string, fn func(context.Context, uuid.UUID, Req) (*Res, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		var req Req
		if !decodeBody(w, r, &req) {
			return
		}
		res, err := fn(r.Context(), id, req)
		if err != nil {
			fail(w, err, entity)
			return
		}
		response.JSON(w, http.StatusOK, res)
	}
}

func makeDelete(entity string, fn func(context.Context, uuid.UUID) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		if err := fn(r.Context(), id); err != nil {
			fail(w, err, entity)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func makeList[P, Res any](
	parse func(*http.Request) (P, error),
	fn func(context.Context, P) (ListResult[Res], error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := parse(r)
		if err != nil {
			fail(w, err, "")
			return
		}
		res, err := fn(r.Context(), params)
		if err != nil {
			fail(w, err, "")
			return
		}
		response.JSONWithMeta(w, http.StatusOK, res.Items, &response.Meta{
			Page:  res.Page,
			Limit: res.Limit,
			Total: res.Total,
		})
	}
}

// ====================== REQUEST HELPERS ======================

const maxBodyBytes = 1 << 20 // 1 MiB

func decodeBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func parseListParams(r *http.Request) (ListParams, error) {
	q := r.URL.Query()
	p := ListParams{Search: strings.TrimSpace(q.Get("search"))}
	p.Page, _ = strconv.Atoi(q.Get("page"))
	p.Limit, _ = strconv.Atoi(q.Get("limit"))

	if v := q.Get("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return p, invalid("is_active", "must be true or false")
		}
		p.IsActive = &b
	}
	return p, nil
}

func parseItemListParams(r *http.Request) (ItemListParams, error) {
	base, err := parseListParams(r)
	if err != nil {
		return ItemListParams{}, err
	}
	p := ItemListParams{ListParams: base, ItemType: r.URL.Query().Get("item_type")}

	if v := r.URL.Query().Get("category_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return p, invalid("category_id", "must be a valid UUID")
		}
		p.CategoryID = &id
	}
	return p, nil
}

// ====================== ERROR MAPPING ======================

// fail maps typed errors to HTTP responses. Unknown errors are logged and
// returned as a generic 500 so internal details never reach the client.
func fail(w http.ResponseWriter, err error, entity string) {
	var (
		ve *ValidationError
		ce *ConflictError
	)
	switch {
	case errors.As(err, &ve):
		// Details is a field -> message map, e.g. {"name": "is required"}.
		response.Error(w, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "Validation failed", ve.Fields)
	case errors.Is(err, ErrNotFound):
		msg := "Resource not found"
		if entity != "" {
			msg = entity + " not found"
		}
		response.Error(w, http.StatusNotFound, "NOT_FOUND", msg, nil)
	case errors.As(err, &ce):
		response.Error(w, http.StatusConflict, "CONFLICT", ce.Message, nil)
	default:
		log.Printf("master: internal error: %v", err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong", nil)
	}
}
