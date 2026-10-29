package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// PeriodService gerencia regras de fechamento contábil e apuração de resultados (ARE)
type PeriodService struct {
	periodRepo  *postgres.PeriodRepository
	journalRepo *postgres.JournalRepository
	accountRepo *postgres.AccountRepository
}

// NewPeriodService cria uma nova instância do serviço de períodos
func NewPeriodService(
	periodRepo *postgres.PeriodRepository,
	journalRepo *postgres.JournalRepository,
	accountRepo *postgres.AccountRepository,
) *PeriodService {
	return &PeriodService{
		periodRepo:  periodRepo,
		journalRepo: journalRepo,
		accountRepo: accountRepo,
	}
}

// ListPeriods lista os períodos do ano
func (s *PeriodService) ListPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.AccountingPeriod, error) {
	if year <= 0 {
		year = time.Now().Year()
	}
	return s.periodRepo.ListPeriods(ctx, companyID, year)
}

// ClosePeriod fecha um período contábil mensal
func (s *PeriodService) ClosePeriod(ctx context.Context, companyID uuid.UUID, year, month int, closedBy string) error {
	if month < 1 || month > 12 {
		return domain.ErrInvalidDate
	}
	if closedBy == "" {
		closedBy = "Sistema Contábil Nortabil"
	}
	return s.periodRepo.ClosePeriod(ctx, companyID, year, month, closedBy)
}

// ReopenPeriod reabre um período previamente fechado
func (s *PeriodService) ReopenPeriod(ctx context.Context, companyID uuid.UUID, year, month int) error {
	if month < 1 || month > 12 {
		return domain.ErrInvalidDate
	}
	return s.periodRepo.ReopenPeriod(ctx, companyID, year, month)
}

// YearClosingInput dados de entrada para o encerramento do exercício
type YearClosingInput struct {
	Year            int        `json:"year"`
	ResultAccountID *uuid.UUID `json:"result_account_id,omitempty"` // Conta de Lucros ou Prejuízos Acumulados
}

