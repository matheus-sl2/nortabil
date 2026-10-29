package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// JournalStatus representa o estado do lançamento contábil
type JournalStatus string

const (
	JournalStatusDraft    JournalStatus = "DRAFT"    // Rascunho
	JournalStatusPosted   JournalStatus = "POSTED"   // Efetivado no Livro Diário/Razão
	JournalStatusReversed JournalStatus = "REVERSED" // Estornado
)

// JournalLineType indica se a partida é a Débito ou a Crédito
type JournalLineType string

const (
	JournalLineDebit  JournalLineType = "DEBIT"  // Débito
	JournalLineCredit JournalLineType = "CREDIT" // Crédito
)

// JournalLine representa uma partida contábil individual (uma perna do lançamento)
type JournalLine struct {
	ID             uuid.UUID       `json:"id"`
	JournalEntryID uuid.UUID       `json:"journal_entry_id"`
	AccountID      uuid.UUID       `json:"account_id"`
	AccountCode    string          `json:"account_code,omitempty"`
	AccountName    string          `json:"account_name,omitempty"`
	Type           JournalLineType `json:"type"`   // DEBIT ou CREDIT
	Amount         decimal.Decimal `json:"amount"` // Precisão decimal exata (evita float64)
	Description    string          `json:"description,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// JournalEntry representa o cabeçalho de um lançamento contábil no Livro Diário
type JournalEntry struct {
	ID          uuid.UUID       `json:"id"`
	CompanyID   uuid.UUID       `json:"company_id"`   // Identificador da empresa proprietária (Multi-tenant)
	EntryNumber int64           `json:"entry_number"` // Número de ordem cronológica no Diário
	Date        time.Time       `json:"date"`         // Data de competência do fato contábil
	Description string          `json:"description"`  // Histórico da operação (Resolução CFC ITG 2000)
	Status      JournalStatus   `json:"status"`       // POSTED, DRAFT, REVERSED
	Lines       []JournalLine   `json:"lines"`        // As partidas dobradas
	TotalDebit  decimal.Decimal `json:"total_debit"`  // Totalizador de débitos
	TotalCredit decimal.Decimal `json:"total_credit"` // Totalizador de créditos
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ValidateDoubleEntry aplica a Lei de Luca Pacioli (Partidas Dobradas):
// A soma absoluta dos débitos deve ser rigorosamente igual à soma dos créditos.
// Além disso, deve haver no mínimo 2 linhas (1 Débito e 1 Crédito) e cada valor > 0.
func (e *JournalEntry) ValidateDoubleEntry() error {
	if len(e.Lines) < 2 {
		return ErrEmptyJournalLines
	}

	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	hasDebit := false
	hasCredit := false

	for i := range e.Lines {
		line := &e.Lines[i]
		if line.Amount.LessThanOrEqual(decimal.Zero) {
			return ErrZeroOrNegativeAmount
		}

		switch line.Type {
		case JournalLineDebit:
			totalDebits = totalDebits.Add(line.Amount)
			hasDebit = true
		case JournalLineCredit:
			totalCredits = totalCredits.Add(line.Amount)
			hasCredit = true
		default:
			return ErrInvalidLineType
		}
	}

	if !hasDebit || !hasCredit {
		return ErrEmptyJournalLines
	}

	// Verifica se Débitos == Créditos com precisão decimal absoluta
	if !totalDebits.Equal(totalCredits) {
		return ErrUnbalancedJournalEntry
	}

	e.TotalDebit = totalDebits
	e.TotalCredit = totalCredits
	return nil
}

// Clean prepara o lançamento aparando campos de texto
func (e *JournalEntry) Clean() {
	e.Description = strings.TrimSpace(e.Description)
	for i := range e.Lines {
		e.Lines[i].Description = strings.TrimSpace(e.Lines[i].Description)
	}
}

