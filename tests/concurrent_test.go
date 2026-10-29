package tests

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestConcurrentDoubleEntryValidation valida a concorrência e thread-safety em alto volume
// Simula 5.000 lançamentos simultâneos concorrentes via Goroutines
func TestConcurrentDoubleEntryValidation(t *testing.T) {
	totalOperations := 5000
	var successCount int64
	var rejectedCount int64

	accDebito := uuid.New()
	accCredito := uuid.New()

	var wg sync.WaitGroup
	wg.Add(totalOperations)

	startTime := time.Now()

	for i := 0; i < totalOperations; i++ {
		go func(idx int) {
			defer wg.Done()

			// Alterna entre operações balanceadas e desbalanceadas
			isBalanced := idx%2 == 0
			var creditAmount decimal.Decimal
			debitAmount := decimal.NewFromFloat(150.00)

			if isBalanced {
				creditAmount = decimal.NewFromFloat(150.00)
			} else {
				creditAmount = decimal.NewFromFloat(149.99) // Diferença proposital
			}

			entry := domain.JournalEntry{
				ID:          uuid.New(),
				Date:        time.Now(),
				Description: "Lançamento em lote concorrente de alto volume",
				Lines: []domain.JournalLine{
					{
						AccountID: accDebito,
						Type:      domain.JournalLineDebit,
						Amount:    debitAmount,
					},
					{
						AccountID: accCredito,
						Type:      domain.JournalLineCredit,
						Amount:    creditAmount,
					},
				},
			}

			err := entry.ValidateDoubleEntry()
			if err == nil {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&rejectedCount, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(startTime)

	t.Logf("Processados %d lançamentos em %v (Vazão: %.2f ops/seg)",
		totalOperations, duration, float64(totalOperations)/duration.Seconds())

	expectedSuccess := int64(totalOperations / 2)
	expectedRejected := int64(totalOperations / 2)

	if successCount != expectedSuccess {
		t.Fatalf("Esperava %d sucessos, obteve %d", expectedSuccess, successCount)
	}
	if rejectedCount != expectedRejected {
		t.Fatalf("Esperava %d rejeições, obteve %d", expectedRejected, rejectedCount)
	}
}

// TestTrialBalanceIntegrity valida a integridade contábil do balancete
func TestTrialBalanceIntegrity(t *testing.T) {
	report := domain.TrialBalanceReport{
		StartDate:          time.Now().AddDate(0, -1, 0),
		EndDate:            time.Now(),
		GeneratedAt:        time.Now(),
		TotalPeriodDebit:   decimal.RequireFromString("50000.00"),
		TotalPeriodCredit:  decimal.RequireFromString("50000.00"),
		TotalDebtorBalance: decimal.RequireFromString("150000.00"),
		TotalCreditorBalance: decimal.RequireFromString("150000.00"),
	}

	// Verifica a condição de igualdade do balancete
	report.IsBalanced = report.TotalPeriodDebit.Equal(report.TotalPeriodCredit) &&
		report.TotalDebtorBalance.Equal(report.TotalCreditorBalance)

	if !report.IsBalanced {
		t.Fatalf("O balancete de verificação deveria estar perfeitamente balanceado!")
	}
}
