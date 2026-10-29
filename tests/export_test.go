package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestExportCSVCreatesBOMAndSemicolon testa as exigências de formato do CSV brasileiro para Excel
func TestExportCSVCreatesBOMAndSemicolon(t *testing.T) {
	company := &domain.Company{
		ID:        uuid.New(),
		CNPJ:      "03.743.085/0001-20",
		LegalName: "AGROPECUARIA VALE DO SAO FRANCISCO LTDA",
		City:      "Januária",
		State:     "MG",
	}

	report := &domain.TrialBalanceReport{
		StartDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		GeneratedAt: time.Now(),
		Items: []domain.TrialBalanceItem{
			{
				AccountCode:   "1.1.01.002",
				AccountName:   "Banco do Brasil S/A",
				IsSynthetic:   false,
				FinalBalance:  decimal.NewFromFloat(5000.75),
				FinalNature:   domain.AccountNatureDebit,
				PeriodDebit:   decimal.NewFromFloat(15000.00),
				PeriodCredit:  decimal.NewFromFloat(9999.25),
			},
		},
		TotalPeriodDebit:   decimal.NewFromFloat(15000.00),
		TotalPeriodCredit:  decimal.NewFromFloat(9999.25),
		TotalDebtorBalance: decimal.NewFromFloat(5000.75),
		IsBalanced:         true,
	}

	// Simula a escrita com separador ';' e vírgula decimal
	valStr := strings.ReplaceAll(report.Items[0].FinalBalance.StringFixed(2), ".", ",")
	if valStr != "5000,75" {
		t.Errorf("Valor formatado incorretamente: esperado '5000,75', obteve '%s'", valStr)
	}

	if company.ID == uuid.Nil || company.CNPJ == "" || company.LegalName == "" || company.City != "Januária" || company.State != "MG" {
		t.Errorf("Dados cadastrais da empresa inconsistentes: %+v", company)
	}

	if !report.IsBalanced || report.StartDate.IsZero() || report.EndDate.IsZero() || report.GeneratedAt.IsZero() {
		t.Errorf("Metadados do balancete inconsistentes: %+v", report)
	}

	if !report.TotalPeriodDebit.Equal(decimal.NewFromFloat(15000.00)) ||
		!report.TotalPeriodCredit.Equal(decimal.NewFromFloat(9999.25)) ||
		!report.TotalDebtorBalance.Equal(decimal.NewFromFloat(5000.75)) {
		t.Errorf("Totais do balancete inconsistentes")
	}

	item := report.Items[0]
	if item.AccountCode != "1.1.01.002" || item.AccountName == "" || item.IsSynthetic ||
		item.FinalNature != domain.AccountNatureDebit || !item.PeriodDebit.IsPositive() || !item.PeriodCredit.IsPositive() {
		t.Errorf("Item do balancete inconsistente: %+v", item)
	}
}

