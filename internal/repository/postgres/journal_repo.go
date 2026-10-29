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

// JournalRepository gerencia a persistência de lançamentos e partidas no PostgreSQL
type JournalRepository struct {
	db *DB
}

// NewJournalRepository cria uma nova instância do repositório de lançamentos
func NewJournalRepository(db *DB) *JournalRepository {
	return &JournalRepository{db: db}
}

// CreateEntry persiste um lançamento contábil e suas partidas de forma estritamente atômica (ACID).
// Utiliza transação explícita com COMMIT/ROLLBACK e delegação para triggers diferidos do PostgreSQL.
func (r *JournalRepository) CreateEntry(ctx context.Context, entry *domain.JournalEntry) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação contábil: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CompanyID == uuid.Nil {
		entry.CompanyID = domain.DefaultCompanyID
	}
	if entry.Status == "" {
		entry.Status = domain.JournalStatusPosted
	}
	if entry.CreatedBy == "" {
		entry.CreatedBy = "Sistema Contábil"
	}
	entry.CreatedAt = now
	entry.UpdatedAt = now

	// 1. Insere o cabeçalho do lançamento
	entryQuery := `
		INSERT INTO journal_entry (id, company_id, date, description, status, total_debit, total_credit, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING entry_number
	`
	err = tx.QueryRow(ctx, entryQuery,
		entry.ID,
		entry.CompanyID,
		entry.Date,
		entry.Description,
		entry.Status,
		entry.TotalDebit,
		entry.TotalCredit,
		entry.CreatedBy,
		entry.CreatedAt,
		entry.UpdatedAt,
	).Scan(&entry.EntryNumber)
	if err != nil {
		return fmt.Errorf("falha ao inserir cabeçalho do lançamento: %w", err)
	}

	// 2. Insere cada partida contábil (linhas de débito e crédito)
	lineQuery := `
		INSERT INTO journal_line (id, journal_entry_id, account_id, type, amount, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for i := range entry.Lines {
		line := &entry.Lines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}
		line.JournalEntryID = entry.ID
		line.CreatedAt = now

		_, err = tx.Exec(ctx, lineQuery,
			line.ID,
			line.JournalEntryID,
			line.AccountID,
			line.Type,
			line.Amount,
			line.Description,
			line.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("falha ao inserir partida contábil (linha %d): %w", i+1, err)
		}
	}

	// 3. Efetivação da transação (executa o trigger deferido trg_enforce_double_entry_balance)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("falha na confirmação (COMMIT) da transação contábil: %w", err)
	}

	return nil
}

// ReverseEntry marca o lançamento original como REVERSED e persiste o novo lançamento de estorno (partida inversa) na mesma transação ACID
func (r *JournalRepository) ReverseEntry(ctx context.Context, originalID uuid.UUID, reversal *domain.JournalEntry) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação de estorno: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()

	// 1. Marca o lançamento original como REVERSED (impede duplo estorno sob concorrência)
	tag, err := tx.Exec(ctx, `
		UPDATE journal_entry
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = $4
	`, domain.JournalStatusReversed, now, originalID, domain.JournalStatusPosted)
	if err != nil {
		return fmt.Errorf("falha ao atualizar status do lançamento original: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrJournalEntryAlreadyReversed
	}

	// 2. Insere o cabeçalho do lançamento de estorno
	if reversal.ID == uuid.Nil {
		reversal.ID = uuid.New()
	}
	if reversal.Status == "" {
		reversal.Status = domain.JournalStatusPosted
	}
	if reversal.CreatedBy == "" {
		reversal.CreatedBy = "Sistema Contábil"
	}
	reversal.CreatedAt = now
	reversal.UpdatedAt = now

	entryQuery := `
		INSERT INTO journal_entry (id, company_id, date, description, status, total_debit, total_credit, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING entry_number
	`
	err = tx.QueryRow(ctx, entryQuery,
		reversal.ID,
		reversal.CompanyID,
		reversal.Date,
		reversal.Description,
		reversal.Status,
		reversal.TotalDebit,
		reversal.TotalCredit,
		reversal.CreatedBy,
		reversal.CreatedAt,
		reversal.UpdatedAt,
	).Scan(&reversal.EntryNumber)
	if err != nil {
		return fmt.Errorf("falha ao inserir cabeçalho do estorno: %w", err)
	}

	// 3. Insere as partidas inversas do estorno
	lineQuery := `
		INSERT INTO journal_line (id, journal_entry_id, account_id, type, amount, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	for i := range reversal.Lines {
		line := &reversal.Lines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}
		line.JournalEntryID = reversal.ID
		line.CreatedAt = now

		_, err = tx.Exec(ctx, lineQuery,
			line.ID,
			line.JournalEntryID,
			line.AccountID,
			line.Type,
			line.Amount,
			line.Description,
			line.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("falha ao inserir partida de estorno (linha %d): %w", i+1, err)
		}
	}

	// 4. Confirma transação ACID (executando o trigger deferido do banco)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("falha na confirmação (COMMIT) do estorno contábil: %w", err)
	}

	return nil
}

