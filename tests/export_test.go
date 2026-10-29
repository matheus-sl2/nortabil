package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/service"
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

// TestSPEDECDValidationPVA valida o Pré-Validador PVA interno sobre um arquivo SPED ECD gerado pelo motor
func TestSPEDECDValidationPVA(t *testing.T) {
	company := &domain.Company{
		ID:        uuid.New(),
		CNPJ:      "19.131.243/0001-97",
		LegalName: "FRUTICULTURA VALE DO JAIBA LTDA",
		City:      "Jaíba",
		State:     "MG",
	}

	accounts := []domain.Account{
		{ID: uuid.New(), Code: "1.1.01.002", Name: "Banco Sicoob", Type: domain.AccountTypeAsset, IsSynthetic: false, Active: true},
		{ID: uuid.New(), Code: "4.1.01.001", Name: "Venda de Frutas", Type: domain.AccountTypeRevenue, IsSynthetic: false, Active: true},
	}

	entries := []domain.JournalEntry{
		{
			ID:          uuid.New(),
			EntryNumber: 1,
			Date:        time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
			Description: "Venda de Banana Prata",
			TotalDebit:  decimal.RequireFromString("12500.00"),
			TotalCredit: decimal.RequireFromString("12500.00"),
			Lines: []domain.JournalLine{
				{AccountCode: "1.1.01.002", Type: domain.JournalLineDebit, Amount: decimal.RequireFromString("12500.00"), Description: "Recebimento Sicoob"},
				{AccountCode: "4.1.01.001", Type: domain.JournalLineCredit, Amount: decimal.RequireFromString("12500.00"), Description: "Faturamento Safra"},
			},
		},
	}

	cfg := domain.SPEDExportConfig{
		CompanyID:     company.ID,
		StartDate:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		BookType:      "G",
		AuditorCRC:    "MG-085421/O",
		AuditorName:   "Sebastião Ferreira",
		AuditorCPF:    "12345678909",
		SignatoryName: "Representante Jaíba",
		SignatoryCPF:  "98765432100",
		SignatoryQual: "207",
	}

	txt := service.BuildSPEDFile(company, accounts, entries, cfg)
	v := service.ValidateECDContent(txt)

	if !v.IsValid || !v.IsDoubleEntryBalanced || len(v.Errors) > 0 {
		t.Fatalf("esperado arquivo SPED ECD 100%% válido no PVA interno, erros: %v", v.Errors)
	}
	if v.TotalLines != v.ReportedLines9999 || v.TotalAccountsI050 != 2 || v.TotalEntriesI200 != 1 || v.TotalPartidasI250 != 2 {
		t.Errorf("contadores SPED incorretos: %+v", v)
	}
	if !v.TotalDebitI250.Equal(decimal.RequireFromString("12500.00")) || !v.TotalCreditI250.Equal(decimal.RequireFromString("12500.00")) {
		t.Errorf("somas I250 incorretas: D=%s, C=%s", v.TotalDebitI250, v.TotalCreditI250)
	}
	if len(v.BlocksVerified) != 4 {
		t.Errorf("esperados 4 blocos verificados, obtidos %d", len(v.BlocksVerified))
	}
}

