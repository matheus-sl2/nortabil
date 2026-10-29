package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// BenchmarkLucaPacioliValidation mede a vazão da validação do Teorema Fundamental das Partidas Dobradas
func BenchmarkLucaPacioliValidation(b *testing.B) {
	accDebito := uuid.New()
	accCredito := uuid.New()
	amount := decimal.NewFromFloat(1500.75)

	entry := domain.JournalEntry{
		ID:          uuid.New(),
		Date:        time.Now(),
		Description: "Lançamento contábil padrão para benchmark de partidas dobradas",
		Lines: []domain.JournalLine{
			{
				AccountID: accDebito,
				Type:      domain.JournalLineDebit,
				Amount:    amount,
			},
			{
				AccountID: accCredito,
				Type:      domain.JournalLineCredit,
				Amount:    amount,
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := entry.ValidateDoubleEntry(); err != nil {
			b.Fatalf("Falha inesperada na validação: %v", err)
		}
	}
}

// BenchmarkComplexEntryValidation mede a validação de lançamentos compostos (1 débito para múltiplos créditos)
func BenchmarkComplexEntryValidation(b *testing.B) {
	accDebito := uuid.New()
	amount1 := decimal.NewFromFloat(1000.00)
	amount2 := decimal.NewFromFloat(500.00)
	amountTotal := decimal.NewFromFloat(1500.00)

	entry := domain.JournalEntry{
		ID:          uuid.New(),
		Date:        time.Now(),
		Description: "Lançamento composto com múltiplas partidas",
		Lines: []domain.JournalLine{
			{
				AccountID: accDebito,
				Type:      domain.JournalLineDebit,
				Amount:    amountTotal,
			},
			{
				AccountID: uuid.New(),
				Type:      domain.JournalLineCredit,
				Amount:    amount1,
			},
			{
				AccountID: uuid.New(),
				Type:      domain.JournalLineCredit,
				Amount:    amount2,
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := entry.ValidateDoubleEntry(); err != nil {
			b.Fatalf("Falha inesperada na validação composta: %v", err)
		}
	}
}
