package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestDRECalculation valida a estrutura e cálculos da Demonstração do Resultado do Exercício (NBC TG 26 / Lei 6.404/76)
func TestDRECalculation(t *testing.T) {
	grossRevenue := decimal.RequireFromString("250000.00") // Receita Bruta de Vendas (safra agrícola de Januária)
	deductions := decimal.RequireFromString("15000.00")    // Impostos sobre Vendas (Simples Nacional / ICMS)
	cmv := decimal.RequireFromString("90000.00")           // Custos (fertilizantes, irrigação, sementes)
	operatingExpenses := decimal.RequireFromString("45000.00") // Pessoal, aluguéis, combustível
	financialRevenues := decimal.RequireFromString("3000.00")  // Aplicações financeiras Sicoob
	financialExpenses := decimal.RequireFromString("5000.00")  // Juros de financiamento PRONAF
	incomeTax := decimal.RequireFromString("8000.00")          // Tributos sobre o lucro

	// Cálculos esperados
	expectedNetRevenue := grossRevenue.Sub(deductions)                                          // 235.000,00
	expectedGrossProfit := expectedNetRevenue.Sub(cmv)                                          // 145.000,00
	expectedOperatingResult := expectedGrossProfit.Sub(operatingExpenses).Add(financialRevenues).Sub(financialExpenses) // 98.000,00
	expectedNetIncome := expectedOperatingResult.Sub(incomeTax)                                // 90.000,00

	dre := domain.DREReport{
		GrossRevenue:       grossRevenue,
		Deductions:         deductions,
		NetRevenue:         expectedNetRevenue,
		CostOfGoodsSold:    cmv,
		GrossProfit:        expectedGrossProfit,
		OperatingExpenses:  operatingExpenses,
		FinancialRevenues:  financialRevenues,
		FinancialExpenses:  financialExpenses,
		OperatingResult:    expectedOperatingResult,
		IncomeTaxProvision: incomeTax,
		NetIncome:          expectedNetIncome,
	}

	if !dre.NetRevenue.Equal(decimal.RequireFromString("235000.00")) {
		t.Errorf("Receita Líquida incorreta: %s", dre.NetRevenue)
	}
	if !dre.GrossProfit.Equal(decimal.RequireFromString("145000.00")) {
		t.Errorf("Lucro Bruto incorreto: %s", dre.GrossProfit)
	}
	if !dre.OperatingResult.Equal(decimal.RequireFromString("98000.00")) {
		t.Errorf("Resultado Operacional incorreto: %s", dre.OperatingResult)
	}
	if !dre.NetIncome.Equal(decimal.RequireFromString("90000.00")) {
		t.Errorf("Lucro Líquido do Exercício incorreto: %s", dre.NetIncome)
	}
}

// TestBalanceSheetEquation valida o Teorema Fundamental do Balanço Patrimonial (Ativo = Passivo + PL)
func TestBalanceSheetEquation(t *testing.T) {
	companyID := uuid.New()
	refDate := time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC)

	currentAsset := decimal.RequireFromString("120000.00")    // Caixa, Bancos, Clientes
	nonCurrentAsset := decimal.RequireFromString("180000.00") // Tratores, Galpão, Pivô Central
	totalAsset := currentAsset.Add(nonCurrentAsset)           // 300.000,00

	currentLiability := decimal.RequireFromString("60000.00")    // Fornecedores, Salários a Pagar
	nonCurrentLiability := decimal.RequireFromString("80000.00") // Financiamento BNDES Longo Prazo
	totalLiability := currentLiability.Add(nonCurrentLiability)  // 140.000,00

	capitalSocial := decimal.RequireFromString("100000.00") // Capital Subscrito
	lucroExercicio := decimal.RequireFromString("60000.00") // Lucro do período incorporado ao PL
	totalEquity := capitalSocial.Add(lucroExercicio)        // 160.000,00

	totalLiabilityAndEquity := totalLiability.Add(totalEquity) // 140.000 + 160.000 = 300.000,00

	bs := domain.BalanceSheetReport{
		CompanyID:               companyID,
		ReferenceDate:           refDate,
		CurrentAsset:            currentAsset,
		NonCurrentAsset:         nonCurrentAsset,
		TotalAsset:              totalAsset,
		CurrentLiability:        currentLiability,
		NonCurrentLiability:     nonCurrentLiability,
		TotalLiability:          totalLiability,
		Equity:                  capitalSocial,
		YearProfitLoss:          lucroExercicio,
		TotalEquity:             totalEquity,
		TotalLiabilityAndEquity: totalLiabilityAndEquity,
		IsBalanced:              totalAsset.Equal(totalLiabilityAndEquity),
	}

	if !bs.IsBalanced {
		t.Fatalf("O Balanço Patrimonial deveria estar em perfeito equilíbrio: Ativo (%s) != Passivo+PL (%s)",
			bs.TotalAsset, bs.TotalLiabilityAndEquity)
	}
	if !bs.TotalAsset.Equal(decimal.RequireFromString("300000.00")) {
		t.Errorf("Total do Ativo incorreto: %s", bs.TotalAsset)
	}
	if !bs.TotalLiabilityAndEquity.Equal(decimal.RequireFromString("300000.00")) {
		t.Errorf("Total Passivo + PL incorreto: %s", bs.TotalLiabilityAndEquity)
	}
}

