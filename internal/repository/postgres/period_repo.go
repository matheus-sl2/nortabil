package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// PeriodRepository gerencia a persistência de períodos contábeis no PostgreSQL
type PeriodRepository struct {
	db *DB
}

// NewPeriodRepository cria uma nova instância do repositório de períodos
func NewPeriodRepository(db *DB) *PeriodRepository {
	return &PeriodRepository{db: db}
}

// IsPeriodOpen verifica se o período contábil correspondente à data informada está aberto para lançamentos.
// Se o período contábil do mês ainda não existir na tabela, inicializa-o automaticamente como OPEN.
func (r *PeriodRepository) IsPeriodOpen(ctx context.Context, companyID uuid.UUID, date time.Time) (bool, error) {
	year := date.Year()
	month := int(date.Month())

	query := `
		SELECT status FROM accounting_period
		WHERE company_id = $1 AND year = $2 AND month = $3
	`
	var status domain.PeriodStatus
	err := r.db.Pool.QueryRow(ctx, query, companyID, year, month).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Período ainda não cadastrado: inicializa como OPEN
			start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 1, -1)

			insertQuery := `
				INSERT INTO accounting_period (id, company_id, year, month, start_date, end_date, status, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, 'OPEN', NOW(), NOW())
				ON CONFLICT (company_id, year, month) DO UPDATE SET updated_at = NOW()
				RETURNING status
			`
			err = r.db.Pool.QueryRow(ctx, insertQuery, uuid.New(), companyID, year, month, start, end).Scan(&status)
			if err != nil {
				return false, fmt.Errorf("falha ao inicializar período contábil: %w", err)
			}
			return status == domain.PeriodStatusOpen, nil
		}
		return false, fmt.Errorf("erro ao verificar status do período contábil: %w", err)
	}

	return status == domain.PeriodStatusOpen, nil
}

// ListPeriods lista os períodos do ano de uma empresa
func (r *PeriodRepository) ListPeriods(ctx context.Context, companyID uuid.UUID, year int) ([]domain.AccountingPeriod, error) {
	// Garante que todos os 12 meses do ano existem na tabela
	for m := 1; m <= 12; m++ {
		start := time.Date(year, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, -1)
		upsert := `
			INSERT INTO accounting_period (id, company_id, year, month, start_date, end_date, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'OPEN', NOW(), NOW())
			ON CONFLICT (company_id, year, month) DO NOTHING
		`
		_, _ = r.db.Pool.Exec(ctx, upsert, uuid.New(), companyID, year, m, start, end)
	}

	query := `
		SELECT id, company_id, year, month, start_date, end_date, status, closed_at, COALESCE(closed_by, ''), created_at, updated_at
		FROM accounting_period
		WHERE company_id = $1 AND year = $2
		ORDER BY month ASC
	`
	rows, err := r.db.Pool.Query(ctx, query, companyID, year)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar períodos contábeis: %w", err)
	}
	defer rows.Close()

	var periods []domain.AccountingPeriod
	for rows.Next() {
		var p domain.AccountingPeriod
		if err := rows.Scan(
			&p.ID,
			&p.CompanyID,
			&p.Year,
			&p.Month,
			&p.StartDate,
			&p.EndDate,
			&p.Status,
			&p.ClosedAt,
			&p.ClosedBy,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler período contábil: %w", err)
		}
		periods = append(periods, p)
	}
	return periods, nil
}

// ClosePeriod encerra um período contábil mensal
func (r *PeriodRepository) ClosePeriod(ctx context.Context, companyID uuid.UUID, year, month int, closedBy string) error {
	now := time.Now()
	query := `
		UPDATE accounting_period 
		SET status = 'CLOSED', closed_at = $1, closed_by = $2, updated_at = $1
		WHERE company_id = $3 AND year = $4 AND month = $5 AND status <> 'LOCKED'
	`
	tag, err := r.db.Pool.Exec(ctx, query, now, closedBy, companyID, year, month)
	if err != nil {
		return fmt.Errorf("erro ao fechar período: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrPeriodNotFound
	}
	return nil
}

// ReopenPeriod reabre um período contábil previamente fechado
func (r *PeriodRepository) ReopenPeriod(ctx context.Context, companyID uuid.UUID, year, month int) error {
	// Verifica se está LOCKED
	var status domain.PeriodStatus
	checkQuery := `SELECT status FROM accounting_period WHERE company_id = $1 AND year = $2 AND month = $3`
	err := r.db.Pool.QueryRow(ctx, checkQuery, companyID, year, month).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPeriodNotFound
		}
		return err
	}
	if status == domain.PeriodStatusLocked {
		return domain.ErrPeriodLocked
	}
	if status == domain.PeriodStatusOpen {
		return domain.ErrPeriodAlreadyOpen
	}

	query := `
		UPDATE accounting_period 
		SET status = 'OPEN', closed_at = NULL, closed_by = NULL, updated_at = NOW()
		WHERE company_id = $1 AND year = $2 AND month = $3
	`
	_, err = r.db.Pool.Exec(ctx, query, companyID, year, month)
	return err
}

// AccountBalanceSummary resumo do saldo da conta para encerramento
type AccountBalanceSummary struct {
	AccountID   uuid.UUID
	AccountCode string
	AccountType domain.AccountType
	Balance     decimal.Decimal
}

// GetResultAccountsBalances consulta o saldo de todas as contas analíticas de Receita e Despesa para encerramento do exercício (ARE)
func (r *PeriodRepository) GetResultAccountsBalances(ctx context.Context, companyID uuid.UUID, year int) ([]AccountBalanceSummary, error) {
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	query := `
		SELECT 
			a.id, 
			a.code, 
			a.type,
			COALESCE(SUM(CASE WHEN jl.type = 'DEBIT' THEN jl.amount ELSE 0 END), 0) AS total_debit,
			COALESCE(SUM(CASE WHEN jl.type = 'CREDIT' THEN jl.amount ELSE 0 END), 0) AS total_credit
		FROM account a
		JOIN journal_line jl ON jl.account_id = a.id
		JOIN journal_entry je ON je.id = jl.journal_entry_id
		WHERE a.company_id = $1 
		  AND a.is_synthetic = FALSE
		  AND a.type IN ('REVENUE', 'EXPENSE')
		  AND je.date >= $2 AND je.date <= $3
		  AND je.status IN ('POSTED', 'REVERSED')
		GROUP BY a.id, a.code, a.type
	`
	rows, err := r.db.Pool.Query(ctx, query, companyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao apurar saldos de contas de resultado: %w", err)
	}
	defer rows.Close()

	var balances []AccountBalanceSummary
	for rows.Next() {
		var id uuid.UUID
		var code string
		var accType domain.AccountType
		var debits, credits decimal.Decimal

		if err := rows.Scan(&id, &code, &accType, &debits, &credits); err != nil {
			return nil, fmt.Errorf("erro ao ler saldo: %w", err)
		}

		var bal decimal.Decimal
		if accType == domain.AccountTypeRevenue {
			// Receita: Saldo = Créditos - Débitos
			bal = credits.Sub(debits)
		} else {
			// Despesa: Saldo = Débitos - Créditos
			bal = debits.Sub(credits)
		}

		if !bal.IsZero() {
			balances = append(balances, AccountBalanceSummary{
				AccountID:   id,
				AccountCode: code,
				AccountType: accType,
				Balance:     bal,
			})
		}
	}

	return balances, nil
}

