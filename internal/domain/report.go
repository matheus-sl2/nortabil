package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LedgerEntryLine representa uma linha de movimentação no Livro Razão
type LedgerEntryLine struct {
	Date           time.Time       `json:"date"`
	JournalEntryID uuid.UUID       `json:"journal_entry_id"`
	EntryNumber    int64           `json:"entry_number"`
	Description    string          `json:"description"`
	LineType       JournalLineType `json:"line_type"` // DEBIT ou CREDIT
	Debit          decimal.Decimal `json:"debit"`
	Credit         decimal.Decimal `json:"credit"`
	RunningBalance decimal.Decimal `json:"running_balance"`
	BalanceNature  AccountNature   `json:"balance_nature"` // DEBIT ou CREDIT
}

// AccountLedger representa o Livro Razão de uma conta específica
type AccountLedger struct {
	AccountID      uuid.UUID         `json:"account_id"`
	AccountCode    string            `json:"account_code"`
	AccountName    string            `json:"account_name"`
	AccountType    AccountType       `json:"account_type"`
	AccountNature  AccountNature     `json:"account_nature"`
	InitialBalance decimal.Decimal   `json:"initial_balance"`
	InitialNature  AccountNature     `json:"initial_nature"`
	TotalDebit     decimal.Decimal   `json:"total_debit"`
	TotalCredit    decimal.Decimal   `json:"total_credit"`
	FinalBalance   decimal.Decimal   `json:"final_balance"`
	FinalNature    AccountNature     `json:"final_nature"`
	Movements      []LedgerEntryLine `json:"movements"`
}

// LedgerReport representa o relatório completo do Livro Razão
type LedgerReport struct {
	StartDate   time.Time       `json:"start_date"`
	EndDate     time.Time       `json:"end_date"`
	GeneratedAt time.Time       `json:"generated_at"`
	Accounts    []AccountLedger `json:"accounts"`
}

// DailyBookReport representa o relatório do Livro Diário Geral
type DailyBookReport struct {
	StartDate    time.Time       `json:"start_date"`
	EndDate      time.Time       `json:"end_date"`
	GeneratedAt  time.Time       `json:"generated_at"`
	TotalEntries int             `json:"total_entries"`
	TotalDebits  decimal.Decimal `json:"total_debits"`
	TotalCredits decimal.Decimal `json:"total_credits"`
	Entries      []JournalEntry  `json:"entries"`
}

// TrialBalanceItem representa uma linha no Balancete de Verificação
type TrialBalanceItem struct {
	AccountID      uuid.UUID       `json:"account_id"`
	AccountCode    string          `json:"account_code"`
	AccountName    string          `json:"account_name"`
	AccountType    AccountType     `json:"account_type"`
	IsSynthetic    bool            `json:"is_synthetic"`
	Level          int             `json:"level"`
	InitialBalance decimal.Decimal `json:"initial_balance"`
	InitialNature  AccountNature   `json:"initial_nature"`
	PeriodDebit    decimal.Decimal `json:"period_debit"`
	PeriodCredit   decimal.Decimal `json:"period_credit"`
	FinalBalance   decimal.Decimal `json:"final_balance"`
	FinalNature    AccountNature   `json:"final_nature"`
}

// TrialBalanceReport representa o relatório consolidado do Balancete de Verificação
type TrialBalanceReport struct {
	StartDate          time.Time          `json:"start_date"`
	EndDate            time.Time          `json:"end_date"`
	GeneratedAt        time.Time          `json:"generated_at"`
	Items              []TrialBalanceItem `json:"items"`
	TotalPeriodDebit   decimal.Decimal    `json:"total_period_debit"`
	TotalPeriodCredit  decimal.Decimal    `json:"total_period_credit"`
	TotalDebtorBalance decimal.Decimal    `json:"total_debtor_balance"`
	TotalCreditorBalance decimal.Decimal  `json:"total_creditor_balance"`
	IsBalanced         bool               `json:"is_balanced"`
}

// DRELine representa uma linha na Demonstração do Resultado do Exercício
type DRELine struct {
	Order       int             `json:"order"`
	Description string          `json:"description"`
	Amount      decimal.Decimal `json:"amount"`
	IsTotal     bool            `json:"is_total"`
	AccountCode string          `json:"account_code,omitempty"`
}

