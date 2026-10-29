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

// CompanyRepository gerencia a persistência de empresas/clientes no PostgreSQL
type CompanyRepository struct {
	db *DB
}

// NewCompanyRepository cria uma nova instância do repositório de empresas
func NewCompanyRepository(db *DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

// Create cadastra uma nova empresa
func (r *CompanyRepository) Create(ctx context.Context, comp *domain.Company) error {
	query := `
		INSERT INTO company (id, cnpj, legal_name, trade_name, tax_regime, city, state, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	now := time.Now()
	if comp.ID == uuid.Nil {
		comp.ID = uuid.New()
	}
	comp.CreatedAt = now
	comp.UpdatedAt = now

	_, err := r.db.Pool.Exec(ctx, query,
		comp.ID,
		comp.CNPJ,
		comp.LegalName,
		comp.TradeName,
		comp.TaxRegime,
		comp.City,
		comp.State,
		comp.Active,
		comp.CreatedAt,
		comp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("falha ao inserir empresa: %w", err)
	}

	return nil
}

// GetByID busca empresa pelo UUID
func (r *CompanyRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Company, error) {
	query := `
		SELECT id, cnpj, legal_name, trade_name, tax_regime, city, state, active, created_at, updated_at
		FROM company
		WHERE id = $1
	`
	var c domain.Company
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.CNPJ,
		&c.LegalName,
		&c.TradeName,
		&c.TaxRegime,
		&c.City,
		&c.State,
		&c.Active,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCompanyNotFound
		}
		return nil, fmt.Errorf("erro ao buscar empresa por ID: %w", err)
	}
	return &c, nil
}

// GetByCNPJ busca empresa pelo CNPJ
func (r *CompanyRepository) GetByCNPJ(ctx context.Context, cnpj string) (*domain.Company, error) {
	query := `
		SELECT id, cnpj, legal_name, trade_name, tax_regime, city, state, active, created_at, updated_at
		FROM company
		WHERE cnpj = $1
	`
	var c domain.Company
	err := r.db.Pool.QueryRow(ctx, query, cnpj).Scan(
		&c.ID,
		&c.CNPJ,
		&c.LegalName,
		&c.TradeName,
		&c.TaxRegime,
		&c.City,
		&c.State,
		&c.Active,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCompanyNotFound
		}
		return nil, fmt.Errorf("erro ao buscar empresa por CNPJ: %w", err)
	}
	return &c, nil
}

// List retorna todas as empresas cadastradas no escritório
func (r *CompanyRepository) List(ctx context.Context, activeOnly bool) ([]domain.Company, error) {
	query := `
		SELECT id, cnpj, legal_name, trade_name, tax_regime, city, state, active, created_at, updated_at
		FROM company
	`
	if activeOnly {
		query += " WHERE active = TRUE"
	}
	query += " ORDER BY legal_name ASC"

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar empresas: %w", err)
	}
	defer rows.Close()

	var companies []domain.Company
	for rows.Next() {
		var c domain.Company
		if err := rows.Scan(
			&c.ID,
			&c.CNPJ,
			&c.LegalName,
			&c.TradeName,
			&c.TaxRegime,
			&c.City,
			&c.State,
			&c.Active,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler registro de empresa: %w", err)
		}
		companies = append(companies, c)
	}
	return companies, nil
}

// CloneDefaultChartOfAccounts clona a árvore do plano de contas padrão para a nova empresa
func (r *CompanyRepository) CloneDefaultChartOfAccounts(ctx context.Context, targetCompanyID uuid.UUID) error {
	// 1. Busca todas as contas da empresa padrão ordenadas por código ASC para garantir pais antes dos filhos
	querySource := `
		SELECT code, name, type, nature, is_synthetic, parent_id, active
		FROM account
		WHERE company_id = $1
		ORDER BY code ASC
	`
	rows, err := r.db.Pool.Query(ctx, querySource, domain.DefaultCompanyID)
	if err != nil {
		return fmt.Errorf("erro ao obter plano de contas padrão: %w", err)
	}
	defer rows.Close()

	type rawAcc struct {
		code        string
		name        string
		accType     domain.AccountType
		nature      domain.AccountNature
		isSynthetic bool
		parentID    *uuid.UUID
		active      bool
	}
	var srcAccounts []rawAcc
	for rows.Next() {
		var a rawAcc
		if err := rows.Scan(&a.code, &a.name, &a.accType, &a.nature, &a.isSynthetic, &a.parentID, &a.active); err != nil {
			return fmt.Errorf("erro ao ler conta padrão: %w", err)
		}
		srcAccounts = append(srcAccounts, a)
	}
	rows.Close()

	// Mapeia código contábil -> novo UUID gerado para vincular relacionamentos pai/filho
	codeToNewID := make(map[string]uuid.UUID)

	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação para clonagem do plano de contas: %w", err)
	}
	defer tx.Rollback(ctx)

	insertQuery := `
		INSERT INTO account (id, company_id, code, name, type, nature, is_synthetic, parent_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
	`

	for _, a := range srcAccounts {
		newID := uuid.New()
		codeToNewID[a.code] = newID

		// Encontra o parent_id correspondente no novo mapa
		var newParentID *uuid.UUID
		if a.code != "1" && a.code != "2" && a.code != "3" && a.code != "4" && a.code != "5" {
			// Extrai código do pai (ex: "1.1.01.001" -> pai é "1.1.01")
			lastDot := -1
			for i := len(a.code) - 1; i >= 0; i-- {
				if a.code[i] == '.' {
					lastDot = i
					break
				}
			}
			if lastDot > 0 {
				parentCode := a.code[:lastDot]
				if pID, exists := codeToNewID[parentCode]; exists {
					newParentID = &pID
				}
			}
		}

		_, err := tx.Exec(ctx, insertQuery,
			newID,
			targetCompanyID,
			a.code,
			a.name,
			a.accType,
			a.nature,
			a.isSynthetic,
			newParentID,
			a.active,
		)
		if err != nil {
			return fmt.Errorf("falha ao clonar conta %s: %w", a.code, err)
		}
	}

	return tx.Commit(ctx)
}
