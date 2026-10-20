package material

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/submit", h.Submit)
	r.Post("/{id}/pm-approve", h.PMApprove)
	r.Post("/{id}/pm-reject", h.PMReject)
	r.Get("/{id}/history", h.GetHistory)

	r.Post("/issues", h.CreateIssue)
	r.Get("/issues/{id}", h.GetIssueByID)
	r.Post("/issues/{id}/acknowledge", h.AcknowledgeIssue)
	r.Get("/{id}/issues", h.ListIssuesByMR)

	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "department_head") {
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}

	var req CreateMRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	mr, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusCreated, mr)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, parseErr := uuid.Parse(userIDStr)
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if parseErr != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	allowed, accessErr := h.service.CanView(r.Context(), id, userID, role)
	if accessErr != nil || !allowed {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You cannot view this requisition", nil)
		return
	}

	mr, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Material requisition not found", nil)
		return
	}

	response.JSON(w, http.StatusOK, mr)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, _ := uuid.Parse(userIDStr)
	role, _ := r.Context().Value(middleware.RoleKey).(string)

	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	list, total, err := h.service.List(r.Context(), userID, role, status, page, limit)
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

	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	h.handleAction(w, r, h.service.Submit)
}

func (h *Handler) PMApprove(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "production_manager") {
		return
	}
	h.handleAction(w, r, h.service.PMApprove)
}

func (h *Handler) PMReject(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "production_manager") {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, _ := uuid.Parse(userIDStr)

	var req ActionRequest
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "A JSON reason is required", nil)
		return
	}

	if err := h.service.PMReject(r.Context(), id, userID, req.Reason); err != nil {
		response.Error(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *Handler) handleAction(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID) error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, _ := uuid.Parse(userIDStr)

	if err := fn(r.Context(), id, userID); err != nil {
		response.Error(w, http.StatusBadRequest, "ACTION_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *Handler) CreateIssue(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, "store_keeper") {
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}

	var req CreateIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	issue, err := h.service.CreateIssue(r.Context(), userID, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusCreated, issue)
}

func (h *Handler) GetIssueByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, parseErr := uuid.Parse(userIDStr)
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if parseErr != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	allowed, accessErr := h.service.CanViewIssue(r.Context(), id, userID, role)
	if accessErr != nil || !allowed {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You cannot view this material issue", nil)
		return
	}

	issue, err := h.service.GetIssueByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Material issue not found", nil)
		return
	}

	response.JSON(w, http.StatusOK, issue)
}

func (h *Handler) ListIssuesByMR(w http.ResponseWriter, r *http.Request) {
	mrID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, parseErr := uuid.Parse(userIDStr)
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if parseErr != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	allowed, accessErr := h.service.CanListIssues(r.Context(), mrID, userID, role)
	if accessErr != nil || !allowed {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You cannot view issues for this requisition", nil)
		return
	}

	list, err := h.service.ListIssuesByMR(r.Context(), mrID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, parseErr := uuid.Parse(userIDStr)
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if parseErr != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	allowed, accessErr := h.service.CanView(r.Context(), id, userID, role)
	if accessErr != nil || !allowed {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "You cannot view this requisition history", nil)
		return
	}
	history, err := h.service.GetHistory(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "HISTORY_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, history)
}

func (h *Handler) AcknowledgeIssue(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}
	userIDStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	if err = h.service.AcknowledgeIssue(r.Context(), id, userID); err != nil {
		response.Error(w, http.StatusBadRequest, "ACKNOWLEDGE_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "receipt acknowledged"})
}

func requireRole(w http.ResponseWriter, r *http.Request, allowed string) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role == allowed || role == "admin" {
		return true
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot perform this action", nil)
	return false
}