// DREReport representa a Demonstração do Resultado do Exercício (Lei 6.404/76 e NBC TG 26)
type DREReport struct {
	CompanyID          uuid.UUID       `json:"company_id"`
	StartDate          time.Time       `json:"start_date"`
	EndDate            time.Time       `json:"end_date"`
	GeneratedAt        time.Time       `json:"generated_at"`
	GrossRevenue       decimal.Decimal `json:"gross_revenue"`        // Receita Bruta
	Deductions         decimal.Decimal `json:"deductions"`           // (-) Deduções e Impostos sobre Vendas
	NetRevenue         decimal.Decimal `json:"net_revenue"`           // (=) Receita Líquida
	CostOfGoodsSold    decimal.Decimal `json:"cost_of_goods_sold"`   // (-) Custos (CMV / CSP)
	GrossProfit        decimal.Decimal `json:"gross_profit"`         // (=) Lucro Bruto
	OperatingExpenses  decimal.Decimal `json:"operating_expenses"`    // (-) Despesas Operacionais (Administrativas, Vendas)
	FinancialRevenues  decimal.Decimal `json:"financial_revenues"`   // (+) Receitas Financeiras
	FinancialExpenses  decimal.Decimal `json:"financial_expenses"`   // (-) Despesas Financeiras
	OperatingResult    decimal.Decimal `json:"operating_result"`     // (=) Resultado Operacional Antes dos Tributos
	IncomeTaxProvision decimal.Decimal `json:"income_tax_provision"` // (-) Provisão Tributos sobre o Lucro
	NetIncome          decimal.Decimal `json:"net_income"`           // (=) Lucro Líquido ou Prejuízo do Exercício
	Lines              []DRELine       `json:"lines"`
}

// BalanceSheetLine representa uma conta ou subtotal no Balanço Patrimonial
type BalanceSheetLine struct {
	AccountCode string          `json:"account_code"`
	AccountName string          `json:"account_name"`
	Level       int             `json:"level"`
	Amount      decimal.Decimal `json:"amount"`
	IsSynthetic bool            `json:"is_synthetic"`
}

// BalanceSheetReport representa o Balanço Patrimonial Estruturado (Lei 6.404/76 e NBC TG 26)
type BalanceSheetReport struct {
	CompanyID               uuid.UUID          `json:"company_id"`
	ReferenceDate           time.Time          `json:"reference_date"`
	GeneratedAt             time.Time          `json:"generated_at"`
	CurrentAsset            decimal.Decimal    `json:"current_asset"`
	NonCurrentAsset         decimal.Decimal    `json:"non_current_asset"`
	TotalAsset              decimal.Decimal    `json:"total_asset"`
	CurrentLiability        decimal.Decimal    `json:"current_liability"`
	NonCurrentLiability     decimal.Decimal    `json:"non_current_liability"`
	TotalLiability          decimal.Decimal    `json:"total_liability"`
	Equity                  decimal.Decimal    `json:"equity"`
	YearProfitLoss          decimal.Decimal    `json:"year_profit_loss"`
	TotalEquity             decimal.Decimal    `json:"total_equity"`
	TotalLiabilityAndEquity decimal.Decimal    `json:"total_liability_and_equity"`
	IsBalanced              bool               `json:"is_balanced"`
	AssetLines              []BalanceSheetLine `json:"asset_lines"`
	LiabilityLines          []BalanceSheetLine `json:"liability_lines"`
}

