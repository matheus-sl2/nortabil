package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// AccountType representa a classificação patrimonial e de resultado (Lei 6.404/76 e NBC TG)
type AccountType string

const (
	AccountTypeAsset     AccountType = "ASSET"     // Ativo (Bens e Direitos)
	AccountTypeLiability AccountType = "LIABILITY" // Passivo (Obrigações)
	AccountTypeEquity    AccountType = "EQUITY"    // Patrimônio Líquido
	AccountTypeRevenue   AccountType = "REVENUE"   // Receita
	AccountTypeExpense   AccountType = "EXPENSE"   // Despesa / Custo
)

// AccountNature indica a natureza contábil usual do saldo da conta
type AccountNature string

const (
	AccountNatureDebit  AccountNature = "DEBIT"  // Natureza Devedora (Ativos, Despesas)
	AccountNatureCredit AccountNature = "CREDIT" // Natureza Credora (Passivos, PL, Receitas)
)

// Account representa uma conta do Plano de Contas
type Account struct {
	ID          uuid.UUID     `json:"id"`
	CompanyID   uuid.UUID     `json:"company_id"`   // Identificador da empresa proprietária da conta (Multi-tenant)
	Code        string        `json:"code"`         // Ex: "1.1.01.001"
	Name        string        `json:"name"`         // Ex: "Caixa Geral"
	Type        AccountType   `json:"type"`         // ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
	Nature      AccountNature `json:"nature"`       // DEBIT ou CREDIT
	IsSynthetic bool          `json:"is_synthetic"` // true = agrupadora/sintética, false = analítica (aceita lançamentos)
	ParentID    *uuid.UUID    `json:"parent_id,omitempty"`
	Active      bool          `json:"active"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// AccountTreeNode representa um nó na árvore hierárquica do Plano de Contas
type AccountTreeNode struct {
	Account
	Children []*AccountTreeNode `json:"children,omitempty"`
}

// IsValidAccountType valida se o tipo contábil fornecido é suportado
func IsValidAccountType(t AccountType) bool {
	switch t {
	case AccountTypeAsset, AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue, AccountTypeExpense:
		return true
	default:
		return false
	}
}

// DeriveNature determina a natureza contábil natural a partir do tipo de conta
// De acordo com a teoria contábil clássica:
// - Ativo e Despesa: Devedora
// - Passivo, Patrimônio Líquido e Receita: Credora
func DeriveNature(t AccountType) AccountNature {
	switch t {
	case AccountTypeAsset, AccountTypeExpense:
		return AccountNatureDebit
	default:
		return AccountNatureCredit
	}
}

// NormalizeAccountCode padroniza o código contábil removendo espaços em excesso
func NormalizeAccountCode(code string) string {
	return strings.TrimSpace(code)
}

