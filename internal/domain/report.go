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
