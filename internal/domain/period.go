package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PeriodStatus define o estado de um período contábil mensal
type PeriodStatus string

const (
	PeriodStatusOpen   PeriodStatus = "OPEN"   // Aberto para novos lançamentos
	PeriodStatusClosed PeriodStatus = "CLOSED" // Fechado (bloqueado para lançamentos operacionais)
	PeriodStatusLocked PeriodStatus = "LOCKED" // Bloqueado e auditado (definitivo)
)

// AccountingPeriod representa um mês contábil de competência de uma empresa
type AccountingPeriod struct {
	ID        uuid.UUID    `json:"id"`
	CompanyID uuid.UUID    `json:"company_id"`
	Year      int          `json:"year"`
	Month     int          `json:"month"`
	StartDate time.Time    `json:"start_date"`
	EndDate   time.Time    `json:"end_date"`
	Status    PeriodStatus `json:"status"`
	ClosedAt  *time.Time   `json:"closed_at,omitempty"`
	ClosedBy  string       `json:"closed_by,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// YearClosingSummary resultado da Apuração do Resultado do Exercício (ARE)
type YearClosingSummary struct {
	Year             int             `json:"year"`
	CompanyID        uuid.UUID       `json:"company_id"`
	TotalRevenues    decimal.Decimal `json:"total_revenues"`
	TotalExpenses    decimal.Decimal `json:"total_expenses"`
	NetResult        decimal.Decimal `json:"net_result"`        // Positivo = Lucro Líquido, Negativo = Prejuízo Líquido
	IsProfit         bool            `json:"is_profit"`
	JournalEntryID   *uuid.UUID      `json:"journal_entry_id,omitempty"`
	JournalEntryNum  int64           `json:"journal_entry_num,omitempty"`
}
