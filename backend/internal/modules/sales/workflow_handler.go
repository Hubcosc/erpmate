package sales

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
func (h *WorkflowHandler) OrderRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListOrders)
	r.Post("/", h.CreateOrder)
	r.Get("/{id}", h.GetOrder)
	r.Post("/{id}/deliveries", h.CreateDelivery)
	return r
}
func (h *WorkflowHandler) DeliveryRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListDeliveries)
	r.Get("/{id}/pdf", h.GetDeliveryPDF)
	r.Get("/{id}", h.GetDelivery)
	return r
}
func (h *WorkflowHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "admin") {
		return
	}
	actor, ok := salesActor(w, r)
	if !ok {
		return
	}
	var req CreateSalesOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", nil)
		return
	}
	order, err := h.service.CreateOrder(r.Context(), actor, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusCreated, order)
}
func (h *WorkflowHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "finance", "admin") {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sales order ID", nil)
		return
	}
	o, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Sales order not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, o)
}
func (h *WorkflowHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "finance", "admin") {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, total, err := h.service.ListOrders(r.Context(), r.URL.Query().Get("status"), page, limit)
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
func (h *WorkflowHandler) CreateDelivery(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "admin") {
		return
	}
	actor, ok := salesActor(w, r)
	if !ok {
		return
	}
	orderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sales order ID", nil)
		return
	}
	var req CreateDeliveryNoteRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON body", nil)
		return
	}
	note, err := h.service.CreateDelivery(r.Context(), orderID, actor, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "DELIVERY_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusCreated, note)
}
func (h *WorkflowHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "finance", "admin") {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid delivery note ID", nil)
		return
	}
	note, err := h.service.GetDelivery(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Delivery note not found", nil)
		return
	}
	response.JSON(w, http.StatusOK, note)
}
func (h *WorkflowHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	if !salesRole(w, r, "sales", "sales_head", "finance", "admin") {
		return
	}
	var orderID *uuid.UUID
	if raw := r.URL.Query().Get("sales_order_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid sales_order_id", nil)
			return
		}
		orderID = &id
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, total, err := h.service.ListDeliveries(r.Context(), orderID, page, limit)
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
func salesActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw, _ := r.Context().Value(middleware.UserIDKey).(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authenticated user", nil)
		return uuid.Nil, false
	}
	return id, true
}
func salesRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	for _, candidate := range roles {
		if role == candidate {
			return true
		}
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot perform this action", nil)
	return false
}
