package service

import (
	"context"
	"fmt"
	"io"
	"strings"

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

// ListItems lista os itens do extrato de uma conta bancária, aplicando sugestões de regras nos pendentes
func (s *OFXService) ListItems(ctx context.Context, companyID, bankAccountID uuid.UUID, onlyUnreconciled bool) ([]domain.BankStatementItem, error) {
	items, err := s.reconciliationRepo.ListStatementItems(ctx, companyID, bankAccountID, onlyUnreconciled)
	if err != nil {
		return nil, err
	}

	rules, err := s.reconciliationRepo.ListRules(ctx, companyID)
	if err != nil {
		rules = nil
	}

	for i := range items {
		dir := domain.DirectionOutflow
		if items[i].Type == domain.JournalLineDebit {
			dir = domain.DirectionInflow
		}
		items[i].Direction = dir

		if !items[i].Reconciled && len(rules) > 0 {
			if matched := domain.MatchReconciliationRule(items[i].Memo, dir, rules); matched != nil {
				accID := matched.CounterpartAccountID
				items[i].SuggestedAccountID = &accID
				items[i].SuggestedRulePattern = matched.Pattern
			}
		}
	}

	return items, nil
}

// UploadOFXForTriage faz o parsing do arquivo OFX e salva os itens como pendentes para classificação individual na tela
func (s *OFXService) UploadOFXForTriage(
	ctx context.Context,
	companyID uuid.UUID,
	bankAccountID uuid.UUID,
	fileReader io.Reader,
) (*domain.OFXImportSummary, error) {
	bankAcc, err := s.accountRepo.GetByID(ctx, bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("conta bancária informada não encontrada: %w", err)
	}
	if bankAcc.CompanyID != companyID {
		return nil, domain.ErrMismatchedCompanyTenant
	}
	if bankAcc.IsSynthetic {
		return nil, domain.ErrSyntheticAccountPostNotAllowed
	}

	statement, err := domain.ParseOFX(fileReader)
	if err != nil {
		return nil, fmt.Errorf("falha ao interpretar arquivo OFX: %w", err)
	}

	summary := &domain.OFXImportSummary{
		TotalRead: len(statement.Transactions),
	}

	for _, trn := range statement.Transactions {
		if trn.Amount.IsZero() {
			continue
		}

		absAmount := trn.Amount.Abs()
		// No extrato bancário vs Contabilidade:
		// Entrada de dinheiro (TRNAMT > 0) = Débito na conta Banco (Ativo)
		// Saída de dinheiro (TRNAMT < 0) = Crédito na conta Banco (Ativo)
		bankLineType := domain.JournalLineCredit
		if trn.Amount.IsPositive() {
			bankLineType = domain.JournalLineDebit
		}

		item := &domain.BankStatementItem{
			ID:            uuid.New(),
			CompanyID:     companyID,
			BankAccountID: bankAccountID,
			FITID:         trn.FITID,
			Date:          trn.Date,
			Amount:        absAmount,
			Type:          bankLineType,
			Memo:          trn.Memo,
			CheckNum:      trn.CheckNum,
			Reconciled:    false,
		}

		inserted, err := s.reconciliationRepo.SaveStatementItem(ctx, item)
		if err != nil {
			summary.TotalFailed++
			continue
		}
		if !inserted {
			summary.TotalDuplicates++
			continue
		}
		summary.TotalImported++
	}

	return summary, nil
}

// ConfirmReconciliationItemInput representa a classificação individual definida pelo contador antes de efetivar
type ConfirmReconciliationItemInput struct {
	ItemID               uuid.UUID `json:"item_id"`
	CounterpartAccountID uuid.UUID `json:"counterpart_account_id"`
	Description          string    `json:"description"`
	LearnRule            bool      `json:"learn_rule"`
	RulePattern          string    `json:"rule_pattern,omitempty"`
}

// ConfirmReconciliationSummary resultado da contabilização em lote após triagem
type ConfirmReconciliationSummary struct {
	TotalRequested int      `json:"total_requested"`
	TotalConfirmed int      `json:"total_confirmed"`
	TotalSkipped   int      `json:"total_skipped"`
	TotalFailed    int      `json:"total_failed"`
	RulesLearned   int      `json:"rules_learned"`
	Errors         []string `json:"errors,omitempty"`
}

// ConfirmReconciliation contabiliza em partidas dobradas os itens classificados e opcionalmente memoriza os padrões
func (s *OFXService) ConfirmReconciliation(
	ctx context.Context,
	companyID uuid.UUID,
	items []ConfirmReconciliationItemInput,
) (*ConfirmReconciliationSummary, error) {
	summary := &ConfirmReconciliationSummary{
		TotalRequested: len(items),
	}

	for _, req := range items {
		if req.ItemID == uuid.Nil || req.CounterpartAccountID == uuid.Nil {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, "item_id e counterpart_account_id são obrigatórios")
			continue
		}

		item, err := s.reconciliationRepo.GetStatementItemByID(ctx, companyID, req.ItemID)
		if err != nil {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("item %s não encontrado", req.ItemID))
			continue
		}
		if item.Reconciled {
			summary.TotalSkipped++
			continue
		}

		bankAcc, err := s.accountRepo.GetByID(ctx, item.BankAccountID)
		if err != nil || bankAcc.CompanyID != companyID || bankAcc.IsSynthetic {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("conta bancária inválida para item %s", item.FITID))
			continue
		}

		counterpartAcc, err := s.accountRepo.GetByID(ctx, req.CounterpartAccountID)
		if err != nil {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("conta de contrapartida não encontrada para item %s", item.FITID))
			continue
		}
		if counterpartAcc.CompanyID != companyID {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, domain.ErrMismatchedCompanyTenant.Error())
			continue
		}
		if counterpartAcc.IsSynthetic {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, domain.ErrSyntheticAccountPostNotAllowed.Error())
			continue
		}

		// Regra contábil estrita (inversa à nomenclatura do extrato bancário):
		// - Entrada no Banco (item.Type == DEBIT): Débito no Banco (Ativo) e Crédito na Contrapartida
		// - Saída do Banco (item.Type == CREDIT): Crédito no Banco (Ativo) e Débito na Contrapartida
		var bankLineType, counterpartLineType domain.JournalLineType
		var direction string
		if item.Type == domain.JournalLineDebit {
			bankLineType = domain.JournalLineDebit
			counterpartLineType = domain.JournalLineCredit
			direction = domain.DirectionInflow
		} else {
			bankLineType = domain.JournalLineCredit
			counterpartLineType = domain.JournalLineDebit
			direction = domain.DirectionOutflow
		}

		desc := strings.TrimSpace(req.Description)
		if desc == "" {
			if item.CheckNum != "" {
				desc = fmt.Sprintf("Extrato Bancário (%s): %s [Doc: %s]", bankAcc.Name, item.Memo, item.CheckNum)
			} else {
				desc = fmt.Sprintf("Extrato Bancário (%s): %s", bankAcc.Name, item.Memo)
			}
		}

		entryInput := CreateJournalEntryInput{
			Date:        item.Date,
			Description: desc,
			Lines: []CreateJournalLineInput{
				{
					AccountID:   item.BankAccountID,
					Type:        bankLineType,
					Amount:      item.Amount,
					Description: item.Memo,
				},
				{
					AccountID:   counterpartAcc.ID,
					Type:        counterpartLineType,
					Amount:      item.Amount,
					Description: desc,
				},
			},
		}

		entry, err := s.journalService.CreateEntry(ctx, companyID, entryInput)
		if err != nil {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("erro ao contabilizar %s: %v", item.FITID, err))
			continue
		}

		if err := s.reconciliationRepo.MarkReconciledWithCounterpart(ctx, item.ID, entry.ID, counterpartAcc.ID); err != nil {
			summary.TotalFailed++
			summary.Errors = append(summary.Errors, fmt.Sprintf("erro ao marcar item %s como conciliado: %v", item.FITID, err))
			continue
		}
		summary.TotalConfirmed++

		if req.LearnRule {
			pattern := strings.ToUpper(strings.TrimSpace(req.RulePattern))
			if pattern == "" {
				pattern = domain.NormalizeMemoPattern(item.Memo)
			}
			if pattern != "" {
				if err := s.reconciliationRepo.UpsertRule(ctx, companyID, pattern, direction, counterpartAcc.ID); err == nil {
					summary.RulesLearned++
				}
			}
		}
	}

	return summary, nil
}

// ListRules lista as regras memorizadas de conciliação bancária da empresa
func (s *OFXService) ListRules(ctx context.Context, companyID uuid.UUID) ([]domain.ReconciliationRule, error) {
	return s.reconciliationRepo.ListRules(ctx, companyID)
}

// DeleteRule remove uma regra memorizada
func (s *OFXService) DeleteRule(ctx context.Context, companyID, ruleID uuid.UUID) error {
	return s.reconciliationRepo.DeleteRule(ctx, companyID, ruleID)
}
