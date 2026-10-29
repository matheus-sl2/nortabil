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
		INSERT INTO bank_statement_item (id, company_id, bank_account_id, fitid, date, amount, type, memo, journal_entry_id, reconciled, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
		item.JournalEntryID,
		item.Reconciled,
		item.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("erro ao registrar item de extrato: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// ListStatementItems lista transações do extrato de uma conta
func (r *ReconciliationRepository) ListStatementItems(ctx context.Context, companyID, bankAccountID uuid.UUID, onlyUnreconciled bool) ([]domain.BankStatementItem, error) {
	query := `
		SELECT id, company_id, bank_account_id, fitid, date, amount, type, memo, journal_entry_id, reconciled, created_at
		FROM bank_statement_item
		WHERE company_id = $1 AND bank_account_id = $2
	`
	if onlyUnreconciled {
		query += " AND reconciled = FALSE"
	}
	query += " ORDER BY date DESC, created_at DESC"

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

