package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// AccountRepository gerencia a persistência de contas contábeis no PostgreSQL
type AccountRepository struct {
	db *DB
}

// NewAccountRepository cria uma nova instância do repositório de contas
func NewAccountRepository(db *DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// Create insere uma nova conta contábil
func (r *AccountRepository) Create(ctx context.Context, acc *domain.Account) error {
	query := `
		INSERT INTO account (id, code, name, type, nature, is_synthetic, parent_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	now := time.Now()
	if acc.ID == uuid.Nil {
		acc.ID = uuid.New()
	}
	acc.CreatedAt = now
	acc.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		acc.ID,
		acc.Code,
		acc.Name,
		acc.Type,
		acc.Nature,
		acc.IsSynthetic,
		acc.ParentID,
		acc.Active,
		acc.CreatedAt,
		acc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("falha ao inserir conta contábil: %w", err)
	}

	return nil
}

// GetByID busca uma conta pelo seu identificador único UUID
func (r *AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, code, name, type, nature, is_synthetic, parent_id, active, created_at, updated_at
		FROM account
		WHERE id = $1
	`
	var acc domain.Account
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&acc.ID,
		&acc.Code,
		&acc.Name,
		&acc.Type,
		&acc.Nature,
		&acc.IsSynthetic,
		&acc.ParentID,
		&acc.Active,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("erro ao buscar conta por ID: %w", err)
	}

	return &acc, nil
}

// GetByCode busca uma conta pelo código estruturado (ex: "1.1.01.001")
func (r *AccountRepository) GetByCode(ctx context.Context, code string) (*domain.Account, error) {
	query := `
		SELECT id, code, name, type, nature, is_synthetic, parent_id, active, created_at, updated_at
		FROM account
		WHERE code = $1
	`
	var acc domain.Account
	err := r.db.Pool.QueryRow(ctx, query, code).Scan(
		&acc.ID,
		&acc.Code,
		&acc.Name,
		&acc.Type,
		&acc.Nature,
		&acc.IsSynthetic,
		&acc.ParentID,
		&acc.Active,
		&acc.CreatedAt,
		&acc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, fmt.Errorf("erro ao buscar conta por código: %w", err)
	}

	return &acc, nil
}

// List retorna todas as contas ordenadas pelo código hierárquico
func (r *AccountRepository) List(ctx context.Context, activeOnly bool) ([]domain.Account, error) {
	query := `
		SELECT id, code, name, type, nature, is_synthetic, parent_id, active, created_at, updated_at
		FROM account
	`
	if activeOnly {
		query += " WHERE active = TRUE"
	}
	query += " ORDER BY code ASC"

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar contas contábeis: %w", err)
	}
	defer rows.Close()

	var accounts []domain.Account
	for rows.Next() {
		var acc domain.Account
		if err := rows.Scan(
			&acc.ID,
			&acc.Code,
			&acc.Name,
			&acc.Type,
			&acc.Nature,
			&acc.IsSynthetic,
			&acc.ParentID,
			&acc.Active,
			&acc.CreatedAt,
			&acc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler registro de conta: %w", err)
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

// HasChildren verifica se a conta possui filhas cadastradas
func (r *AccountRepository) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM account WHERE parent_id = $1)`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar filhas da conta: %w", err)
	}
	return exists, nil
}

// HasEntries verifica se a conta possui partidas contábeis associadas
func (r *AccountRepository) HasEntries(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM journal_line WHERE account_id = $1)`
	var exists bool
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erro ao verificar lançamentos da conta: %w", err)
	}
	return exists, nil
}

// SetSynthetic atualiza o flag is_synthetic de uma conta
func (r *AccountRepository) SetSynthetic(ctx context.Context, id uuid.UUID, isSynthetic bool) error {
	query := `UPDATE account SET is_synthetic = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, query, isSynthetic, id)
	return err
}