// ExecuteYearClosing executa a Apuração do Resultado do Exercício (ARE)
// Zera todas as contas de Receita e Despesa e transfere o resultado líquido para o Patrimônio Líquido.
func (s *PeriodService) ExecuteYearClosing(ctx context.Context, companyID uuid.UUID, input YearClosingInput) (*domain.YearClosingSummary, error) {
	if input.Year <= 0 {
		input.Year = time.Now().Year()
	}

	// 1. Obtém a conta de destino no Patrimônio Líquido (Lucros/Prejuízos Acumulados)
	var targetAccount *domain.Account
	var err error
	if input.ResultAccountID != nil && *input.ResultAccountID != uuid.Nil {
		targetAccount, err = s.accountRepo.GetByID(ctx, *input.ResultAccountID)
		if err != nil {
			return nil, domain.ErrResultAccountNotFound
		}
	} else {
		// Busca conta padrão de Lucros Acumulados (ex: código "3.2.01" ou similar)
		all, err := s.accountRepo.List(ctx, companyID, false)
		if err != nil {
			return nil, err
		}
		for _, a := range all {
			if !a.IsSynthetic && a.Type == domain.AccountTypeEquity && (a.Code == "3.2.01" || a.Code == "3.3.01") {
				targetAccount = &a
				break
			}
		}
		if targetAccount == nil {
			return nil, domain.ErrResultAccountNotFound
		}
	}

	if targetAccount.IsSynthetic {
		return nil, domain.ErrSyntheticAccountPostNotAllowed
	}

	// 2. Consulta os saldos de todas as contas analíticas de Receita e Despesa do ano
	balances, err := s.periodRepo.GetResultAccountsBalances(ctx, companyID, input.Year)
	if err != nil {
		return nil, err
	}

	summary := &domain.YearClosingSummary{
		Year:          input.Year,
		CompanyID:     companyID,
		TotalRevenues: decimal.Zero,
		TotalExpenses: decimal.Zero,
	}

	if len(balances) == 0 {
		// Não há receitas ou despesas com saldo para encerrar
		return summary, nil
	}

	// 3. Monta o lançamento de zeramento (ARE)
	// Para cada Receita (saldo credor): Debita o valor do saldo para zerar
	// Para cada Despesa (saldo devedor): Credita o valor do saldo para zerar
	var lines []domain.JournalLine
	closingDate := time.Date(input.Year, 12, 31, 23, 59, 59, 0, time.UTC)
	entryID := uuid.New()

	for _, b := range balances {
		if b.AccountType == domain.AccountTypeRevenue {
			summary.TotalRevenues = summary.TotalRevenues.Add(b.Balance)
			lines = append(lines, domain.JournalLine{
				ID:             uuid.New(),
				JournalEntryID: entryID,
				AccountID:      b.AccountID,
				Type:           domain.JournalLineDebit, // Debita para zerar o crédito da receita
				Amount:         b.Balance,
				Description:    fmt.Sprintf("Zeramento de Receita %s para ARE %d", b.AccountCode, input.Year),
			})
		} else {
			summary.TotalExpenses = summary.TotalExpenses.Add(b.Balance)
			lines = append(lines, domain.JournalLine{
				ID:             uuid.New(),
				JournalEntryID: entryID,
				AccountID:      b.AccountID,
				Type:           domain.JournalLineCredit, // Credita para zerar o débito da despesa
				Amount:         b.Balance,
				Description:    fmt.Sprintf("Zeramento de Despesa %s para ARE %d", b.AccountCode, input.Year),
			})
		}
	}

	// 4. Calcula o Resultado Líquido: Lucro = Receitas - Despesas
	netResult := summary.TotalRevenues.Sub(summary.TotalExpenses)
	summary.NetResult = netResult
	summary.IsProfit = netResult.IsPositive()

	if netResult.IsPositive() {
		// Lucro Líquido: Credita na conta de Patrimônio Líquido
		lines = append(lines, domain.JournalLine{
			ID:             uuid.New(),
			JournalEntryID: entryID,
			AccountID:      targetAccount.ID,
			Type:           domain.JournalLineCredit,
			Amount:         netResult,
			Description:    fmt.Sprintf("Transferência do Lucro Líquido do Exercício de %d", input.Year),
		})
	} else if netResult.IsNegative() {
		// Prejuízo Líquido: Debita na conta de Patrimônio Líquido
		lines = append(lines, domain.JournalLine{
			ID:             uuid.New(),
			JournalEntryID: entryID,
			AccountID:      targetAccount.ID,
			Type:           domain.JournalLineDebit,
			Amount:         netResult.Abs(),
			Description:    fmt.Sprintf("Transferência do Prejuízo Líquido do Exercício de %d", input.Year),
		})
	}

	entry := &domain.JournalEntry{
		ID:          entryID,
		CompanyID:   companyID,
		Date:        closingDate,
		Description: fmt.Sprintf("Apuração do Resultado do Exercício (ARE) referente ao ano de %d", input.Year),
		Status:      domain.JournalStatusPosted,
		Lines:       lines,
	}

	if err := entry.ValidateDoubleEntry(); err != nil {
		return nil, fmt.Errorf("falha ao validar balanceamento de Luca Pacioli no ARE: %w", err)
	}

	if err := s.journalRepo.CreateEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("falha ao registrar lançamento contábil de encerramento do exercício: %w", err)
	}

	// 5. Fecha todos os meses do ano encerrado
	for m := 1; m <= 12; m++ {
		_ = s.periodRepo.ClosePeriod(ctx, companyID, input.Year, m, "Apuração de Resultado do Exercício")
	}

	summary.JournalEntryID = &entry.ID
	summary.JournalEntryNum = entry.EntryNumber

	return summary, nil
}

