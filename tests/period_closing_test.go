package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestPeriodStatusAndValidation testa as regras de status de período contábil
func TestPeriodStatusAndValidation(t *testing.T) {
	companyID := uuid.New()
	year := 2026
	month := 1

	period := domain.AccountingPeriod{
		ID:        uuid.New(),
		CompanyID: companyID,
		Year:      year,
		Month:     month,
		StartDate: time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(year, time.Month(month), 31, 23, 59, 59, 0, time.UTC),
		Status:    domain.PeriodStatusOpen,
	}

	if period.Status != domain.PeriodStatusOpen {
		t.Fatalf("O período recém-criado deveria estar com status OPEN")
	}

	// Simula encerramento do período
	now := time.Now()
	period.Status = domain.PeriodStatusClosed
	period.ClosedAt = &now
	period.ClosedBy = "Contador Responsável CRC/MG 12345"

	if period.ID == uuid.Nil || period.CompanyID != companyID || period.Year != year || period.Month != month ||
		period.StartDate.IsZero() || period.EndDate.IsZero() {
		t.Fatalf("Metadados do período contábil inválidos: %+v", period)
	}

	if period.Status != domain.PeriodStatusClosed || period.ClosedAt == nil || period.ClosedBy == "" {
		t.Fatalf("O período deveria estar com status CLOSED e metadados de fechamento preenchidos")
	}
}

// TestYearClosingCalculation testa a lógica contábil da Apuração do Resultado do Exercício (ARE)
func TestYearClosingCalculation(t *testing.T) {
	// Receitas: R$ 150.000,00
	receitaVendas := decimal.RequireFromString("100000.00")
	receitaServicos := decimal.RequireFromString("50000.00")
	totalReceitas := receitaVendas.Add(receitaServicos)

	// Despesas: R$ 90.000,00
	cmv := decimal.RequireFromString("40000.00")
	salarios := decimal.RequireFromString("35000.00")
	aluguel := decimal.RequireFromString("15000.00")
	totalDespesas := cmv.Add(salarios).Add(aluguel)

	// Lucro Líquido = R$ 60.000,00
	lucroLiquido := totalReceitas.Sub(totalDespesas)
	expectedLucro := decimal.RequireFromString("60000.00")

	if !lucroLiquido.Equal(expectedLucro) {
		t.Fatalf("Lucro líquido apurado (%v) diferente do esperado (%v)", lucroLiquido, expectedLucro)
	}

	// Testa balanceamento da partida de encerramento do exercício
	// Débito nas Receitas para zerar: R$ 100.000 + R$ 50.000 = R$ 150.000 (Débito)
	// Crédito nas Despesas para zerar: R$ 40.000 + R$ 35.000 + R$ 15.000 = R$ 90.000 (Crédito)
	// Crédito em Lucros Acumulados no Patrimônio Líquido: R$ 60.000 (Crédito)
	// Total Débitos: 150.000 | Total Créditos: 90.000 + 60.000 = 150.000 -> Perfeitamente balanceado!

	accReceita1 := uuid.New()
	accReceita2 := uuid.New()
	accDespesa1 := uuid.New()
	accDespesa2 := uuid.New()
	accDespesa3 := uuid.New()
	accLucrosAcum := uuid.New()

	closingEntry := domain.JournalEntry{
		ID:          uuid.New(),
		Date:        time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		Description: "Apuração do Resultado do Exercício (ARE) 2026",
		Lines: []domain.JournalLine{
			{AccountID: accReceita1, Type: domain.JournalLineDebit, Amount: receitaVendas},
			{AccountID: accReceita2, Type: domain.JournalLineDebit, Amount: receitaServicos},
			{AccountID: accDespesa1, Type: domain.JournalLineCredit, Amount: cmv},
			{AccountID: accDespesa2, Type: domain.JournalLineCredit, Amount: salarios},
			{AccountID: accDespesa3, Type: domain.JournalLineCredit, Amount: aluguel},
			{AccountID: accLucrosAcum, Type: domain.JournalLineCredit, Amount: lucroLiquido},
		},
	}

	err := closingEntry.ValidateDoubleEntry()
	if err != nil {
		t.Fatalf("O lançamento de ARE deveria estar perfeitamente balanceado segundo Luca Pacioli: %v", err)
	}

	if !closingEntry.TotalDebit.Equal(closingEntry.TotalCredit) {
		t.Fatalf("Total Débitos (%v) diverge de Total Créditos (%v) no ARE", closingEntry.TotalDebit, closingEntry.TotalCredit)
	}
}

