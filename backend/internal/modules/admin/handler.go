package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler serves the administration API.
type Handler struct {
	service *Service
}

// NewHandler builds the admin handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the admin router. Every route requires the admin role.
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/models", h.Models)
	r.Get("/lookups", h.Lookups)
	r.Get("/overview", h.Overview)

	// Static segments are matched before the {model} parameter by chi, so
	// /admin/models never resolves to the "models" model.
	r.Get("/{model}", h.List)
	r.Post("/{model}", h.Create)
	r.Get("/{model}/{id}", h.Get)
	r.Put("/{model}/{id}", h.Update)
	r.Patch("/{model}/{id}", h.Update)
	r.Delete("/{model}/{id}", h.Delete)

	return r
}

// requireSuperAdmin rejects anyone who is not an administrator.
func (h *Handler) requireSuperAdmin(w http.ResponseWriter, r *http.Request) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role == "admin" {
		return true
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only a super administrator can use this area", nil)
	return false
}

// Models returns the registry so the frontend can render the forms.
func (h *Handler) Models(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"models": Registry(),
		"groups": Groups(),
	})
}

// Lookups returns the reference lists used by the admin forms.
func (h *Handler) Lookups(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	lookups, err := h.service.BuildLookups(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LOOKUPS_FAILED", "Could not load reference data", nil)
		return
	}
	response.JSON(w, http.StatusOK, lookups)
}

// Overview returns the number of records in every model.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	counts, err := h.service.Overview(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "OVERVIEW_FAILED", "Could not load the overview", nil)
		return
	}
	response.JSON(w, http.StatusOK, counts)
}

// model resolves the {model} path segment against the registry.
func (h *Handler) model(w http.ResponseWriter, r *http.Request) (Model, bool) {
	key := chi.URLParam(r, "model")
	m, ok := Find(key)
	if !ok {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Unknown model: "+key, nil)
		return Model{}, false
	}
	return m, true
}

// List returns a page of records.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	m, ok := h.model(w, r)
	if !ok {
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	items, total, err := h.service.List(r.Context(), m, search, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "Could not load the records", nil)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, items, &response.Meta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

// Get returns one record with all of its form fields.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	m, ok := h.model(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid id", nil)
		return
	}

	record, err := h.service.Get(r.Context(), m, id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Record not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, record)
}

// Create inserts a record.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	m, ok := h.model(w, r)
	if !ok {
		return
	}
	if !m.Editable {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", m.LabelPlural+" are managed by the workflow and cannot be created here", nil)
		return
	}

	var values map[string]any
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	record, err := h.service.Create(r.Context(), m, values)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusCreated, record)
}

// Update changes a record.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	m, ok := h.model(w, r)
	if !ok {
		return
	}
	if !m.Editable {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", m.LabelPlural+" are managed by the workflow and cannot be changed here", nil)
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid id", nil)
		return
	}

	var values map[string]any
	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	record, err := h.service.Update(r.Context(), m, id, values)
	if err != nil {
		status := http.StatusBadRequest
		code := "VALIDATION_ERROR"
		if err == ErrNotFound {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		response.Error(w, status, code, err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, record)
}

// Delete removes a record.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.requireSuperAdmin(w, r) {
		return
	}
	m, ok := h.model(w, r)
	if !ok {
		return
	}
	if !m.Editable {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", m.LabelPlural+" are managed by the workflow and cannot be deleted here", nil)
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid id", nil)
		return
	}

	if err := h.service.Delete(r.Context(), m, id); err != nil {
		status := http.StatusInternalServerError
		code := "DELETE_FAILED"
		if err == ErrNotFound {
			status = http.StatusNotFound
			code = "NOT_FOUND"
		}
		response.Error(w, status, code, err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}
