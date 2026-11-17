package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// ReconciliationRepository gerencia as transações bancárias e conciliação
type ReconciliationRepository struct {
	db *DB
}

// NewReconciliationRepository cria uma nova instância do repositório
func NewReconciliationRepository(db *DB) *ReconciliationRepository {
	return &ReconciliationRepository{db: db}
}

// SaveStatementItem insere uma transação bancária se o FITID não existir para aquela conta e empresa.
// Retorna (inserted bool, err error).
func (r *ReconciliationRepository) SaveStatementItem(ctx context.Context, item *domain.BankStatementItem) (bool, error) {
	query := `
		INSERT INTO bank_statement_item (
			id, company_id, bank_account_id, fitid, date, amount, type, memo, checknum, counterpart_account_id, journal_entry_id, reconciled, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (company_id, bank_account_id, fitid) DO NOTHING
	`
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	item.CreatedAt = time.Now()

	tag, err := r.db.Pool.Exec(ctx, query,
		item.ID,
		item.CompanyID,
		item.BankAccountID,
		item.FITID,
		item.Date,
		item.Amount,
		item.Type,
		item.Memo,
		item.CheckNum,
		item.CounterpartAccountID,
		item.JournalEntryID,
		item.Reconciled,
		item.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("erro ao registrar item de extrato: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// GetStatementItemByID busca um item de extrato garantindo isolamento por empresa
func (r *ReconciliationRepository) GetStatementItemByID(ctx context.Context, companyID, itemID uuid.UUID) (*domain.BankStatementItem, error) {
	query := `
		SELECT id, company_id, bank_account_id, fitid, date, amount, type, memo, COALESCE(checknum, ''), counterpart_account_id, journal_entry_id, reconciled, created_at
		FROM bank_statement_item
		WHERE company_id = $1 AND id = $2
	`
	var it domain.BankStatementItem
	err := r.db.Pool.QueryRow(ctx, query, companyID, itemID).Scan(
		&it.ID,
		&it.CompanyID,
		&it.BankAccountID,
		&it.FITID,
		&it.Date,
		&it.Amount,
		&it.Type,
		&it.Memo,
		&it.CheckNum,
		&it.CounterpartAccountID,
		&it.JournalEntryID,
		&it.Reconciled,
		&it.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("item de extrato não encontrado: %w", err)
	}
	return &it, nil
}

// ListStatementItems lista transações do extrato de uma conta
func (r *ReconciliationRepository) ListStatementItems(ctx context.Context, companyID, bankAccountID uuid.UUID, onlyUnreconciled bool) ([]domain.BankStatementItem, error) {
	query := `
		SELECT id, company_id, bank_account_id, fitid, date, amount, type, memo, COALESCE(checknum, ''), counterpart_account_id, journal_entry_id, reconciled, created_at
		FROM bank_statement_item
		WHERE company_id = $1 AND bank_account_id = $2
	`
	if onlyUnreconciled {
		query += " AND reconciled = FALSE"
	}
	query += " ORDER BY reconciled ASC, date DESC, created_at DESC"

	rows, err := r.db.Pool.Query(ctx, query, companyID, bankAccountID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar transações bancárias: %w", err)
	}
	defer rows.Close()

	var items []domain.BankStatementItem
	for rows.Next() {
		var it domain.BankStatementItem
		if err := rows.Scan(
			&it.ID,
			&it.CompanyID,
			&it.BankAccountID,
			&it.FITID,
			&it.Date,
			&it.Amount,
			&it.Type,
			&it.Memo,
			&it.CheckNum,
			&it.CounterpartAccountID,
			&it.JournalEntryID,
			&it.Reconciled,
			&it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler transação bancária: %w", err)
		}
		items = append(items, it)
	}

	return items, nil
}

// MarkReconciled vincula a transação bancária ao lançamento contábil gerado
func (r *ReconciliationRepository) MarkReconciled(ctx context.Context, itemID, journalEntryID uuid.UUID) error {
	query := `
		UPDATE bank_statement_item 
		SET reconciled = TRUE, journal_entry_id = $1
		WHERE id = $2
	`
	_, err := r.db.Pool.Exec(ctx, query, journalEntryID, itemID)
	return err
}

// MarkReconciledWithCounterpart vincula a transação bancária ao lançamento contábil e à conta de contrapartida escolhida
func (r *ReconciliationRepository) MarkReconciledWithCounterpart(ctx context.Context, itemID, journalEntryID, counterpartAccountID uuid.UUID) error {
	query := `
		UPDATE bank_statement_item 
		SET reconciled = TRUE, journal_entry_id = $1, counterpart_account_id = $2
		WHERE id = $3
	`
	_, err := r.db.Pool.Exec(ctx, query, journalEntryID, counterpartAccountID, itemID)
	return err
}

// UpsertRule salva ou atualiza uma regra de classificação (De-Para histórico -> conta contábil)
func (r *ReconciliationRepository) UpsertRule(ctx context.Context, companyID uuid.UUID, pattern, direction string, counterpartAccountID uuid.UUID) error {
	query := `
		INSERT INTO bank_reconciliation_rule (id, company_id, pattern, direction, counterpart_account_id, hit_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 1, NOW(), NOW())
		ON CONFLICT (company_id, pattern, direction)
		DO UPDATE SET
			counterpart_account_id = EXCLUDED.counterpart_account_id,
			hit_count = bank_reconciliation_rule.hit_count + 1,
			updated_at = NOW()
	`
	_, err := r.db.Pool.Exec(ctx, query, uuid.New(), companyID, pattern, direction, counterpartAccountID)
	if err != nil {
		return fmt.Errorf("erro ao salvar regra de conciliação: %w", err)
	}
	return nil
}

// ListRules lista todas as regras de classificação memorizadas da empresa
func (r *ReconciliationRepository) ListRules(ctx context.Context, companyID uuid.UUID) ([]domain.ReconciliationRule, error) {
	query := `
		SELECT r.id, r.company_id, r.pattern, r.direction, r.counterpart_account_id,
		       COALESCE(a.code, ''), COALESCE(a.name, ''), r.hit_count, r.created_at, r.updated_at
		FROM bank_reconciliation_rule r
		LEFT JOIN account a ON a.id = r.counterpart_account_id
		WHERE r.company_id = $1
		ORDER BY r.hit_count DESC, r.updated_at DESC
	`
	rows, err := r.db.Pool.Query(ctx, query, companyID)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar regras de conciliação: %w", err)
	}
	defer rows.Close()

	var rules []domain.ReconciliationRule
	for rows.Next() {
		var rule domain.ReconciliationRule
		if err := rows.Scan(
			&rule.ID,
			&rule.CompanyID,
			&rule.Pattern,
			&rule.Direction,
			&rule.CounterpartAccountID,
			&rule.CounterpartAccountCode,
			&rule.CounterpartAccountName,
			&rule.HitCount,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler regra de conciliação: %w", err)
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// DeleteRule remove uma regra de classificação memorizada
func (r *ReconciliationRepository) DeleteRule(ctx context.Context, companyID, ruleID uuid.UUID) error {
	query := `DELETE FROM bank_reconciliation_rule WHERE company_id = $1 AND id = $2`
	_, err := r.db.Pool.Exec(ctx, query, companyID, ruleID)
	return err
}

