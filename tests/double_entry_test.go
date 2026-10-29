package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestDoubleEntryBalance valida o Teorema Fundamental de Luca Pacioli (Partidas Dobradas)
func TestDoubleEntryBalance(t *testing.T) {
	accDebito := uuid.New()
	accCredito := uuid.New()

	tests := []struct {
		name          string
		lines         []domain.JournalLine
		expectSuccess bool
		expectedError error
	}{
		{
			name: "Lançamento simples balanceado (1 Débito = 1 Crédito de R$ 1.500,50)",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(1500.50),
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineCredit,
					Amount:    decimal.NewFromFloat(1500.50),
				},
			},
			expectSuccess: true,
		},
		{
			name: "Lançamento composto balanceado (1 Débito = 2 Créditos)",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(1000.00),
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineCredit,
					Amount:    decimal.NewFromFloat(400.00),
				},
				{
					AccountID: uuid.New(),
					Type:      domain.JournalLineCredit,
					Amount:    decimal.NewFromFloat(600.00),
				},
			},
			expectSuccess: true,
		},
		{
			name: "Lançamento desbalanceado por 1 centavo (R$ 100,00 Débito vs R$ 99,99 Crédito)",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(100.00),
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineCredit,
					Amount:    decimal.NewFromFloat(99.99),
				},
			},
			expectSuccess: false,
			expectedError: domain.ErrUnbalancedJournalEntry,
		},
		{
			name: "Lançamento com valor zero deve ser rejeitado",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.Zero,
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineCredit,
					Amount:    decimal.Zero,
				},
			},
			expectSuccess: false,
			expectedError: domain.ErrZeroOrNegativeAmount,
		},
		{
			name: "Lançamento com valor negativo deve ser rejeitado",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(-50.00),
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineCredit,
					Amount:    decimal.NewFromFloat(50.00),
				},
			},
			expectSuccess: false,
			expectedError: domain.ErrZeroOrNegativeAmount,
		},
		{
			name: "Lançamento com apenas uma partida deve ser rejeitado",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(100.00),
				},
			},
			expectSuccess: false,
			expectedError: domain.ErrEmptyJournalLines,
		},
		{
			name: "Lançamento somente com partidas a débito deve ser rejeitado",
			lines: []domain.JournalLine{
				{
					AccountID: accDebito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(100.00),
				},
				{
					AccountID: accCredito,
					Type:      domain.JournalLineDebit,
					Amount:    decimal.NewFromFloat(100.00),
				},
			},
			expectSuccess: false,
			expectedError: domain.ErrEmptyJournalLines,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := domain.JournalEntry{
				ID:          uuid.New(),
				Date:        time.Now(),
				Description: "Teste de validação de partidas dobradas",
				Lines:       tt.lines,
			}

			err := entry.ValidateDoubleEntry()
			if tt.expectSuccess {
				if err != nil {
					t.Fatalf("Esperava sucesso mas obteve erro: %v", err)
				}
				if !entry.TotalDebit.Equal(entry.TotalCredit) {
					t.Fatalf("Total de débitos (%v) diverge do total de créditos (%v)", entry.TotalDebit, entry.TotalCredit)
				}
			} else {
				if err == nil {
					t.Fatalf("Esperava erro (%v) mas a validação passou com sucesso", tt.expectedError)
				}
				if tt.expectedError != nil && err != tt.expectedError {
					t.Fatalf("Esperava erro '%v', mas obteve '%v'", tt.expectedError, err)
				}
			}
		})
	}
}

// TestAccountNatureDerivation testa a derivação contábil clássica (NBC / Lei 6.404/76)
func TestAccountNatureDerivation(t *testing.T) {
	tests := []struct {
		accType        domain.AccountType
		expectedNature domain.AccountNature
	}{
		{domain.AccountTypeAsset, domain.AccountNatureDebit},     // Ativo: Devedora
		{domain.AccountTypeExpense, domain.AccountNatureDebit},   // Despesa: Devedora
		{domain.AccountTypeLiability, domain.AccountNatureCredit}, // Passivo: Credora
		{domain.AccountTypeEquity, domain.AccountNatureCredit},    // PL: Credora
		{domain.AccountTypeRevenue, domain.AccountNatureCredit},   // Receita: Credora
	}

	for _, tt := range tests {
		t.Run(string(tt.accType), func(t *testing.T) {
			nature := domain.DeriveNature(tt.accType)
			if nature != tt.expectedNature {
				t.Fatalf("Para tipo %v, esperava natureza %v mas obteve %v", tt.accType, tt.expectedNature, nature)
			}
		})
	}
}

