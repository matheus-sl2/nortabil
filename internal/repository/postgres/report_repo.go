package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// ReportRepository gerencia a geração de relatórios contábeis e demonstrativos legais
type ReportRepository struct {
	db          *DB
	accountRepo *AccountRepository
}

// NewReportRepository cria uma nova instância do repositório de relatórios
func NewReportRepository(db *DB, accountRepo *AccountRepository) *ReportRepository {
	return &ReportRepository{db: db, accountRepo: accountRepo}
}

// GetLedgerReport gera o Livro Razão analítico para um período
func (r *ReportRepository) GetLedgerReport(ctx context.Context, startDate, endDate time.Time, accountID *uuid.UUID) (*domain.LedgerReport, error) {
	report := &domain.LedgerReport{
		StartDate:   startDate,
		EndDate:     endDate,
		GeneratedAt: time.Now(),
		Accounts:    []domain.AccountLedger{},
	}

	// 1. Seleciona as contas analíticas a serem processadas
	var accounts []domain.Account
	if accountID != nil {
		acc, err := r.accountRepo.GetByID(ctx, *accountID)
		if err != nil {
			return nil, err
		}
		if acc.IsSynthetic {
			return nil, domain.ErrSyntheticAccountPostNotAllowed
		}
		accounts = append(accounts, *acc)
	} else {
		all, err := r.accountRepo.List(ctx, false)
		if err != nil {
			return nil, err
		}
		for _, a := range all {
			if !a.IsSynthetic {
				accounts = append(accounts, a)
			}
		}
	}

	// 2. Para cada conta analítica, calcula o saldo inicial anterior e as movimentações do período
	for _, acc := range accounts {
		// Saldo anterior a startDate
		initialQuery := `
			SELECT 
				COALESCE(SUM(CASE WHEN jl.type = 'DEBIT' THEN jl.amount ELSE 0 END), 0) AS debits,
				COALESCE(SUM(CASE WHEN jl.type = 'CREDIT' THEN jl.amount ELSE 0 END), 0) AS credits
			FROM journal_line jl
			JOIN journal_entry je ON je.id = jl.journal_entry_id
			WHERE jl.account_id = $1 AND je.date < $2 AND je.status = 'POSTED'
		`
		var prevDebits, prevCredits decimal.Decimal
		err := r.db.Pool.QueryRow(ctx, initialQuery, acc.ID, startDate).Scan(&prevDebits, &prevCredits)
		if err != nil {
			return nil, fmt.Errorf("erro ao calcular saldo inicial da conta %s: %w", acc.Code, err)
		}

		// Cálculo do saldo inicial baseado na natureza contábil da conta
		var initialBalance decimal.Decimal
		var initialNature domain.AccountNature
		diff := prevDebits.Sub(prevCredits)
		if diff.IsPositive() {
			initialBalance = diff
			initialNature = domain.AccountNatureDebit
		} else if diff.IsNegative() {
			initialBalance = diff.Abs()
			initialNature = domain.AccountNatureCredit
		} else {
			initialBalance = decimal.Zero
			initialNature = acc.Nature
		}

		// Movimentações no período
		movQuery := `
			SELECT 
				je.date,
				je.id,
				je.entry_number,
				COALESCE(NULLIF(jl.description, ''), je.description) AS description,
				jl.type,
				jl.amount
			FROM journal_line jl
			JOIN journal_entry je ON je.id = jl.journal_entry_id
			WHERE jl.account_id = $1 AND je.date >= $2 AND je.date <= $3 AND je.status = 'POSTED'
			ORDER BY je.date ASC, je.entry_number ASC, jl.created_at ASC
		`
		rows, err := r.db.Pool.Query(ctx, movQuery, acc.ID, startDate, endDate)
		if err != nil {
			return nil, fmt.Errorf("erro ao consultar movimentações da conta %s: %w", acc.Code, err)
		}

		var movements []domain.LedgerEntryLine
		totalDebit := decimal.Zero
		totalCredit := decimal.Zero

		// Saldo corrente acumulado linha a linha
		// Convenção interna: Débito é positivo (+), Crédito é negativo (-)
		currentBalanceSign := prevDebits.Sub(prevCredits)

		for rows.Next() {
			var date time.Time
			var entryID uuid.UUID
			var entryNum int64
			var desc string
			var lineType domain.JournalLineType
			var amount decimal.Decimal

			if err := rows.Scan(&date, &entryID, &entryNum, &desc, &lineType, &amount); err != nil {
				rows.Close()
				return nil, fmt.Errorf("erro ao ler linha do razão: %w", err)
			}

			var debitAmount, creditAmount decimal.Decimal
			if lineType == domain.JournalLineDebit {
				debitAmount = amount
				totalDebit = totalDebit.Add(amount)
				currentBalanceSign = currentBalanceSign.Add(amount)
			} else {
				creditAmount = amount
				totalCredit = totalCredit.Add(amount)
				currentBalanceSign = currentBalanceSign.Sub(amount)
			}

			var runBalance decimal.Decimal
			var runNature domain.AccountNature
			if currentBalanceSign.IsPositive() {
				runBalance = currentBalanceSign
				runNature = domain.AccountNatureDebit
			} else if currentBalanceSign.IsNegative() {
				runBalance = currentBalanceSign.Abs()
				runNature = domain.AccountNatureCredit
			} else {
				runBalance = decimal.Zero
				runNature = acc.Nature
			}

			movements = append(movements, domain.LedgerEntryLine{
				Date:           date,
				JournalEntryID: entryID,
				EntryNumber:    entryNum,
				Description:    desc,
				LineType:       lineType,
				Debit:          debitAmount,
				Credit:         creditAmount,
				RunningBalance: runBalance,
				BalanceNature:  runNature,
			})
		}
		rows.Close()

		// Saldo final da conta
		var finalBalance decimal.Decimal
		var finalNature domain.AccountNature
		if currentBalanceSign.IsPositive() {
			finalBalance = currentBalanceSign
			finalNature = domain.AccountNatureDebit
		} else if currentBalanceSign.IsNegative() {
			finalBalance = currentBalanceSign.Abs()
			finalNature = domain.AccountNatureCredit
		} else {
			finalBalance = decimal.Zero
			finalNature = acc.Nature
		}

		// Se a conta não tem saldo anterior nem movimentação no período, só adicionamos se filtrada individualmente
		if accountID != nil || !initialBalance.IsZero() || len(movements) > 0 {
			report.Accounts = append(report.Accounts, domain.AccountLedger{
				AccountID:      acc.ID,
				AccountCode:    acc.Code,
				AccountName:    acc.Name,
				AccountType:    acc.Type,
				AccountNature:  acc.Nature,
				InitialBalance: initialBalance,
				InitialNature:  initialNature,
				TotalDebit:     totalDebit,
				TotalCredit:    totalCredit,
				FinalBalance:   finalBalance,
				FinalNature:    finalNature,
				Movements:      movements,
			})
		}
	}

	return report, nil
}

