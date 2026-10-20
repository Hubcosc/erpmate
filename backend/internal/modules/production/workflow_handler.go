package production

import (
	"encoding/json"
	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"strconv"
)

type WorkflowHandler struct{ service *WorkflowService }

func NewWorkflowHandler(service *WorkflowService) *WorkflowHandler {
	return &WorkflowHandler{service: service}
}
func (h *WorkflowHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	return r
}
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !allowProductionRole(w, r, "production_store_keeper", "admin") {
		return
	}
	actor, ok := productionActor(w, r)
	if !ok {
		return
	}
	var req CreateReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", nil)
		return
	}
	receipt, err := h.service.Create(r.Context(), actor, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusCreated, receipt)
}
func (h *WorkflowHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if !allowProductionRole(w, r, "production_store_keeper", "production_manager", "sales", "sales_head", "admin") {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid production receipt ID", nil)
		return
	}
	receipt, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Production receipt not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, receipt)
}
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	if !allowProductionRole(w, r, "production_store_keeper", "production_manager", "sales", "sales_head", "admin") {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, total, err := h.service.List(r.Context(), page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{Page: page, Limit: limit, Total: total})
}
func productionActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw, _ := r.Context().Value(middleware.UserIDKey).(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authenticated user", nil)
		return uuid.Nil, false
	}
	return id, true
}
func allowProductionRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	for _, candidate := range roles {
		if role == candidate {
			return true
		}
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot perform this action", nil)
	return false
}
