package purchase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

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

type actionFunc func(ctx context.Context, actor Actor, id uuid.UUID, reason *string) (*PurchaseRequisition, error)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/summary", h.Summary)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{id}/history", h.GetHistory)

	r.Post("/{id}/submit", h.action(h.service.Submit, RoleDepartmentHead))
	r.Post("/{id}/pm-approve", h.action(h.service.PMApprove, RoleProductionManager))
	r.Post("/{id}/pm-reject", h.action(h.service.PMReject, RoleProductionManager))
	r.Post("/{id}/pm-return", h.action(h.service.PMReturn, RoleProductionManager))
	r.Post("/{id}/finance-approve", h.action(h.service.FinanceApprove, RoleFinance))
	r.Post("/{id}/finance-suspend", h.action(h.service.FinanceSuspend, RoleFinance))
	r.Post("/{id}/finance-reject", h.action(h.service.FinanceReject, RoleFinance))

	return r
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, RoleDepartmentHead) {
		return
	}
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	var req CreatePRRequest
	if err := decodeBody(w, r, &req, false); err != nil {
		h.writeError(w, r, err)
		return
	}
	pr, err := h.service.Create(r.Context(), actor, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, pr)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	pr, err := h.service.Get(r.Context(), actor, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, pr)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	filter, err := parseListFilter(r)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	list, total, err := h.service.List(r.Context(), actor, filter)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSONWithMeta(w, http.StatusOK, list, &response.Meta{
		Page:  filter.Page,
		Limit: filter.Limit,
		Total: total,
	})
}

// Summary returns status counts for dashboard tabs and badges.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	sum, err := h.service.Summary(r.Context(), actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, sum)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, RoleDepartmentHead) {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	var req UpdatePRRequest
	if err := decodeBody(w, r, &req, false); err != nil {
		h.writeError(w, r, err)
		return
	}
	pr, err := h.service.Update(r.Context(), actor, id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, pr)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !requireRole(w, r, RoleDepartmentHead) {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), actor, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	actor, ok := actorFrom(w, r)
	if !ok {
		return
	}
	history, err := h.service.GetHistory(r.Context(), actor, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, history)
}

// action builds a handler for a workflow transition. The JSON body is optional:
// reject / return / suspend require a "reason" (enforced by the service), the
// other actions accept it as an optional comment. The response is the updated
// requisition, including available_actions, so the UI can refresh in one call.
func (h *Handler) action(fn actionFunc, role string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireRole(w, r, role) {
			return
		}
		id, ok := parseID(w, r)
		if !ok {
			return
		}
		actor, ok := actorFrom(w, r)
		if !ok {
			return
		}
		var req ActionRequest
		if err := decodeBody(w, r, &req, true); err != nil {
			h.writeError(w, r, err)
			return
		}
		pr, err := fn(r.Context(), actor, id, req.Reason)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		response.JSON(w, http.StatusOK, pr)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// writeError maps domain errors to HTTP codes. Unknown errors are logged and
// reported generically, so database details never reach the client.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var perr *Error
	if errors.As(err, &perr) {
		switch perr.Kind {
		case KindValidation:
			response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", perr.Message, nil)
		case KindNotFound:
			response.Error(w, http.StatusNotFound, "NOT_FOUND", perr.Message, nil)
		case KindForbidden:
			response.Error(w, http.StatusForbidden, "FORBIDDEN", perr.Message, nil)
		case KindConflict:
			response.Error(w, http.StatusConflict, "CONFLICT", perr.Message, nil)
		default:
			response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong. Please try again.", nil)
		}
		return
	}
	slog.ErrorContext(r.Context(), "purchase requisition request failed",
		"method", r.Method, "path", r.URL.Path, "error", err)
	response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong. Please try again.", nil)
}

func actorFrom(w http.ResponseWriter, r *http.Request) (Actor, bool) {
	idStr, _ := r.Context().Value(middleware.UserIDKey).(string)
	userID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user", nil)
		return Actor{}, false
	}
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	return Actor{UserID: userID, Role: role}, true
}

func parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid ID", nil)
		return uuid.Nil, false
	}
	return id, true
}

// decodeBody reads a JSON body (max 1 MB). When optional is true an empty body
// is accepted.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any, optional bool) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if optional && errors.Is(err, io.EOF) {
			return nil
		}
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return Validation("request body is too large")
		}
		return Validation("invalid request body")
	}
	return nil
}

// requireRole allows the given roles; admin is always allowed.
func requireRole(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	if role == RoleAdmin {
		return true
	}
	for _, a := range allowed {
		if role == a {
			return true
		}
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot perform this action", nil)
	return false
}

func parseDate(field, raw string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, Validation(fmt.Sprintf("%s must use the format YYYY-MM-DD", field))
	}
	return &t, nil
}

func parseListFilter(r *http.Request) (ListFilter, error) {
	q := r.URL.Query()
	f := ListFilter{
		Search:  strings.TrimSpace(q.Get("q")),
		SortBy:  "created_at",
		SortDir: "desc",
		Page:    1,
		Limit:   20,
	}
	if len([]rune(f.Search)) > 100 {
		return f, Validation("search text is too long")
	}

	for _, s := range strings.Split(q.Get("status"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !IsValidStatus(s) {
			return f, Validation(fmt.Sprintf("invalid status %q", s))
		}
		f.Statuses = append(f.Statuses, s)
	}

	if raw := q.Get("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return f, Validation("invalid department_id")
		}
		f.DepartmentID = &id
	}
	if raw := q.Get("from"); raw != "" {
		t, err := parseDate("from", raw)
		if err != nil {
			return f, err
		}
		f.DateFrom = t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := parseDate("to", raw)
		if err != nil {
			return f, err
		}
		f.DateTo = t
	}
	if f.DateFrom != nil && f.DateTo != nil && f.DateTo.Before(*f.DateFrom) {
		return f, Validation("'to' date cannot be before 'from' date")
	}
	if raw := q.Get("mine"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return f, Validation("mine must be true or false")
		}
		f.MineOnly = b
	}
	if raw := q.Get("sort_by"); raw != "" {
		if _, ok := sortColumns[raw]; !ok {
			return f, Validation("invalid sort_by")
		}
		f.SortBy = raw
	}
	if raw := strings.ToLower(q.Get("order")); raw != "" {
		if raw != "asc" && raw != "desc" {
			return f, Validation("order must be asc or desc")
		}
		f.SortDir = raw
	}
	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return f, Validation("page must be a positive number")
		}
		f.Page = n
	}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return f, Validation("limit must be a positive number")
		}
		if n > 100 {
			n = 100
		}
		f.Limit = n
	}
	return f, nil
}