// FinancialIndicatorsReport representa o painel de Indicadores Econômico-Financeiros (Análise de Balanços)
type FinancialIndicatorsReport struct {
	CompanyID              uuid.UUID       `json:"company_id"`
	StartDate              time.Time       `json:"start_date"`
	EndDate                time.Time       `json:"end_date"`
	GeneratedAt            time.Time       `json:"generated_at"`
	CurrentLiquidity       decimal.Decimal `json:"current_liquidity"`        // Liquidez Corrente (AC / PC)
	ImmediateLiquidity     decimal.Decimal `json:"immediate_liquidity"`      // Liquidez Imediata (Disponibilidades / PC)
	GeneralLiquidity       decimal.Decimal `json:"general_liquidity"`        // Liquidez Geral (Ativo Total / Passivo Total)
	GeneralIndebtednessPct decimal.Decimal `json:"general_indebtedness_pct"` // Endividamento Geral % (Passivo Total / Ativo Total * 100)
	DebtCompositionPct     decimal.Decimal `json:"debt_composition_pct"`     // Composição do Endividamento % (PC / Passivo Total * 100)
	GrossMarginPct         decimal.Decimal `json:"gross_margin_pct"`         // Margem Bruta % (Lucro Bruto / Receita Líquida * 100)
	OperatingMarginPct     decimal.Decimal `json:"operating_margin_pct"`     // Margem Operacional % (Resultado Operacional / Receita Líquida * 100)
	NetMarginPct           decimal.Decimal `json:"net_margin_pct"`           // Margem Líquida % (Lucro Líquido / Receita Líquida * 100)
	ROAPct                 decimal.Decimal `json:"roa_pct"`                  // Retorno sobre o Ativo - ROA % (Lucro Líquido / Ativo Total * 100)
	ROEPct                 decimal.Decimal `json:"roe_pct"`                  // Retorno sobre o PL - ROE % (Lucro Líquido / PL Total * 100)
}

// CalculateFinancialIndicators computa os índices de Liquidez, Endividamento e Rentabilidade com precisão decimal
func CalculateFinancialIndicators(bs *BalanceSheetReport, dre *DREReport) *FinancialIndicatorsReport {
	hundred := decimal.NewFromInt(100)
	rep := &FinancialIndicatorsReport{
		CompanyID:   bs.CompanyID,
		StartDate:   dre.StartDate,
		EndDate:     dre.EndDate,
		GeneratedAt: time.Now().UTC(),
	}

	// Extrai disponibilidades (1.1.01) para Liquidez Imediata
	cashAndEquivalents := decimal.Zero
	for _, l := range bs.AssetLines {
		if l.AccountCode == "1.1.01" {
			cashAndEquivalents = l.Amount
			break
		}
	}
	if cashAndEquivalents.IsZero() {
		for _, l := range bs.AssetLines {
			if !l.IsSynthetic && len(l.AccountCode) >= 6 && l.AccountCode[:6] == "1.1.01" {
				cashAndEquivalents = cashAndEquivalents.Add(l.Amount)
			}
		}
	}

	// 1. Liquidez
	if bs.CurrentLiability.IsPositive() {
		rep.CurrentLiquidity = bs.CurrentAsset.DivRound(bs.CurrentLiability, 2)
		rep.ImmediateLiquidity = cashAndEquivalents.DivRound(bs.CurrentLiability, 2)
	}
	if bs.TotalLiability.IsPositive() {
		rep.GeneralLiquidity = bs.TotalAsset.DivRound(bs.TotalLiability, 2)
		rep.DebtCompositionPct = bs.CurrentLiability.Mul(hundred).DivRound(bs.TotalLiability, 2)
	}

	// 2. Endividamento e ROA
	if bs.TotalAsset.IsPositive() {
		rep.GeneralIndebtednessPct = bs.TotalLiability.Mul(hundred).DivRound(bs.TotalAsset, 2)
		rep.ROAPct = dre.NetIncome.Mul(hundred).DivRound(bs.TotalAsset, 2)
	}

	// 3. Margens sobre a Receita Líquida
	if dre.NetRevenue.IsPositive() {
		rep.GrossMarginPct = dre.GrossProfit.Mul(hundred).DivRound(dre.NetRevenue, 2)
		rep.OperatingMarginPct = dre.OperatingResult.Mul(hundred).DivRound(dre.NetRevenue, 2)
		rep.NetMarginPct = dre.NetIncome.Mul(hundred).DivRound(dre.NetRevenue, 2)
	}

	// 4. ROE (Retorno sobre o Patrimônio Líquido)
	if bs.TotalEquity.IsPositive() {
		rep.ROEPct = dre.NetIncome.Mul(hundred).DivRound(bs.TotalEquity, 2)
	}

	return rep
}


