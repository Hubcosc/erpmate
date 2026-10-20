package main

import (
	"log"
	"net/http"
	"time"
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/Ethio-ERP/erpmate/config"
	"github.com/Ethio-ERP/erpmate/internal/database"
	mw "github.com/Ethio-ERP/erpmate/internal/middleware"

	"github.com/Ethio-ERP/erpmate/internal/modules/auth"
	"github.com/Ethio-ERP/erpmate/internal/modules/inventory"
	"github.com/Ethio-ERP/erpmate/internal/modules/master"
	"github.com/Ethio-ERP/erpmate/internal/modules/material"
	"github.com/Ethio-ERP/erpmate/internal/modules/notification"
	"github.com/Ethio-ERP/erpmate/internal/modules/procurement"
	"github.com/Ethio-ERP/erpmate/internal/modules/production"
	"github.com/Ethio-ERP/erpmate/internal/modules/purchase"
	"github.com/Ethio-ERP/erpmate/internal/modules/report"
	"github.com/Ethio-ERP/erpmate/internal/modules/sales"
	"github.com/Ethio-ERP/erpmate/internal/modules/user"
)

func main() {
	// 1. Load configuration from .env + environment variables
	cfg := config.Load()

	// 1b. Validate required secrets/config before starting
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// 2. Connect to database
	db := database.Connect(cfg)

	// Bootstrap admin on startup
	if err := database.BootstrapAdmin(context.Background(), db, cfg); err != nil {
		log.Fatalf("bootstrap admin failed: %v", err)
	}

	// 3. Initialize modules
	// Auth
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, cfg.JWTSecret)
	authHandler := auth.NewHandler(authService)

	// User Management
	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// Master Data
	masterRepo := master.NewRepository(db)
	masterService := master.NewService(masterRepo)
	masterHandler := master.NewHandler(masterService)

	// Purchase Requisition (Your module)
	purchaseRepo := purchase.NewRepository(db)
	purchaseService := purchase.NewService(purchaseRepo)
	purchaseHandler := purchase.NewHandler(purchaseService)

	// Material Requisition + Issue (Your module)
	materialRepo := material.NewRepository(db)
	materialService := material.NewService(materialRepo)
	materialHandler := material.NewHandler(materialService)

	// In-app notifications (read by the authenticated recipient only)
	notificationRepo := notification.NewRepository(db)
	notificationService := notification.NewService(notificationRepo)
	notificationHandler := notification.NewHandler(notificationService)

	// Procurement (PO + GRN), production receipts, and sales/delivery.
	procurementRepo := procurement.NewWorkflowRepository(db)
	procurementService := procurement.NewService(procurementRepo)
	procurementHandler := procurement.NewWorkflowHandler(procurementService)
	productionRepo := production.NewWorkflowRepository(db)
	productionService := production.NewWorkflowService(productionRepo)
	productionHandler := production.NewWorkflowHandler(productionService)
	salesRepo := sales.NewWorkflowRepository(db)
	salesService := sales.NewWorkflowService(salesRepo)
	salesHandler := sales.NewWorkflowHandler(salesService)
	inventoryStockRepo := inventory.NewStockRepository(db)
	inventoryStockHandler := inventory.NewStockHandler(inventoryStockRepo)
	go inventory.RunAlertWorker(db, 15*time.Minute, 30)
	reportRepo := report.NewRepository(db)
	reportHandler := report.NewHandler(reportRepo)

	// 4. Setup Router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// 5. Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})

		// ========== Public Routes ==========
		r.Post("/auth/login", authHandler.Login)

		// ========== Protected Routes ==========
		r.Group(func(r chi.Router) {
			r.Use(mw.Auth(cfg.JWTSecret))

			// Auth
			r.Get("/auth/me", authHandler.Me)

			// User Management
			r.Mount("/users", userHandler.Routes())

			// Master Data
			r.Mount("/master", masterHandler.Routes())

			// Purchase Requisition (Your module)
			r.Mount("/purchase-requisitions", purchaseHandler.Routes())

			// Material Requisition + Issue (Your module)
			r.Mount("/material-requisitions", materialHandler.Routes())

			// Status notifications
			r.Mount("/notifications", notificationHandler.Routes())

			// Procurement and inventory receipt workflows.
			r.Mount("/purchase-orders", procurementHandler.PurchaseOrderRoutes())
			r.Mount("/goods-receipt-notes", procurementHandler.GRNRoutes())
			r.Mount("/production-receipts", productionHandler.Routes())
			r.Mount("/sales-orders", salesHandler.OrderRoutes())
			r.Mount("/delivery-notes", salesHandler.DeliveryRoutes())
			r.Mount("/inventory", inventoryStockHandler.Routes())
			r.Mount("/reports", reportHandler.Routes())
		})
	})

	// 6. Start server
	log.Printf("Server starting on http://localhost:%s", cfg.Port)
	log.Printf("API Base URL: http://localhost:%s/api/v1", cfg.Port)

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
