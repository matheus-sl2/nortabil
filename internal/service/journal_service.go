package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// JournalService orquestra as regras do motor de partidas dobradas
type JournalService struct {
	journalRepo *postgres.JournalRepository
	accountRepo *postgres.AccountRepository
}

// NewJournalService cria uma nova instância do serviço contábil
func NewJournalService(journalRepo *postgres.JournalRepository, accountRepo *postgres.AccountRepository) *JournalService {
	return &JournalService{
		journalRepo: journalRepo,
		accountRepo: accountRepo,
	}
}

// CreateJournalLineInput representa a partida no payload de entrada
type CreateJournalLineInput struct {
	AccountID   uuid.UUID              `json:"account_id"`
	Type        domain.JournalLineType `json:"type"`
	Amount      decimal.Decimal        `json:"amount"`
	Description string                 `json:"description,omitempty"`
}

// CreateJournalEntryInput representa o lançamento no payload de entrada
type CreateJournalEntryInput struct {
	Date        time.Time                `json:"date"`
	Description string                   `json:"description"`
	Lines       []CreateJournalLineInput `json:"lines"`
}

// BatchResult armazena o resultado do processamento de um item do lote
type BatchItemResult struct {
	Index       int                  `json:"index"`
	Success     bool                 `json:"success"`
	EntryID     *uuid.UUID           `json:"entry_id,omitempty"`
	EntryNumber int64                `json:"entry_number,omitempty"`
	Error       string               `json:"error,omitempty"`
}

// BatchProcessReport resume a execução em lote
type BatchProcessReport struct {
	TotalProcessed int               `json:"total_processed"`
	TotalSuccess   int               `json:"total_success"`
	TotalFailed    int               `json:"total_failed"`
	DurationMs     int64             `json:"duration_ms"`
	Results        []BatchItemResult `json:"results"`
}

// CreateEntry valida e registra um lançamento contábil garantindo a Lei de Luca Pacioli
func (s *JournalService) CreateEntry(ctx context.Context, input CreateJournalEntryInput) (*domain.JournalEntry, error) {
	if input.Date.IsZero() {
		return nil, domain.ErrInvalidDate
	}

	entry := &domain.JournalEntry{
		ID:          uuid.New(),
		Date:        input.Date,
		Description: input.Description,
		Status:      domain.JournalStatusPosted,
		Lines:       make([]domain.JournalLine, len(input.Lines)),
	}

	for i, l := range input.Lines {
		entry.Lines[i] = domain.JournalLine{
			ID:          uuid.New(),
			AccountID:   l.AccountID,
			Type:        l.Type,
			Amount:      l.Amount,
			Description: l.Description,
		}
	}

	entry.Clean()

	// 1. Validação Matemática da Lei de Luca Pacioli (Partidas Dobradas) em memória
	if err := entry.ValidateDoubleEntry(); err != nil {
		return nil, err
	}

	// 2. Validação Contábil de cada conta envolvida
	for _, line := range entry.Lines {
		acc, err := s.accountRepo.GetByID(ctx, line.AccountID)
		if err != nil {
			return nil, fmt.Errorf("conta %s inválida: %w", line.AccountID, err)
		}

		if !acc.Active {
			return nil, fmt.Errorf("a conta %s (%s) está inativa: %w", acc.Code, acc.Name, domain.ErrInactiveAccount)
		}

		// Lançamentos contábeis são permitidos EXCLUSIVAMENTE em contas analíticas (Resolução CFC ITG 2000)
		if acc.IsSynthetic {
			return nil, fmt.Errorf("a conta %s (%s) é sintética: %w", acc.Code, acc.Name, domain.ErrSyntheticAccountPostNotAllowed)
		}
	}

	// 3. Persistência Atômica no Banco de Dados
	if err := s.journalRepo.CreateEntry(ctx, entry); err != nil {
		return nil, err
	}

	return entry, nil
}

// CreateBatchEntries processa uma lista de lançamentos contábeis de forma concorrente e otimizada
// Ideal para fechamentos mensais de escritórios contábeis envolvendo milhares de lançamentos.
func (s *JournalService) CreateBatchEntries(ctx context.Context, entries []CreateJournalEntryInput) *BatchProcessReport {
	startTime := time.Now()
	total := len(entries)
	results := make([]BatchItemResult, total)

	// Worker pool concorrente com controle de paralelismo
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup

	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(idx int, input CreateJournalEntryInput) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			entry, err := s.CreateEntry(ctx, input)
			if err != nil {
				results[idx] = BatchItemResult{
					Index:   idx,
					Success: false,
					Error:   err.Error(),
				}
			} else {
				results[idx] = BatchItemResult{
					Index:       idx,
					Success:     true,
					EntryID:     &entry.ID,
					EntryNumber: entry.EntryNumber,
				}
			}
		}(i, entries[i])
	}

	wg.Wait()

	successCount := 0
	failedCount := 0
	for _, res := range results {
		if res.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	return &BatchProcessReport{
		TotalProcessed: total,
		TotalSuccess:   successCount,
		TotalFailed:    failedCount,
		DurationMs:     time.Since(startTime).Milliseconds(),
		Results:        results,
	}
}

// GetByID busca um lançamento pelo seu UUID
func (s *JournalService) GetByID(ctx context.Context, id uuid.UUID) (*domain.JournalEntry, error) {
	return s.journalRepo.GetByID(ctx, id)
}

// ListEntries lista lançamentos paginados
func (s *JournalService) ListEntries(ctx context.Context, f postgres.ListFilter) ([]domain.JournalEntry, int, error) {
	return s.journalRepo.List(ctx, f)
}
