package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/middleware"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
	"github.com/matheus-sl2/nortabil/internal/service"
	"github.com/matheus-sl2/nortabil/web"
)

// SetupRouter configura as rotas e middlewares da aplicação HTTP
func SetupRouter(
	db *postgres.DB,
	companyHandler *CompanyHandler,
	accountHandler *AccountHandler,
	journalHandler *JournalHandler,
	reportHandler *ReportHandler,
	periodHandler *PeriodHandler,
	reconciliationHandler *ReconciliationHandler,
	nfeHandler *NFeHandler,
	spedHandler *SPEDHandler,
	exportHandler *ExportHandler,
	authHandler *AuthHandler,
	authService *service.AuthService,
	swaggerYAML []byte,
) http.Handler {
	r := chi.NewRouter()

	// Middlewares globais
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)

	// Injeta claims JWT se token for enviado
	if authService != nil {
		r.Use(middleware.AuthMiddleware(func(tokenStr string) (uuid.UUID, string, string, domain.UserRole, error) {
			claims, err := authService.ValidateToken(tokenStr)
			if err != nil {
				return uuid.Nil, "", "", "", err
			}
			return claims.UserID, claims.Email, claims.Name, claims.Role, nil
		}))
	}

	// Configuração de CORS (Cross-Origin Resource Sharing)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Company-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Company-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health Check para orquestradores (Docker, Kubernetes, Cloud Run)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		dbStatus := "healthy"
		if err := db.Pool.Ping(ctx); err != nil {
			dbStatus = "unhealthy: " + err.Error()
			JSONError(w, http.StatusServiceUnavailable, "DB_DOWN", "Falha de conexão com o banco de dados", dbStatus)
			return
		}

		JSON(w, http.StatusOK, map[string]interface{}{
			"status":      "UP",
			"database":    dbStatus,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"service":     "nortabil-double-entry-engine",
			"environment": "cloud-ready",
		})
	})

	// Rota para documentação OpenAPI / Swagger
	r.Get("/docs/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(swaggerYAML)
	})

	// Swagger UI interativo embutido
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		swaggerHTML := `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8" />
  <title>Nortabil API - Documentação Interativa</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
</head>
<body style="margin: 0; background-color: #fafafa;">
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: '/docs/swagger.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
      });
    };
  </script>
</body>
</html>`
		_, _ = w.Write([]byte(swaggerHTML))
	})

	// Registra interface web estática (Dashboard Contábil)
	web.RegisterWebRoutes(r)

	// Agrupamento da API v1
	r.Route("/api/v1", func(api chi.Router) {
		// Rotas de Autenticação e Gestão de Usuários
		if authHandler != nil {
			api.Route("/auth", func(auth chi.Router) {
				auth.Post("/login", authHandler.Login)
				auth.Get("/me", authHandler.Me)
				auth.Get("/lgpd/export", authHandler.ExportMyData)
				auth.Delete("/lgpd/delete-account", authHandler.DeleteMyData)
				// Gestão de usuários restrita ao perfil ADMIN (impede autocadastro de administradores)
				auth.With(middleware.RequireRole(domain.RoleAdmin)).Post("/register", authHandler.Register)
				auth.With(middleware.RequireRole(domain.RoleAdmin)).Get("/users", authHandler.ListUsers)
			})
		}

		// Rotas de Empresas / Clientes do Escritório
		api.Route("/companies", func(comp chi.Router) {
			comp.Post("/", companyHandler.Create)
			comp.Get("/", companyHandler.List)
			comp.Get("/{id}", companyHandler.GetByID)

			// Rotas de contas e lançamentos com companyId explícito na URL
			comp.Route("/{companyId}", func(sub chi.Router) {
				sub.Use(middleware.TenantMiddleware)

				sub.Route("/accounts", func(acc chi.Router) {
					acc.Post("/", accountHandler.Create)
					acc.Get("/", accountHandler.List)
					acc.Get("/{id}", accountHandler.GetByID)
					acc.Get("/code/{code}", accountHandler.GetByCode)
				})

				sub.Route("/journal-entries", func(je chi.Router) {
					je.Post("/", journalHandler.Create)
					je.Post("/batch", journalHandler.CreateBatch)
					je.Get("/", journalHandler.List)
					je.Get("/{id}", journalHandler.GetByID)
					je.Post("/{id}/reverse", journalHandler.Reverse)
				})

				sub.Route("/reports", func(rep chi.Router) {
					rep.Get("/ledger", reportHandler.GetLedger)
					rep.Get("/journal", reportHandler.GetDailyBook)
					rep.Get("/trial-balance", reportHandler.GetTrialBalance)
					rep.Get("/dre", reportHandler.GetDRE)
					rep.Get("/dfc", reportHandler.GetCashFlow)
					rep.Get("/balance-sheet", reportHandler.GetBalanceSheet)
					rep.Get("/indicators", reportHandler.GetIndicators)
					rep.Get("/export/csv", exportHandler.ExportCSV)
					rep.Get("/export/print", exportHandler.ExportPrintable)
				})

				sub.Route("/periods", func(per chi.Router) {
					per.Get("/", periodHandler.List)
					per.With(middleware.RequireRole(domain.RoleContador)).Post("/close", periodHandler.Close)
					per.With(middleware.RequireRole(domain.RoleContador)).Post("/reopen", periodHandler.Reopen)
					per.With(middleware.RequireRole(domain.RoleContador)).Post("/closing-year", periodHandler.CloseYear)
				})

				sub.Route("/reconciliation", func(rec chi.Router) {
					rec.Post("/import-ofx", reconciliationHandler.ImportOFX)
					rec.Get("/items", reconciliationHandler.ListItems)
				})

				if nfeHandler != nil {
					sub.Route("/nfe", func(nfe chi.Router) {
						nfe.Post("/import-xml", nfeHandler.ImportXML)
					})
				}

				sub.Get("/export/sped-ecd", spedHandler.ExportECD)
				sub.Get("/export/sped-ecd/validate", spedHandler.ValidateECD)
			})
		})

		// Rotas de conveniência utilizando X-Company-ID no Header ou query param
		api.Group(func(tenantGroup chi.Router) {
			tenantGroup.Use(middleware.TenantMiddleware)

			tenantGroup.Route("/accounts", func(acc chi.Router) {
				acc.Post("/", accountHandler.Create)
				acc.Get("/", accountHandler.List)
				acc.Get("/{id}", accountHandler.GetByID)
				acc.Get("/code/{code}", accountHandler.GetByCode)
			})

			tenantGroup.Route("/journal-entries", func(je chi.Router) {
				je.Post("/", journalHandler.Create)
				je.Post("/batch", journalHandler.CreateBatch)
				je.Get("/", journalHandler.List)
				je.Get("/{id}", journalHandler.GetByID)
				je.Post("/{id}/reverse", journalHandler.Reverse)
			})

			tenantGroup.Route("/reports", func(rep chi.Router) {
				rep.Get("/ledger", reportHandler.GetLedger)
				rep.Get("/journal", reportHandler.GetDailyBook)
				rep.Get("/trial-balance", reportHandler.GetTrialBalance)
				rep.Get("/dre", reportHandler.GetDRE)
				rep.Get("/dfc", reportHandler.GetCashFlow)
				rep.Get("/balance-sheet", reportHandler.GetBalanceSheet)
				rep.Get("/indicators", reportHandler.GetIndicators)
				rep.Get("/export/csv", exportHandler.ExportCSV)
				rep.Get("/export/print", exportHandler.ExportPrintable)
			})

			tenantGroup.Route("/periods", func(per chi.Router) {
				per.Get("/", periodHandler.List)
				per.With(middleware.RequireRole(domain.RoleContador)).Post("/close", periodHandler.Close)
				per.With(middleware.RequireRole(domain.RoleContador)).Post("/reopen", periodHandler.Reopen)
				per.With(middleware.RequireRole(domain.RoleContador)).Post("/closing-year", periodHandler.CloseYear)
			})

			tenantGroup.Route("/reconciliation", func(rec chi.Router) {
				rec.Post("/import-ofx", reconciliationHandler.ImportOFX)
				rec.Get("/items", reconciliationHandler.ListItems)
			})

			if nfeHandler != nil {
				tenantGroup.Route("/nfe", func(nfe chi.Router) {
					nfe.Post("/import-xml", nfeHandler.ImportXML)
				})
			}

			tenantGroup.Get("/export/sped-ecd", spedHandler.ExportECD)
			tenantGroup.Get("/export/sped-ecd/validate", spedHandler.ValidateECD)
		})
	})

	return r
}
