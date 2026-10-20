package procurement

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20 // 1 MiB

type WorkflowHandler struct{ service *Service }

func NewWorkflowHandler(service *Service) *WorkflowHandler { return &WorkflowHandler{service: service} }

// PurchaseOrderRoutes is meant to be mounted at /purchase-orders.
func (h *WorkflowHandler) PurchaseOrderRoutes() chi.Router {
	r := chi.NewRouter()
	r.With(requireRoles("purchasing", "finance", "admin")).Get("/", h.ListPurchaseOrders)
	r.With(requireRoles("purchasing", "admin")).Post("/from-requisition/{id}", h.CreateFromPR)
	r.With(requireRoles("purchasing", "finance", "admin")).Get("/{id}", h.GetPurchaseOrder)
	r.With(requireRoles("purchasing", "admin")).Put("/{id}", h.UpdatePurchaseOrder)
	r.With(requireRoles("purchasing", "admin")).Patch("/{id}", h.UpdatePurchaseOrder)
	r.With(requireRoles("purchasing", "admin")).Post("/{id}/issue", h.IssuePurchaseOrder)
	return r
}

// GRNRoutes is meant to be mounted at /goods-receipts.
func (h *WorkflowHandler) GRNRoutes() chi.Router {
	r := chi.NewRouter()
	r.With(requireRoles("store_keeper", "purchasing", "finance", "admin")).Get("/", h.ListGRNs)
	r.With(requireRoles("store_keeper", "admin")).Post("/", h.CreateGRN)
	r.With(requireRoles("store_keeper", "purchasing", "finance", "admin")).Get("/{id}", h.GetGRN)
	return r
}

// ---------- purchase orders ----------

func (h *WorkflowHandler) CreateFromPR(w http.ResponseWriter, r *http.Request) {
	actor, ok := requestActor(w, r)
	if !ok {
		return
	}
	prID, ok := pathID(w, r, "purchase requisition")
	if !ok {
		return
	}
	po, created, err := h.service.CreateFromApprovedPR(r.Context(), prID, actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusOK // already existed: idempotent
	if created {
		status = http.StatusCreated
	}
	response.JSON(w, status, po)
}

func (h *WorkflowHandler) GetPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "purchase order")
	if !ok {
		return
	}
	po, err := h.service.GetPurchaseOrder(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, po)
}

// ListPurchaseOrders supports ?status=&supplier_id=&search=&page=&limit=
func (h *WorkflowHandler) ListPurchaseOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := PurchaseOrderFilter{Status: q.Get("status"), Search: q.Get("search")}
	if raw := q.Get("supplier_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid supplier_id", nil)
			return
		}
		f.SupplierID = &id
	}
	page, limit := NormalizePaging(atoi(q.Get("page")), atoi(q.Get("limit")))
	list, total, err := h.service.ListPurchaseOrders(r.Context(), f, page, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{Page: page, Limit: limit, Total: total})
}

func (h *WorkflowHandler) UpdatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := requestActor(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "purchase order")
	if !ok {
		return
	}
	var req UpdatePurchaseOrderRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	po, err := h.service.UpdateDraftPurchaseOrder(r.Context(), id, actor, req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, po)
}

func (h *WorkflowHandler) IssuePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	actor, ok := requestActor(w, r)
	if !ok {
		return
	}
	id, ok := pathID(w, r, "purchase order")
	if !ok {
		return
	}
	po, err := h.service.IssuePurchaseOrder(r.Context(), id, actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, po)
}

// ---------- goods receipt notes ----------

func (h *WorkflowHandler) CreateGRN(w http.ResponseWriter, r *http.Request) {
	actor, ok := requestActor(w, r)
	if !ok {
		return
	}
	var req CreateGoodsReceiptNoteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	grn, err := h.service.CreateGRN(r.Context(), actor, req)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, grn)
}

func (h *WorkflowHandler) GetGRN(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "goods receipt note")
	if !ok {
		return
	}
	grn, err := h.service.GetGRN(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, grn)
}

// ListGRNs supports ?purchase_order_id=&page=&limit=
func (h *WorkflowHandler) ListGRNs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var poID *uuid.UUID
	if raw := q.Get("purchase_order_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid purchase_order_id", nil)
			return
		}
		poID = &id
	}
	page, limit := NormalizePaging(atoi(q.Get("page")), atoi(q.Get("limit")))
	list, total, err := h.service.ListGRNs(r.Context(), poID, page, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{Page: page, Limit: limit, Total: total})
}

// ---------- helpers ----------

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func pathID(w http.ResponseWriter, r *http.Request, what string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid "+what+" ID", nil)
		return uuid.Nil, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON body", nil)
		return false
	}
	return true
}

func requestActor(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw, _ := r.Context().Value(middleware.UserIDKey).(string)
	actor, err := uuid.Parse(raw)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authenticated user", nil)
		return uuid.Nil, false
	}
	return actor, true
}

func requireRoles(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(middleware.RoleKey).(string)
			for _, candidate := range roles {
				if role == candidate {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot perform this action", nil)
		})
	}
}

// writeError maps service errors to HTTP responses. Unknown errors (database,
// network...) are logged and returned as a generic 500 so internals never leak.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if ae, ok := AsAppError(err); ok {
		switch ae.Kind {
		case KindNotFound:
			response.Error(w, http.StatusNotFound, "NOT_FOUND", ae.Msg, nil)
		case KindConflict:
			response.Error(w, http.StatusConflict, "CONFLICT", ae.Msg, nil)
		default:
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", ae.Msg, nil)
		}
		return
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		response.Error(w, http.StatusRequestEntityTooLarge, "VALIDATION_ERROR", "Request body too large", nil)
		return
	}
	slog.Error("procurement request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong. Please try again.", nil)
}
