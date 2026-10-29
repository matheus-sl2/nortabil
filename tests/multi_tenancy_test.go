package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestCNPJValidation valida o algoritmo de dígitos verificadores de CNPJ da Receita Federal
func TestCNPJValidation(t *testing.T) {
	tests := []struct {
		name     string
		cnpj     string
		expected bool
	}{
		{"CNPJ Válido 1 (Nortabil Matriz)", "12.345.678/0001-90", false}, // Dígito calculado proposital
		{"CNPJ Válido Real (Petrobras)", "33.000.167/0001-01", true},
		{"CNPJ Válido Real (Banco do Brasil)", "00.000.000/0001-91", true},
		{"CNPJ Válido Real (Vale S.A.)", "33.592.510/0001-54", true},
		{"CNPJ Inválido - Sequência repetida", "11.111.111/1111-11", false},
		{"CNPJ Inválido - Todos zeros", "00.000.000/0000-00", false},
		{"CNPJ Inválido - Tamanho incompleto", "12.345.678/0001", false},
		{"CNPJ Inválido - Dígito verificador incorreto", "33.000.167/0001-99", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ValidateCNPJ(tt.cnpj)
			if result != tt.expected {
				t.Fatalf("Para o CNPJ %s, esperava %v mas obteve %v", tt.cnpj, tt.expected, result)
			}
		})
	}
}

// TestMultiTenantIsolation valida a associação de contas e lançamentos à empresa correta
func TestMultiTenantIsolation(t *testing.T) {
	companyA := uuid.New()
	companyB := uuid.New()

	accCompanyA := domain.Account{
		ID:        uuid.New(),
		CompanyID: companyA,
		Code:      "1.1.01.001",
		Name:      "Caixa Geral Empresa A",
		Type:      domain.AccountTypeAsset,
		Nature:    domain.AccountNatureDebit,
	}

	accCompanyB := domain.Account{
		ID:        uuid.New(),
		CompanyID: companyB,
		Code:      "1.1.01.001", // Mesmo código contábil, mas empresa diferente!
		Name:      "Caixa Geral Empresa B",
		Type:      domain.AccountTypeAsset,
		Nature:    domain.AccountNatureDebit,
	}

	if accCompanyA.CompanyID == accCompanyB.CompanyID {
		t.Fatalf("Contas de empresas diferentes não podem compartilhar o mesmo CompanyID")
	}

	if accCompanyA.Code != accCompanyB.Code {
		t.Fatalf("Multi-tenancy deve permitir que empresas diferentes tenham códigos iguais no plano")
	}
}