// TestCalculateFinancialIndicators valida os cálculos de Liquidez, Endividamento, Margens, ROA e ROE
func TestCalculateFinancialIndicators(t *testing.T) {
	companyID := uuid.New()
	startDate := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC)

	bs := &domain.BalanceSheetReport{
		CompanyID:           companyID,
		ReferenceDate:       endDate,
		CurrentAsset:        decimal.RequireFromString("120000.00"),
		NonCurrentAsset:     decimal.RequireFromString("180000.00"),
		TotalAsset:          decimal.RequireFromString("300000.00"),
		CurrentLiability:    decimal.RequireFromString("60000.00"),
		NonCurrentLiability: decimal.RequireFromString("90000.00"),
		TotalLiability:      decimal.RequireFromString("150000.00"),
		TotalEquity:         decimal.RequireFromString("150000.00"),
		AssetLines: []domain.BalanceSheetLine{
			{
				AccountCode: "1.1.01",
				AccountName: "Disponibilidades",
				Level:       3,
				Amount:      decimal.RequireFromString("30000.00"),
				IsSynthetic: true,
			},
		},
	}

	dre := &domain.DREReport{
		CompanyID:       companyID,
		StartDate:       startDate,
		EndDate:         endDate,
		NetRevenue:      decimal.RequireFromString("200000.00"),
		GrossProfit:     decimal.RequireFromString("100000.00"),
		OperatingResult: decimal.RequireFromString("50000.00"),
		NetIncome:       decimal.RequireFromString("30000.00"),
	}

	if bs.ReferenceDate.IsZero() || bs.NonCurrentAsset.IsZero() || bs.NonCurrentLiability.IsZero() ||
		bs.AssetLines[0].AccountName == "" || bs.AssetLines[0].Level == 0 || dre.CompanyID == uuid.Nil {
		t.Fatal("metadados de entrada inválidos")
	}

	ind := domain.CalculateFinancialIndicators(bs, dre)

	if ind.CompanyID != companyID || ind.StartDate.IsZero() || ind.EndDate.IsZero() || ind.GeneratedAt.IsZero() {
		t.Error("metadados do relatório de indicadores não preenchidos")
	}
	if !ind.CurrentLiquidity.Equal(decimal.RequireFromString("2.00")) {
		t.Errorf("Liquidez Corrente esperada 2.00, obtida %s", ind.CurrentLiquidity)
	}
	if !ind.ImmediateLiquidity.Equal(decimal.RequireFromString("0.50")) {
		t.Errorf("Liquidez Imediata esperada 0.50, obtida %s", ind.ImmediateLiquidity)
	}
	if !ind.GeneralLiquidity.Equal(decimal.RequireFromString("2.00")) {
		t.Errorf("Liquidez Geral esperada 2.00, obtida %s", ind.GeneralLiquidity)
	}
	if !ind.GeneralIndebtednessPct.Equal(decimal.RequireFromString("50.00")) {
		t.Errorf("Endividamento Geral esperado 50.00%%, obtido %s%%", ind.GeneralIndebtednessPct)
	}
	if !ind.DebtCompositionPct.Equal(decimal.RequireFromString("40.00")) {
		t.Errorf("Composição do Endividamento esperada 40.00%%, obtida %s%%", ind.DebtCompositionPct)
	}
	if !ind.GrossMarginPct.Equal(decimal.RequireFromString("50.00")) ||
		!ind.OperatingMarginPct.Equal(decimal.RequireFromString("25.00")) ||
		!ind.NetMarginPct.Equal(decimal.RequireFromString("15.00")) {
		t.Errorf("Margens incorretas: Bruta=%s, Oper=%s, Liq=%s", ind.GrossMarginPct, ind.OperatingMarginPct, ind.NetMarginPct)
	}
	if !ind.ROAPct.Equal(decimal.RequireFromString("10.00")) || !ind.ROEPct.Equal(decimal.RequireFromString("20.00")) {
		t.Errorf("ROA/ROE incorretos: ROA=%s, ROE=%s", ind.ROAPct, ind.ROEPct)
	}
}

