package notification

import (
	"net/http"
	"strconv"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/read-all", h.MarkAllRead)
	r.Post("/{id}/read", h.MarkRead)
	return r
}

func requestUserID(r *http.Request) (uuid.UUID, error) {
	value, _ := r.Context().Value(middleware.UserIDKey).(string)
	return uuid.Parse(value)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := requestUserID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	unread := r.URL.Query().Get("unread") == "true"
	items, total, err := h.service.List(r.Context(), userID, unread, page, limit)
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
	response.JSONWithMeta(w, http.StatusOK, items, &response.Meta{Page: page, Limit: limit, Total: total})
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	userID, err := requestUserID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid notification ID", nil)
		return
	}
	if err = h.service.MarkRead(r.Context(), userID, id); err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}

func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, err := requestUserID(r)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return
	}
	if err = h.service.MarkAllRead(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "success"})
}
