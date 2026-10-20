package inventory

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Ethio-ERP/erpmate/internal/common/response"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type StockHandler struct {
	svc *StockService
}

// NewStockHandler keeps the original signature so existing wiring does not change.
func NewStockHandler(repo *StockRepository) *StockHandler {
	return &StockHandler{svc: NewStockService(repo)}
}

// Routes:
//
//	GET /lookups                  filter values for dropdowns
//	GET /stock                    balances  (?search&item_type&warehouse_type&warehouse_id&category_id&status&sort&order&page&limit)
//	GET /stock/summary            status counts for tabs/KPI cards (same filters, status ignored)
//	GET /stock/{itemID}           one item across all warehouses, with totals
//	GET /stock/{itemID}/batches   FEFO batches, all warehouses unless filtered (?warehouse_type&warehouse_id&expiry_status)
//	GET /movements                ledger (?item_id&warehouse_id&movement_type&reference_type&search&date_from&date_to)
//	GET /movements/types          distinct movement types for the filter dropdown
//	GET /valuation                valuation lines (?search&item_type&warehouse_type&warehouse_id&cost_source&low_stock&sort&order)
//	GET /valuation/summary        total value, lines without cost, value by warehouse type
//	GET /alerts                   active alerts (?alert_type&severity&warehouse_id&search)
//	GET /alerts/summary           counts by type and severity
func (h *StockHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/lookups", h.Lookups)

	r.Get("/stock", h.ListBalances)
	r.Get("/stock/summary", h.StockSummary)
	r.Get("/stock/{itemID}", h.GetItemStock)
	r.Get("/stock/{itemID}/batches", h.ListBatches)

	r.Get("/movements", h.ListMovements)
	r.Get("/movements/types", h.MovementTypes)

	r.Get("/valuation", h.ListValuation)
	r.Get("/valuation/summary", h.ValuationSummary)

	r.Get("/alerts", h.ListAlerts)
	r.Get("/alerts/summary", h.AlertSummary)
	return r
}

func (h *StockHandler) Lookups(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string][]string{
		"stock_statuses":   StockStatuses,
		"alert_types":      AlertTypes,
		"alert_severities": AlertSeverities,
		"cost_sources":     CostSources,
		"expiry_statuses":  ExpiryStatuses,
	})
}

// ====================== STOCK ======================

