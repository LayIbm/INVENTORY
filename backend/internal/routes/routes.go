package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/layssagonzalez/device-inventory/backend/internal/authhandler"
	"github.com/layssagonzalez/device-inventory/backend/internal/billing"
	"github.com/layssagonzalez/device-inventory/backend/internal/enlace"
	"github.com/layssagonzalez/device-inventory/backend/internal/equipment"
	"github.com/layssagonzalez/device-inventory/backend/internal/importer"
	"github.com/layssagonzalez/device-inventory/backend/internal/importexport"
	"github.com/layssagonzalez/device-inventory/backend/internal/laptop"
	"github.com/layssagonzalez/device-inventory/backend/internal/middleware"
	"github.com/layssagonzalez/device-inventory/backend/internal/user"
)

// Register wires all routes and returns the root handler.
func Register(pool *pgxpool.Pool, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Stores & handlers
	laptopStore := laptop.NewStore(pool)
	laptopHandler := laptop.NewHandler(laptopStore)

	userStore := user.NewStore(pool)
	userHandler := user.NewHandler(userStore)
	authHandler := authhandler.NewHandler(userStore, jwtSecret)

	billingStore := billing.NewStore(pool)
	billingHandler := billing.NewHandler(billingStore)

	enlaceStore := enlace.NewStore(pool)
	enlaceHandler := enlace.NewHandler(enlaceStore)

	equipmentStore := equipment.NewStore(pool)
	equipmentHandler := equipment.NewHandler(equipmentStore)

	// Our official importer (preview + apply)
	importSvc := importer.NewService(pool)
	importHandler := importer.NewHandler(importSvc)

	// Layssa's exportexport handler (export only — Import route NOT registered)
	importExportHandler := importexport.NewHandler(laptopStore, equipmentStore, enlaceStore)

	// Public routes
	r.Post("/api/auth/login", authHandler.Login)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(jwtSecret))

		// Change password — any authenticated user
		r.Post("/api/auth/change-password", authHandler.ChangePassword)

		// Laptops — read: any role; write: manager + admin
		r.Get("/api/laptops", laptopHandler.List)
		r.Get("/api/laptops/{id}", laptopHandler.Get)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/laptops", laptopHandler.Create)
		r.With(middleware.RequireRole("manager", "admin")).Put("/api/laptops/{id}", laptopHandler.Update)
		r.With(middleware.RequireRole("manager", "admin")).Delete("/api/laptops/{id}", laptopHandler.Delete)

		// Users — admin only
		r.With(middleware.RequireRole("admin")).Get("/api/users", userHandler.List)
		r.With(middleware.RequireRole("admin")).Post("/api/users", userHandler.Create)
		r.With(middleware.RequireRole("admin")).Put("/api/users/{id}", userHandler.Update)
		r.With(middleware.RequireRole("admin")).Patch("/api/users/{id}/toggle-active", userHandler.ToggleActive)
		r.With(middleware.RequireRole("admin")).Delete("/api/users/{id}", userHandler.Delete)

		// Equipment (Desktop/Monitor/Adapter) — read: any role; write: manager + admin
		r.Get("/api/equipment", equipmentHandler.List)
		r.Get("/api/equipment/{id}", equipmentHandler.Get)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/equipment", equipmentHandler.Create)
		r.With(middleware.RequireRole("manager", "admin")).Put("/api/equipment/{id}", equipmentHandler.Update)
		r.With(middleware.RequireRole("manager", "admin")).Delete("/api/equipment/{id}", equipmentHandler.Delete)

		// Invoices — read: any role; write: manager + admin
		r.Get("/api/invoices", billingHandler.List)
		r.Get("/api/invoices/{id}", billingHandler.Get)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/invoices", billingHandler.Create)
		r.With(middleware.RequireRole("manager", "admin")).Put("/api/invoices/{id}", billingHandler.Update)
		r.With(middleware.RequireRole("manager", "admin")).Delete("/api/invoices/{id}", billingHandler.Delete)

		// Enlaces — read: any role; write: manager + admin
		r.Get("/api/enlaces", enlaceHandler.List)
		r.Get("/api/enlaces/{id}", enlaceHandler.Get)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/enlaces", enlaceHandler.Create)
		r.With(middleware.RequireRole("manager", "admin")).Put("/api/enlaces/{id}", enlaceHandler.Update)
		r.With(middleware.RequireRole("manager", "admin")).Delete("/api/enlaces/{id}", enlaceHandler.Delete)

		// Import — official importer (preview + apply) — manager + admin only
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/import/inventory/preview", importHandler.Preview)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/import/inventory/apply", importHandler.Apply)

		// Import / Export — legacy + standard Excel flow — manager + admin only
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/import/excel", importExportHandler.Import)
		r.With(middleware.RequireRole("manager", "admin")).Post("/api/import/links", importExportHandler.ImportLinks)
		r.With(middleware.RequireRole("manager", "admin")).Get("/api/export/excel", importExportHandler.Export)
		r.With(middleware.RequireRole("manager", "admin")).Get("/api/export/template", importExportHandler.ExportTemplate)
	})

	return r
}