// GetDailyBookReport gera o Livro Diário Geral ordenado cronologicamente
func (r *ReportRepository) GetDailyBookReport(ctx context.Context, startDate, endDate time.Time) (*domain.DailyBookReport, error) {
	report := &domain.DailyBookReport{
		StartDate:    startDate,
		EndDate:      endDate,
		GeneratedAt:  time.Now(),
		TotalDebits:  decimal.Zero,
		TotalCredits: decimal.Zero,
		Entries:      []domain.JournalEntry{},
	}

	entryQuery := `
		SELECT id, entry_number, date, description, status, total_debit, total_credit, created_at, updated_at
		FROM journal_entry
		WHERE date >= $1 AND date <= $2 AND status = 'POSTED'
		ORDER BY date ASC, entry_number ASC
	`
	rows, err := r.db.Pool.Query(ctx, entryQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar lançamentos do diário: %w", err)
	}
	defer rows.Close()

	var entries []domain.JournalEntry
	for rows.Next() {
		var e domain.JournalEntry
		if err := rows.Scan(
			&e.ID,
			&e.EntryNumber,
			&e.Date,
			&e.Description,
			&e.Status,
			&e.TotalDebit,
			&e.TotalCredit,
			&e.CreatedAt,
			&e.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("erro ao ler lançamento do diário: %w", err)
		}
		report.TotalDebits = report.TotalDebits.Add(e.TotalDebit)
		report.TotalCredits = report.TotalCredits.Add(e.TotalCredit)
		entries = append(entries, e)
	}
	rows.Close()

	// Busca as partidas de todos os lançamentos
	for i := range entries {
		linesQuery := `
			SELECT jl.id, jl.journal_entry_id, jl.account_id, a.code, a.name, jl.type, jl.amount, COALESCE(jl.description, ''), jl.created_at
			FROM journal_line jl
			JOIN account a ON a.id = jl.account_id
			WHERE jl.journal_entry_id = $1
			ORDER BY jl.type DESC, jl.amount DESC
		`
		lineRows, err := r.db.Pool.Query(ctx, linesQuery, entries[i].ID)
		if err != nil {
			return nil, fmt.Errorf("erro ao buscar partidas do diário: %w", err)
		}
		for lineRows.Next() {
			var line domain.JournalLine
			if err := lineRows.Scan(
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
				lineRows.Close()
				return nil, fmt.Errorf("erro ao ler linha da partida: %w", err)
			}
			entries[i].Lines = append(entries[i].Lines, line)
		}
		lineRows.Close()
	}

	report.TotalEntries = len(entries)
	report.Entries = entries
	return report, nil
}

