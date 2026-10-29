package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// ReportService gerencia a geração de relatórios e livros contábeis legais
type ReportService struct {
	reportRepo *postgres.ReportRepository
}

// NewReportService cria uma nova instância do serviço de relatórios
func NewReportService(reportRepo *postgres.ReportRepository) *ReportService {
	return &ReportService{reportRepo: reportRepo}
}

// GetLedger gera o Livro Razão por empresa
func (s *ReportService) GetLedger(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time, accountID *uuid.UUID) (*domain.LedgerReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return s.reportRepo.GetLedgerReport(ctx, companyID, startDate, endDate, accountID)
}

// GetDailyBook gera o Livro Diário Geral por empresa
func (s *ReportService) GetDailyBook(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.DailyBookReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return s.reportRepo.GetDailyBookReport(ctx, companyID, startDate, endDate)
}

// GetTrialBalance gera o Balancete de Verificação por empresa
func (s *ReportService) GetTrialBalance(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.TrialBalanceReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return s.reportRepo.GetTrialBalanceReport(ctx, companyID, startDate, endDate)
}