// GetByID busca um lançamento pelo seu ID incluindo todas as partidas contábeis
func (r *JournalRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.JournalEntry, error) {
	entryQuery := `
		SELECT id, company_id, entry_number, date, description, status, total_debit, total_credit, COALESCE(created_by, 'Sistema Contábil'), created_at, updated_at
		FROM journal_entry
		WHERE id = $1
	`
	var entry domain.JournalEntry
	err := r.db.Pool.QueryRow(ctx, entryQuery, id).Scan(
		&entry.ID,
		&entry.CompanyID,
		&entry.EntryNumber,
		&entry.Date,
		&entry.Description,
		&entry.Status,
		&entry.TotalDebit,
		&entry.TotalCredit,
		&entry.CreatedBy,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJournalEntryNotFound
		}
		return nil, fmt.Errorf("erro ao buscar cabeçalho do lançamento: %w", err)
	}

	// Busca as partidas com os dados das contas
	linesQuery := `
		SELECT jl.id, jl.journal_entry_id, jl.account_id, a.code, a.name, jl.type, jl.amount, COALESCE(jl.description, ''), jl.created_at
		FROM journal_line jl
		JOIN account a ON a.id = jl.account_id
		WHERE jl.journal_entry_id = $1
		ORDER BY jl.type DESC, jl.amount DESC
	`
	rows, err := r.db.Pool.Query(ctx, linesQuery, id)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar partidas do lançamento: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var line domain.JournalLine
		if err := rows.Scan(
			&line.ID,
			&line.JournalEntryID,
			&line.AccountID,
			&line.AccountCode,
			&line.AccountName,
			&line.Type,
			&line.Amount,
			&line.Description,
			&line.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler partida contábil: %w", err)
		}
		entry.Lines = append(entry.Lines, line)
	}

	return &entry, nil
}

// ListFilter parâmetros para filtragem da listagem de lançamentos
type ListFilter struct {
	CompanyID uuid.UUID
	StartDate *time.Time
	EndDate   *time.Time
	Status    string
	Limit     int
	Offset    int
}