// GetTrialBalanceReport gera o Balancete de Verificação completo com totalização hierárquica
func (r *ReportRepository) GetTrialBalanceReport(ctx context.Context, startDate, endDate time.Time) (*domain.TrialBalanceReport, error) {
	allAccounts, err := r.accountRepo.List(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar contas para balancete: %w", err)
	}

	report := &domain.TrialBalanceReport{
		StartDate:            startDate,
		EndDate:              endDate,
		GeneratedAt:          time.Now(),
		Items:                []domain.TrialBalanceItem{},
		TotalPeriodDebit:     decimal.Zero,
		TotalPeriodCredit:    decimal.Zero,
		TotalDebtorBalance:   decimal.Zero,
		TotalCreditorBalance: decimal.Zero,
	}

	// Mapeia totais por conta analítica
	type accTotals struct {
		prevDebits   decimal.Decimal
		prevCredits  decimal.Decimal
		periodDebit  decimal.Decimal
		periodCredit decimal.Decimal
	}
	analyticTotals := make(map[uuid.UUID]*accTotals)

	// Consulta agregações do período anterior e corrente para todas as contas analíticas
	summaryQuery := `
		SELECT 
			jl.account_id,
			COALESCE(SUM(CASE WHEN je.date < $1 AND jl.type = 'DEBIT' THEN jl.amount ELSE 0 END), 0) AS prev_debits,
			COALESCE(SUM(CASE WHEN je.date < $1 AND jl.type = 'CREDIT' THEN jl.amount ELSE 0 END), 0) AS prev_credits,
			COALESCE(SUM(CASE WHEN je.date >= $1 AND je.date <= $2 AND jl.type = 'DEBIT' THEN jl.amount ELSE 0 END), 0) AS period_debits,
			COALESCE(SUM(CASE WHEN je.date >= $1 AND je.date <= $2 AND jl.type = 'CREDIT' THEN jl.amount ELSE 0 END), 0) AS period_credits
		FROM journal_line jl
		JOIN journal_entry je ON je.id = jl.journal_entry_id
		WHERE je.status = 'POSTED' AND je.date <= $2
		GROUP BY jl.account_id
	`
	rows, err := r.db.Pool.Query(ctx, summaryQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter somatórios para o balancete: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var accID uuid.UUID
		t := &accTotals{}
		if err := rows.Scan(&accID, &t.prevDebits, &t.prevCredits, &t.periodDebit, &t.periodCredit); err != nil {
			return nil, fmt.Errorf("erro ao ler somatório do balancete: %w", err)
		}
		analyticTotals[accID] = t
	}

	// Para cada conta (sintética ou analítica), calculamos o valor considerando todos os descendentes com prefixo do código
	for _, acc := range allAccounts {
		item := domain.TrialBalanceItem{
			AccountID:   acc.ID,
			AccountCode: acc.Code,
			AccountName: acc.Name,
			AccountType: acc.Type,
			IsSynthetic: acc.IsSynthetic,
			Level:       len(acc.Code), // Nível de profundidade aproximado
		}

		var sumPrevDebits, sumPrevCredits, sumPeriodDebit, sumPeriodCredit decimal.Decimal

		if acc.IsSynthetic {
			// Soma de todas as contas filhas analíticas cujo código começa com o prefixo da conta sintética
			prefix := acc.Code + "."
			for _, sub := range allAccounts {
				if !sub.IsSynthetic && (sub.Code == acc.Code || len(sub.Code) > len(prefix) && sub.Code[:len(prefix)] == prefix) {
					if t, ok := analyticTotals[sub.ID]; ok {
						sumPrevDebits = sumPrevDebits.Add(t.prevDebits)
						sumPrevCredits = sumPrevCredits.Add(t.prevCredits)
						sumPeriodDebit = sumPeriodDebit.Add(t.periodDebit)
						sumPeriodCredit = sumPeriodCredit.Add(t.periodCredit)
					}
				}
			}
		} else {
			if t, ok := analyticTotals[acc.ID]; ok {
				sumPrevDebits = t.prevDebits
				sumPrevCredits = t.prevCredits
				sumPeriodDebit = t.periodDebit
				sumPeriodCredit = t.periodCredit
			}
		}

		// Saldo inicial
		prevDiff := sumPrevDebits.Sub(sumPrevCredits)
		if prevDiff.IsPositive() {
			item.InitialBalance = prevDiff
			item.InitialNature = domain.AccountNatureDebit
		} else if prevDiff.IsNegative() {
			item.InitialBalance = prevDiff.Abs()
			item.InitialNature = domain.AccountNatureCredit
		} else {
			item.InitialBalance = decimal.Zero
			item.InitialNature = acc.Nature
		}

		item.PeriodDebit = sumPeriodDebit
		item.PeriodCredit = sumPeriodCredit

		// Saldo final
		finalDiff := (sumPrevDebits.Add(sumPeriodDebit)).Sub(sumPrevCredits.Add(sumPeriodCredit))
		if finalDiff.IsPositive() {
			item.FinalBalance = finalDiff
			item.FinalNature = domain.AccountNatureDebit
		} else if finalDiff.IsNegative() {
			item.FinalBalance = finalDiff.Abs()
			item.FinalNature = domain.AccountNatureCredit
		} else {
			item.FinalBalance = decimal.Zero
			item.FinalNature = acc.Nature
		}

		// Acumula totalizadores gerais apenas das contas analíticas para evitar dupla contagem com sintéticas
		if !acc.IsSynthetic {
			report.TotalPeriodDebit = report.TotalPeriodDebit.Add(sumPeriodDebit)
			report.TotalPeriodCredit = report.TotalPeriodCredit.Add(sumPeriodCredit)

			if item.FinalNature == domain.AccountNatureDebit {
				report.TotalDebtorBalance = report.TotalDebtorBalance.Add(item.FinalBalance)
			} else {
				report.TotalCreditorBalance = report.TotalCreditorBalance.Add(item.FinalBalance)
			}
		}

		report.Items = append(report.Items, item)
	}

	// Balancete está balanceado se Total Débitos do Período == Total Créditos do Período
	// e Total Saldo Devedor == Total Saldo Credor
	report.IsBalanced = report.TotalPeriodDebit.Equal(report.TotalPeriodCredit) &&
		report.TotalDebtorBalance.Equal(report.TotalCreditorBalance)

	return report, nil
}

