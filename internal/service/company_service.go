package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// CompanyService orquestra as regras de negócio de empresas e clientes do escritório
type CompanyService struct {
	companyRepo *postgres.CompanyRepository
}

// NewCompanyService cria uma nova instância do serviço de empresas
func NewCompanyService(companyRepo *postgres.CompanyRepository) *CompanyService {
	return &CompanyService{companyRepo: companyRepo}
}

// CreateCompanyInput dados de entrada para criação de empresa
type CreateCompanyInput struct {
	CNPJ                 string           `json:"cnpj"`
	LegalName            string           `json:"legal_name"`
	TradeName            string           `json:"trade_name"`
	TaxRegime            domain.TaxRegime `json:"tax_regime"`
	City                 string           `json:"city"`
	State                string           `json:"state"`
	CloneDefaultAccounts bool             `json:"clone_default_accounts"` // Clona plano de contas padrão (SPED/ECD)
}

// CreateCompany cadastra uma nova empresa cliente no escritório contábil
func (s *CompanyService) CreateCompany(ctx context.Context, input CreateCompanyInput) (*domain.Company, error) {
	input.CNPJ = domain.CleanCNPJ(input.CNPJ)
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.TradeName = strings.TrimSpace(input.TradeName)
	input.City = strings.TrimSpace(input.City)
	input.State = strings.ToUpper(strings.TrimSpace(input.State))

	if !domain.ValidateCNPJ(input.CNPJ) {
		return nil, domain.ErrInvalidCNPJ
	}
	if input.LegalName == "" {
		return nil, domain.ErrInvalidAccountCode // Nome inválido
	}
	if !domain.IsValidTaxRegime(input.TaxRegime) {
		return nil, domain.ErrInvalidTaxRegime
	}
	if input.City == "" {
		input.City = "Januária"
	}
	if input.State == "" {
		input.State = "MG"
	}

	formattedCNPJ := domain.FormatCNPJ(input.CNPJ)

	// Verifica duplicidade de CNPJ
	existing, err := s.companyRepo.GetByCNPJ(ctx, formattedCNPJ)
	if err == nil && existing != nil {
		return nil, domain.ErrCompanyAlreadyExists
	}

	company := &domain.Company{
		ID:        uuid.New(),
		CNPJ:      formattedCNPJ,
		LegalName: input.LegalName,
		TradeName: input.TradeName,
		TaxRegime: input.TaxRegime,
		City:      input.City,
		State:     input.State,
		Active:    true,
	}

	if err := s.companyRepo.Create(ctx, company); err != nil {
		return nil, err
	}

	// Se solicitado, clona o plano de contas referencial padrão para a nova empresa
	if input.CloneDefaultAccounts {
		if err := s.companyRepo.CloneDefaultChartOfAccounts(ctx, company.ID); err != nil {
			return nil, err
		}
	}

	return company, nil
}

// GetByID busca empresa pelo ID
func (s *CompanyService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	return s.companyRepo.GetByID(ctx, id)
}

// ListCompanies lista todas as empresas
func (s *CompanyService) ListCompanies(ctx context.Context, activeOnly bool) ([]domain.Company, error) {
	return s.companyRepo.List(ctx, activeOnly)
}