// List busca lançamentos contábeis conforme filtros de empresa, período e paginação
func (r *JournalRepository) List(ctx context.Context, f ListFilter) ([]domain.JournalEntry, int, error) {
	if f.CompanyID == uuid.Nil {
		f.CompanyID = domain.DefaultCompanyID
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	whereClause := " WHERE company_id = $1"
	args := []interface{}{f.CompanyID}
	argIdx := 2

	if f.StartDate != nil {
		whereClause += fmt.Sprintf(" AND date >= $%d", argIdx)
		args = append(args, *f.StartDate)
		argIdx++
	}
	if f.EndDate != nil {
		whereClause += fmt.Sprintf(" AND date <= $%d", argIdx)
		args = append(args, *f.EndDate)
		argIdx++
	}
	if f.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, f.Status)
		argIdx++
	}

	// Contagem total para metadados de paginação
	countQuery := "SELECT COUNT(*) FROM journal_entry" + whereClause
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("erro ao contar lançamentos: %w", err)
	}

	// Busca com paginação
	query := `
		SELECT id, company_id, entry_number, date, description, status, total_debit, total_credit, COALESCE(created_by, 'Sistema Contábil'), created_at, updated_at
		FROM journal_entry
	` + whereClause + fmt.Sprintf(" ORDER BY date DESC, entry_number DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, f.Limit, f.Offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao listar lançamentos: %w", err)
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		var e domain.JournalEntry
		if err := rows.Scan(
			&e.ID,
			&e.CompanyID,
			&e.EntryNumber,
			&e.Date,
			&e.Description,
			&e.Status,
			&e.TotalDebit,
			&e.TotalCredit,
			&e.CreatedBy,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("erro ao ler cabeçalho: %w", err)
		}
		entries = append(entries, e)
	}

	// Carrega as partidas dobradas de cada lançamento listado
	linesQuery := `
		SELECT jl.id, jl.journal_entry_id, jl.account_id, a.code, a.name, jl.type, jl.amount, COALESCE(jl.description, ''), jl.created_at
		FROM journal_line jl
		JOIN account a ON a.id = jl.account_id
		WHERE jl.journal_entry_id = $1
		ORDER BY jl.type DESC, jl.amount DESC
	`
	for i := range entries {
		lRows, err := r.db.Pool.Query(ctx, linesQuery, entries[i].ID)
		if err != nil {
			continue
		}
		for lRows.Next() {
			var line domain.JournalLine
			if err := lRows.Scan(
				&line.ID,
				&line.JournalEntryID,
				&line.AccountID,
				&line.AccountCode,
				&line.AccountName,
				&line.Type,
				&line.Amount,
				&line.Description,
				&line.CreatedAt,
			); err == nil {
				entries[i].Lines = append(entries[i].Lines, line)
			}
		}
		lRows.Close()
	}

	return entries, total, nil
}

// GetEntriesForPeriod busca todos os lançamentos com suas partidas no intervalo de datas em ordem cronológica
func (r *JournalRepository) GetEntriesForPeriod(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) ([]domain.JournalEntry, error) {
	if companyID == uuid.Nil {
		companyID = domain.DefaultCompanyID
	}

	entriesQuery := `
		SELECT id, company_id, entry_number, date, description, status, total_debit, total_credit, COALESCE(created_by, 'Sistema Contábil'), created_at, updated_at
		FROM journal_entry
		WHERE company_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC, entry_number ASC
	`
	rows, err := r.db.Pool.Query(ctx, entriesQuery, companyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar lançamentos do período: %w", err)
	}
	defer rows.Close()

	entryMap := make(map[uuid.UUID]*domain.JournalEntry)
	var orderedEntries []domain.JournalEntry

	for rows.Next() {
		var e domain.JournalEntry
		if err := rows.Scan(
			&e.ID,
			&e.CompanyID,
			&e.EntryNumber,
			&e.Date,
			&e.Description,
			&e.Status,
			&e.TotalDebit,
			&e.TotalCredit,
			&e.CreatedBy,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler lançamento: %w", err)
		}
		e.Lines = make([]domain.JournalLine, 0)
		orderedEntries = append(orderedEntries, e)
	}

	for i := range orderedEntries {
		entryMap[orderedEntries[i].ID] = &orderedEntries[i]
	}

	if len(orderedEntries) == 0 {
		return orderedEntries, nil
	}

	linesQuery := `
		SELECT jl.id, jl.journal_entry_id, jl.account_id, a.code, a.name, jl.type, jl.amount, COALESCE(jl.description, ''), jl.created_at
		FROM journal_line jl
		JOIN journal_entry je ON je.id = jl.journal_entry_id
		JOIN account a ON a.id = jl.account_id
		WHERE je.company_id = $1 AND je.date >= $2 AND je.date <= $3
		ORDER BY je.date ASC, je.entry_number ASC, jl.type DESC, jl.amount DESC
	`
	lineRows, err := r.db.Pool.Query(ctx, linesQuery, companyID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar partidas do período: %w", err)
	}
	defer lineRows.Close()

	for lineRows.Next() {
		var l domain.JournalLine
		if err := lineRows.Scan(
			&l.ID,
			&l.JournalEntryID,
			&l.AccountID,
			&l.AccountCode,
			&l.AccountName,
			&l.Type,
			&l.Amount,
			&l.Description,
			&l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler partida do período: %w", err)
		}

		if entryPtr, ok := entryMap[l.JournalEntryID]; ok {
			entryPtr.Lines = append(entryPtr.Lines, l)
		}
	}

	return orderedEntries, nil
}