// TestPrecisionDecimalMath testa se não há perda de precisão em operações cumulativas
func TestPrecisionDecimalMath(t *testing.T) {
	// Em float64 comum, 0.1 + 0.2 != 0.3. Com shopspring/decimal, a precisão é exata!
	d1 := decimal.RequireFromString("0.1")
	d2 := decimal.RequireFromString("0.2")
	d3 := decimal.RequireFromString("0.3")

	sum := d1.Add(d2)
	if !sum.Equal(d3) {
		t.Fatalf("Erro de precisão decimal: %v + %v = %v, esperado %v", d1, d2, sum, d3)
	}

	// Simula 10.000 transações de 0,01 centavo
	cent := decimal.RequireFromString("0.01")
	accum := decimal.Zero
	for i := 0; i < 10000; i++ {
		accum = accum.Add(cent)
	}
	expectedHundred := decimal.RequireFromString("100.00")
	if !accum.Equal(expectedHundred) {
		t.Fatalf("Acúmulo de centavos divergiu: obteve %v, esperava %v", accum, expectedHundred)
	}
}

// TestJournalEntryReversalITG2000 valida o estorno contábil por partida inversa (Resolução CFC ITG 2000)
func TestJournalEntryReversalITG2000(t *testing.T) {
	accCaixa := uuid.New()
	accFornecedor := uuid.New()
	valor := decimal.RequireFromString("2750.40")

	original := domain.JournalEntry{
		ID:          uuid.New(),
		CompanyID:   domain.DefaultCompanyID,
		EntryNumber: 42,
		Date:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Description: "Pagamento de duplicata em duplicidade",
		Status:      domain.JournalStatusPosted,
		Lines: []domain.JournalLine{
			{AccountID: accFornecedor, Type: domain.JournalLineDebit, Amount: valor, Description: "Baixa fornecedor"},
			{AccountID: accCaixa, Type: domain.JournalLineCredit, Amount: valor, Description: "Saída de caixa"},
		},
	}

	if err := original.ValidateDoubleEntry(); err != nil {
		t.Fatalf("Lançamento original deveria ser válido: %v", err)
	}

	reversal, err := original.BuildReversal(original.Date, "Pagamento lançado em duplicidade", "Contador CRC-MG")
	if err != nil {
		t.Fatalf("Falha ao construir estorno contábil: %v", err)
	}

	if !reversal.TotalDebit.Equal(valor) || !reversal.TotalCredit.Equal(valor) {
		t.Fatalf("Total do estorno (%v / %v) diverge do original (%v)", reversal.TotalDebit, reversal.TotalCredit, valor)
	}

	// Verifica se as naturezas foram exatamente invertidas (Débito <-> Crédito)
	if reversal.Lines[0].Type != domain.JournalLineCredit || reversal.Lines[1].Type != domain.JournalLineDebit {
		t.Fatalf("Partidas do estorno não inverteram Débito e Crédito corretamente")
	}

	if reversal.ID == uuid.Nil || reversal.CompanyID != original.CompanyID || reversal.Status != domain.JournalStatusPosted || reversal.CreatedBy == "" {
		t.Fatalf("Metadados do lançamento de estorno inconsistentes: %+v", reversal)
	}

	// Simula marcação como REVERSED e garante bloqueio de duplo estorno
	original.Status = domain.JournalStatusReversed
	if _, err := original.BuildReversal(original.Date, "Tentativa duplicada", "Contador"); err != domain.ErrJournalEntryAlreadyReversed {
		t.Fatalf("Esperava erro ErrJournalEntryAlreadyReversed, mas obteve: %v", err)
	}
}

