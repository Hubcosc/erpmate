package procurement

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler is the module entry point: it wires repository -> service -> HTTP
// handlers and exposes every procurement route under one router.
type Handler struct{ workflows *WorkflowHandler }

func NewHandler(db *pgxpool.Pool) *Handler {
	svc := NewService(NewWorkflowRepository(db))
	return &Handler{workflows: NewWorkflowHandler(svc)}
}

// Routes returns:
//
//	/purchase-orders/...
//	/goods-receipts/...
//
// Mount it behind your auth middleware, e.g. r.Mount("/procurement", h.Routes()).
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Mount("/purchase-orders", h.workflows.PurchaseOrderRoutes())
	r.Mount("/goods-receipts", h.workflows.GRNRoutes())
	return r
}