func (h *StockHandler) ListBalances(w http.ResponseWriter, r *http.Request) {
	f, err := parseStockFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.ListBalances(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	writeList(w, res)
}

func (h *StockHandler) StockSummary(w http.ResponseWriter, r *http.Request) {
	f, err := parseStockFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.StockSummary(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *StockHandler) GetItemStock(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseItemID(w, r)
	if !ok {
		return
	}
	res, err := h.svc.GetItemStock(r.Context(), itemID)
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

func (h *StockHandler) ListBatches(w http.ResponseWriter, r *http.Request) {
	itemID, ok := parseItemID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	f := BatchFilter{
		ItemID:         itemID,
		WarehouseType:  strings.TrimSpace(q.Get("warehouse_type")),
		ExpiryStatuses: csvParam(q.Get("expiry_status")),
	}
	var err error
	if f.WarehouseID, err = uuidParam(q, "warehouse_id"); err != nil {
		fail(w, err)
		return
	}
	list, err := h.svc.ListBatches(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// ====================== MOVEMENTS ======================

func (h *StockHandler) ListMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, limit := parsePagination(r)
	f := MovementFilter{
		Pagination:    Pagination{Page: page, Limit: limit},
		MovementTypes: csvParam(q.Get("movement_type")),
		ReferenceType: strings.TrimSpace(q.Get("reference_type")),
		Search:        strings.TrimSpace(q.Get("search")),
	}
	var err error
	if f.ItemID, err = uuidParam(q, "item_id"); err != nil {
		fail(w, err)
		return
	}
	if f.WarehouseID, err = uuidParam(q, "warehouse_id"); err != nil {
		fail(w, err)
		return
	}
	if f.DateFrom, err = dateParam(q, "date_from"); err != nil {
		fail(w, err)
		return
	}
	if f.DateTo, err = dateParam(q, "date_to"); err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.ListMovements(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	writeList(w, res)
}

func (h *StockHandler) MovementTypes(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.MovementTypes(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, list)
}

// ====================== VALUATION ======================

func (h *StockHandler) ListValuation(w http.ResponseWriter, r *http.Request) {
	f, err := parseValuationFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.ListValuation(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	writeList(w, res)
}

func (h *StockHandler) ValuationSummary(w http.ResponseWriter, r *http.Request) {
	f, err := parseValuationFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.ValuationSummary(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

// ====================== ALERTS ======================

func (h *StockHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	f, err := parseAlertFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.ListAlerts(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	writeList(w, res)
}

func (h *StockHandler) AlertSummary(w http.ResponseWriter, r *http.Request) {
	f, err := parseAlertFilter(r)
	if err != nil {
		fail(w, err)
		return
	}
	res, err := h.svc.AlertSummary(r.Context(), f)
	if err != nil {
		fail(w, err)
		return
	}
	response.JSON(w, http.StatusOK, res)
}

// ====================== REQUEST PARSING ======================

// parsePagination keeps its original signature and defaults (page 1, limit 20, max 100).
func parsePagination(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxLimit {
		limit = defaultLimit
	}
	return page, limit
}

func parseStockFilter(r *http.Request) (StockFilter, error) {
	q := r.URL.Query()
	page, limit := parsePagination(r)
	f := StockFilter{
		Pagination:    Pagination{Page: page, Limit: limit},
		Search:        strings.TrimSpace(q.Get("search")),
		ItemType:      strings.TrimSpace(q.Get("item_type")),
		WarehouseType: strings.TrimSpace(q.Get("warehouse_type")),
		Statuses:      csvParam(q.Get("status")),
		Sort:          q.Get("sort"),
		Order:         strings.ToLower(q.Get("order")),
	}
	var err error
	if f.WarehouseID, err = uuidParam(q, "warehouse_id"); err != nil {
		return f, err
	}
	if f.CategoryID, err = uuidParam(q, "category_id"); err != nil {
		return f, err
	}
	return f, nil
}

func parseValuationFilter(r *http.Request) (ValuationFilter, error) {
	q := r.URL.Query()
	page, limit := parsePagination(r)
	f := ValuationFilter{
		Pagination:    Pagination{Page: page, Limit: limit},
		Search:        strings.TrimSpace(q.Get("search")),
		ItemType:      strings.TrimSpace(q.Get("item_type")),
		WarehouseType: strings.TrimSpace(q.Get("warehouse_type")),
		CostSources:   csvParam(q.Get("cost_source")),
		Sort:          q.Get("sort"),
		Order:         strings.ToLower(q.Get("order")),
	}
	var err error
	if f.WarehouseID, err = uuidParam(q, "warehouse_id"); err != nil {
		return f, err
	}
	if raw := q.Get("low_stock"); raw != "" {
		b, perr := strconv.ParseBool(raw)
		if perr != nil {
			return f, invalid("low_stock", "must be true or false")
		}
		f.LowStock = &b
	}
	return f, nil
}

func parseAlertFilter(r *http.Request) (AlertFilter, error) {
	q := r.URL.Query()
	page, limit := parsePagination(r)
	f := AlertFilter{
		Pagination: Pagination{Page: page, Limit: limit},
		AlertTypes: csvParam(q.Get("alert_type")),
		Severities: csvParam(q.Get("severity")),
		Search:     strings.TrimSpace(q.Get("search")),
	}
	var err error
	if f.WarehouseID, err = uuidParam(q, "warehouse_id"); err != nil {
		return f, err
	}
	return f, nil
}

func parseItemID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "itemID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid item ID", nil)
		return uuid.Nil, false
	}
	return id, true
}

// csvParam splits "a,b,c" into trimmed, non-empty values. Empty input returns nil.
func csvParam(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func uuidParam(q url.Values, key string) (*uuid.UUID, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, invalid(key, "must be a valid UUID")
	}
	return &id, nil
}

// dateParam parses YYYY-MM-DD.
func dateParam(q url.Values, key string) (*time.Time, error) {
	raw := q.Get(key)
	if raw == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, invalid(key, "must be a date in YYYY-MM-DD format")
	}
	return &t, nil
}

// ====================== RESPONSE HELPERS ======================

func writeList[T any](w http.ResponseWriter, res ListResult[T]) {
	response.JSONWithMeta(w, http.StatusOK, res.Items, &response.Meta{
		Page:  res.Page,
		Limit: res.Limit,
		Total: res.Total,
	})
}

// fail maps typed errors to HTTP responses. Unknown errors are logged and returned
// as a generic 500 so database details never reach the client.
func fail(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		// Details is a field -> message map, e.g. {"status": "must be one of: ..."}.
		// Same status/code as the procurement module so the frontend handles one error shape.
		response.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "Validation failed", ve.Fields)
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "Item not found", nil)
	default:
		log.Printf("inventory: internal error: %v", err)
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Something went wrong", nil)
	}
}
