package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/matheus-sl2/nortabil/docs"
	"github.com/matheus-sl2/nortabil/internal/config"
	"github.com/matheus-sl2/nortabil/internal/handler"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
	"github.com/matheus-sl2/nortabil/internal/service"
)

func main() {
	log.Println("=================================================================")
	log.Println("   NORTABIL - Motor de Partidas Dobradas (Núcleo ERP Contábil)   ")
	log.Println("   Foco Regional: Norte de Minas Gerais (Januária e Região)     ")
	log.Println("=================================================================")

	// 1. Carregamento de Configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[ERRO CRÍTICO] Falha ao carregar configurações: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Conexão e Inicialização do Banco de Dados PostgreSQL
	log.Println("[INFO] Estabelecendo conexão com o PostgreSQL...")
	db, err := postgres.NewDB(ctx, cfg)
	if err != nil {
		log.Printf("[AVISO] Falha ao conectar ao banco de dados: %v", err)
		log.Println("[AVISO] Para operação completa, configure o PostgreSQL local ou execute via 'docker-compose up -d'")
	} else {
		defer db.Close()
		log.Println("[INFO] Conexão com o banco de dados estabelecida com sucesso.")
	}

	// 3. Injeção de Dependências
	var companyHandler *handler.CompanyHandler
	var accountHandler *handler.AccountHandler
	var journalHandler *handler.JournalHandler
	var reportHandler *handler.ReportHandler
	var periodHandler *handler.PeriodHandler
	var reconciliationHandler *handler.ReconciliationHandler
	var spedHandler *handler.SPEDHandler
	var exportHandler *handler.ExportHandler
	var authHandler *handler.AuthHandler
	var authService *service.AuthService

	if db != nil {
		companyRepo := postgres.NewCompanyRepository(db)
		accountRepo := postgres.NewAccountRepository(db)
		journalRepo := postgres.NewJournalRepository(db)
		reportRepo := postgres.NewReportRepository(db, accountRepo)
		periodRepo := postgres.NewPeriodRepository(db)
		reconciliationRepo := postgres.NewReconciliationRepository(db)
		userRepo := postgres.NewUserRepository(db)

		companyService := service.NewCompanyService(companyRepo)
		accountService := service.NewAccountService(accountRepo)
		journalService := service.NewJournalService(journalRepo, accountRepo, periodRepo)
		reportService := service.NewReportService(reportRepo)
		periodService := service.NewPeriodService(periodRepo, journalRepo, accountRepo)
		ofxService := service.NewOFXService(reconciliationRepo, journalService, accountRepo)
		spedService := service.NewSPEDService(companyRepo, accountRepo, journalRepo)
		exportService := service.NewExportService(companyRepo, reportRepo)
		authService = service.NewAuthService(userRepo, cfg.JWTSecret)

		companyHandler = handler.NewCompanyHandler(companyService)
		accountHandler = handler.NewAccountHandler(accountService)
		journalHandler = handler.NewJournalHandler(journalService)
		reportHandler = handler.NewReportHandler(reportService)
		periodHandler = handler.NewPeriodHandler(periodService)
		reconciliationHandler = handler.NewReconciliationHandler(ofxService)
		spedHandler = handler.NewSPEDHandler(spedService)
		exportHandler = handler.NewExportHandler(exportService)
		authHandler = handler.NewAuthHandler(authService)
	}

	// 4. Configuração das Rotas e Servidor HTTP
	router := handler.SetupRouter(db, companyHandler, accountHandler, journalHandler, reportHandler, periodHandler, reconciliationHandler, spedHandler, exportHandler, authHandler, authService, docs.SwaggerYAML)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Inicialização Concorrente do Servidor
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[INFO] Servidor HTTP ouvindo na porta %s (http://localhost:%s)", cfg.Port, cfg.Port)
		log.Printf("[INFO] Documentação Swagger UI disponível em http://localhost:%s/swagger", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 6. Graceful Shutdown (captura de sinais SIGINT e SIGTERM)
	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("[ERRO CRÍTICO] Falha na inicialização do servidor: %v", err)
	case sig := <-shutdownSignal:
		log.Printf("[INFO] Sinal de interrupção recebido (%v). Encerrando graciosamente...", sig)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("[ERRO] Erro durante o encerramento do servidor HTTP: %v", err)
			_ = server.Close()
		}
		log.Println("[INFO] Servidor Nortabil encerrado com sucesso.")
	}
}
