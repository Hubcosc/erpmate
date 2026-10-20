package report

import (
	"net/http"
	"time"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/Ethio-ERP/erpmate/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/summary", h.Summary)
	r.Get("/sales", h.Sales)
	r.Get("/material-consumption", h.MaterialConsumption)
	r.Get("/purchase-commitments", h.PurchaseCommitments)
	return r
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	if !reportRole(w, r, "finance", "production_manager", "sales_head", "store_keeper", "production_store_keeper", "admin") {
		return
	}
	period, err := requestedPeriod(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
		return
	}
	result, err := h.repo.Summary(r.Context(), period)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "REPORT_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) Sales(w http.ResponseWriter, r *http.Request) {
	if !reportRole(w, r, "finance", "sales", "sales_head", "admin") {
		return
	}
	period, err := requestedPeriod(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
		return
	}
	groupBy := r.URL.Query().Get("group_by")
	if groupBy == "" {
		groupBy = "product"
	}
	if groupBy != "product" && groupBy != "customer" {
		response.Error(w, http.StatusBadRequest, "INVALID_GROUP", "group_by must be product or customer", nil)
		return
	}
	result, err := h.repo.Sales(r.Context(), period, groupBy)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "REPORT_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) MaterialConsumption(w http.ResponseWriter, r *http.Request) {
	if !reportRole(w, r, "production_manager", "store_keeper", "admin") {
		return
	}
	period, err := requestedPeriod(r)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), nil)
		return
	}
	result, err := h.repo.MaterialConsumption(r.Context(), period)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "REPORT_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *Handler) PurchaseCommitments(w http.ResponseWriter, r *http.Request) {
	if !reportRole(w, r, "finance", "production_manager", "admin") {
		return
	}
	result, err := h.repo.PurchaseCommitments(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "REPORT_FAILED", err.Error(), nil)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func requestedPeriod(r *http.Request) (Period, error) {
	now := time.Now().UTC()
	period := Period{From: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC), To: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
	if raw := r.URL.Query().Get("from"); raw != "" {
		value, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return Period{}, err
		}
		period.From = value
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		value, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return Period{}, err
		}
		period.To = value
	}
	if period.From.After(period.To) {
		return Period{}, &periodError{}
	}
	return period, nil
}

type periodError struct{}

func (*periodError) Error() string { return "from must be on or before to; dates must use YYYY-MM-DD" }

func reportRole(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	role, _ := r.Context().Value(middleware.RoleKey).(string)
	for _, value := range allowed {
		if role == value {
			return true
		}
	}
	response.Error(w, http.StatusForbidden, "FORBIDDEN", "Your role cannot access this report", nil)
	return false
}
