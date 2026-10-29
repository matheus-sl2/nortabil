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

// GetDRE gera a Demonstração do Resultado do Exercício
func (s *ReportService) GetDRE(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.DREReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return s.reportRepo.GetDREReport(ctx, companyID, startDate, endDate)
}

// GetBalanceSheet gera o Balanço Patrimonial Estruturado
func (s *ReportService) GetBalanceSheet(ctx context.Context, companyID uuid.UUID, refDate time.Time) (*domain.BalanceSheetReport, error) {
	return s.reportRepo.GetBalanceSheetReport(ctx, companyID, refDate)
}

// GetFinancialIndicators calcula os índices de Liquidez, Endividamento e Rentabilidade do período
func (s *ReportService) GetFinancialIndicators(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.FinancialIndicatorsReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	dre, err := s.reportRepo.GetDREReport(ctx, companyID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	bs, err := s.reportRepo.GetBalanceSheetReport(ctx, companyID, endDate)
	if err != nil {
		return nil, err
	}
	return domain.CalculateFinancialIndicators(bs, dre), nil
}

// GetCashFlow gera a Demonstração dos Fluxos de Caixa (DFC - Método Direto conforme NBC TG 03)
func (s *ReportService) GetCashFlow(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) (*domain.CashFlowReport, error) {
	if startDate.After(endDate) {
		return nil, domain.ErrInvalidDateRange
	}
	return s.reportRepo.GetCashFlowReport(ctx, companyID, startDate, endDate)
}

