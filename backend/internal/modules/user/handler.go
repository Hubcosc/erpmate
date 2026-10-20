package user

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
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
	r.Get("/roles", h.ListRoles)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Put("/{id}/password", h.ChangePassword)
	r.Delete("/{id}", h.Delete)

	return r
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Only admin can create users
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only admin can create users", nil)
		return
	}

	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	user, err := h.service.Create(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusCreated, user)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "User not found", nil)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	var roleID *uuid.UUID
	if rid := r.URL.Query().Get("role_id"); rid != "" {
		if parsed, err := uuid.Parse(rid); err == nil {
			roleID = &parsed
		}
	}

	list, total, err := h.service.List(r.Context(), search, roleID, page, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only admin can update users", nil)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	user, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, user)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}

	if err := h.service.ChangePassword(r.Context(), id, req.NewPassword); err != nil {
		response.Error(w, http.StatusBadRequest, "PASSWORD_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "password updated successfully"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role != "admin" {
		response.Error(w, http.StatusForbidden, "FORBIDDEN", "Only admin can delete users", nil)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "user deleted successfully"})
}

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error(), nil)
		return
	}

	response.JSON(w, http.StatusOK, roles)
}