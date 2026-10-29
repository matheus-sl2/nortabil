package service

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// OFXService orquestra o parsing e conciliação automática de extratos bancários
type OFXService struct {
	reconciliationRepo *postgres.ReconciliationRepository
	journalService     *JournalService
	accountRepo        *postgres.AccountRepository
}

// NewOFXService cria uma nova instância do serviço de OFX e conciliação
func NewOFXService(
	reconciliationRepo *postgres.ReconciliationRepository,
	journalService *JournalService,
	accountRepo *postgres.AccountRepository,
) *OFXService {
	return &OFXService{
		reconciliationRepo: reconciliationRepo,
		journalService:     journalService,
		accountRepo:        accountRepo,
	}
}

// ImportOFXInput parâmetros para importação de extrato bancário
type ImportOFXInput struct {
	BankAccountID        uuid.UUID `json:"bank_account_id"`        // Conta analítica do Banco no Ativo
	DefaultCounterpartID uuid.UUID `json:"default_counterpart_id"` // Conta analítica de contrapartida (ex: Clientes ou Fornecedores)
}

// ImportOFX processa o arquivo OFX, gera os lançamentos em partidas dobradas e concilia as transações
func (s *OFXService) ImportOFX(
	ctx context.Context,
	companyID uuid.UUID,
	input ImportOFXInput,
	fileReader io.Reader,
) (*domain.OFXImportSummary, error) {
	// 1. Valida conta bancária
	bankAcc, err := s.accountRepo.GetByID(ctx, input.BankAccountID)
	if err != nil {
		return nil, fmt.Errorf("conta bancária informada não encontrada: %w", err)
	}
	if bankAcc.CompanyID != companyID {
		return nil, domain.ErrMismatchedCompanyTenant
	}
	if bankAcc.IsSynthetic {
		return nil, domain.ErrSyntheticAccountPostNotAllowed
	}

	// 2. Valida conta de contrapartida
	counterpartAcc, err := s.accountRepo.GetByID(ctx, input.DefaultCounterpartID)
	if err != nil {
		return nil, fmt.Errorf("conta de contrapartida não encontrada: %w", err)
	}
	if counterpartAcc.CompanyID != companyID {
		return nil, domain.ErrMismatchedCompanyTenant
	}
	if counterpartAcc.IsSynthetic {
		return nil, domain.ErrSyntheticAccountPostNotAllowed
	}

	// 3. Faz o parsing do extrato OFX
	statement, err := domain.ParseOFX(fileReader)
	if err != nil {
		return nil, fmt.Errorf("falha ao interpretar arquivo OFX: %w", err)
	}

	summary := &domain.OFXImportSummary{
		TotalRead: len(statement.Transactions),
	}

	// 4. Processa cada movimentação bancária
	for _, trn := range statement.Transactions {
		if trn.Amount.IsZero() {
			continue
		}

		absAmount := trn.Amount.Abs()
		var bankLineType, counterpartLineType domain.JournalLineType

		// No extrato bancário:
		// Valor Positivo = Entrada/Crédito bancário -> Na contabilidade: Débito no Ativo (Banco) e Crédito na Contrapartida
		// Valor Negativo = Saída/Débito bancário -> Na contabilidade: Crédito no Ativo (Banco) e Débito na Contrapartida
		if trn.Amount.IsPositive() {
			bankLineType = domain.JournalLineDebit
			counterpartLineType = domain.JournalLineCredit
		} else {
			bankLineType = domain.JournalLineCredit
			counterpartLineType = domain.JournalLineDebit
		}

		item := &domain.BankStatementItem{
			ID:            uuid.New(),
			CompanyID:     companyID,
			BankAccountID: input.BankAccountID,
			FITID:         trn.FITID,
			Date:          trn.Date,
			Amount:        absAmount,
			Type:          bankLineType,
			Memo:          trn.Memo,
			Reconciled:    false,
		}

		inserted, err := s.reconciliationRepo.SaveStatementItem(ctx, item)
		if err != nil {
			summary.TotalFailed++
			continue
		}
		if !inserted {
			// Já importado anteriormente (duplicidade prevenida via FITID)
			summary.TotalDuplicates++
			continue
		}

		// Gera o lançamento contábil em partidas dobradas
		desc := fmt.Sprintf("Extrato Bancário (%s): %s [Doc: %s]", bankAcc.Name, trn.Memo, trn.CheckNum)
		if trn.CheckNum == "" {
			desc = fmt.Sprintf("Extrato Bancário (%s): %s", bankAcc.Name, trn.Memo)
		}

		entryInput := CreateJournalEntryInput{
			Date:        trn.Date,
			Description: desc,
			Lines: []CreateJournalLineInput{
				{
					AccountID:   input.BankAccountID,
					Type:        bankLineType,
					Amount:      absAmount,
					Description: trn.Memo,
				},
				{
					AccountID:   input.DefaultCounterpartID,
					Type:        counterpartLineType,
					Amount:      absAmount,
					Description: "Contrapartida de conciliação bancária",
				},
			},
		}

		entry, err := s.journalService.CreateEntry(ctx, companyID, entryInput)
		if err != nil {
			summary.TotalFailed++
			continue
		}

		// Marca a transação bancária como conciliada
		_ = s.reconciliationRepo.MarkReconciled(ctx, item.ID, entry.ID)
		summary.TotalImported++
	}

	return summary, nil
}

// ListItems lista os itens do extrato de uma conta bancária
func (s *OFXService) ListItems(ctx context.Context, companyID, bankAccountID uuid.UUID, onlyUnreconciled bool) ([]domain.BankStatementItem, error) {
	return s.reconciliationRepo.ListStatementItems(ctx, companyID, bankAccountID, onlyUnreconciled)
}
