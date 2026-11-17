package domain

import (
	"bufio"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OFXTransaction representa um item de movimentação extraído do arquivo OFX
type OFXTransaction struct {
	FITID    string          `json:"fitid"`    // Identificador único da transação bancária
	Type     string          `json:"type"`     // DEBIT ou CREDIT
	Date     time.Time       `json:"date"`     // Data da transação
	Amount   decimal.Decimal `json:"amount"`   // Valor (positivo para crédito, negativo para débito no extrato)
	Memo     string          `json:"memo"`     // Histórico bancário da transação
	CheckNum string          `json:"checknum"` // Número do documento/cheque
}

// OFXStatement dados consolidados do extrato bancário OFX
type OFXStatement struct {
	BankID       string           `json:"bank_id"`
	BranchID     string           `json:"branch_id"`
	AccountID    string           `json:"account_id"`
	Currency     string           `json:"currency"`
	StartDate    time.Time        `json:"start_date"`
	EndDate      time.Time        `json:"end_date"`
	Transactions []OFXTransaction `json:"transactions"`
}

// Direções de fluxo bancário para regras de classificação
const (
	DirectionInflow  = "INFLOW"  // Entrada de dinheiro no banco (Contábil: Débito no Banco / Crédito na Contrapartida)
	DirectionOutflow = "OUTFLOW" // Saída de dinheiro do banco (Contábil: Crédito no Banco / Débito na Contrapartida)
)

// BankStatementItem entidade de persistência da transação para conciliação
type BankStatementItem struct {
	ID                   uuid.UUID       `json:"id"`
	CompanyID            uuid.UUID       `json:"company_id"`
	BankAccountID        uuid.UUID       `json:"bank_account_id"`
	FITID                string          `json:"fitid"`
	Date                 time.Time       `json:"date"`
	Amount               decimal.Decimal `json:"amount"`
	Type                 JournalLineType `json:"type"` // Na visão da conta contábil do Banco: DEBIT (entrada) ou CREDIT (saída)
	Memo                 string          `json:"memo"`
	CheckNum             string          `json:"checknum,omitempty"`
	CounterpartAccountID *uuid.UUID      `json:"counterpart_account_id,omitempty"`
	JournalEntryID       *uuid.UUID      `json:"journal_entry_id,omitempty"`
	Reconciled           bool            `json:"reconciled"`
	CreatedAt            time.Time       `json:"created_at"`

	// Campos calculados em memória para auxiliar a triagem na UI
	Direction            string     `json:"direction,omitempty"`
	SuggestedAccountID   *uuid.UUID `json:"suggested_account_id,omitempty"`
	SuggestedRulePattern string     `json:"suggested_rule_pattern,omitempty"`
}

// ReconciliationRule representa uma regra aprendida de classificação (De-Para histórico -> conta contábil)
type ReconciliationRule struct {
	ID                     uuid.UUID `json:"id"`
	CompanyID              uuid.UUID `json:"company_id"`
	Pattern                string    `json:"pattern"`
	Direction              string    `json:"direction"` // INFLOW ou OUTFLOW
	CounterpartAccountID   uuid.UUID `json:"counterpart_account_id"`
	CounterpartAccountCode string    `json:"counterpart_account_code,omitempty"`
	CounterpartAccountName string    `json:"counterpart_account_name,omitempty"`
	HitCount               int       `json:"hit_count"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// OFXImportSummary resumo do processamento do extrato bancário
type OFXImportSummary struct {
	TotalRead       int `json:"total_read"`
	TotalImported   int `json:"total_imported"`
	TotalDuplicates int `json:"total_duplicates"`
	TotalFailed     int `json:"total_failed"`
}

var (
	reDatePattern   = regexp.MustCompile(`\b\d{2}/\d{2}(?:/\d{2,4})?\b`)
	reLongDigits    = regexp.MustCompile(`\b\d{5,}\b`)
	reMultiSpaces   = regexp.MustCompile(`\s+`)
)

// NormalizeMemoPattern extrai um padrão limpo do histórico bancário para aprendizado de regras
func NormalizeMemoPattern(memo string) string {
	clean := strings.ToUpper(strings.TrimSpace(memo))
	if clean == "" {
		return ""
	}
	clean = reDatePattern.ReplaceAllString(clean, " ")
	clean = reLongDigits.ReplaceAllString(clean, " ")
	clean = strings.Trim(reMultiSpaces.ReplaceAllString(clean, " "), " -:/")
	if len(clean) < 3 {
		return strings.ToUpper(strings.TrimSpace(memo))
	}
	return clean
}

// MatchReconciliationRule encontra a melhor regra aprendida compatível com o histórico e direção
func MatchReconciliationRule(memo string, direction string, rules []ReconciliationRule) *ReconciliationRule {
	norm := NormalizeMemoPattern(memo)
	upperMemo := strings.ToUpper(strings.TrimSpace(memo))
	if norm == "" && upperMemo == "" {
		return nil
	}

	var bestExact *ReconciliationRule
	var bestPartial *ReconciliationRule

	for i := range rules {
		r := &rules[i]
		if r.Direction != direction || r.Pattern == "" {
			continue
		}
		pat := strings.ToUpper(strings.TrimSpace(r.Pattern))

		if pat == norm || pat == upperMemo {
			if bestExact == nil || r.HitCount > bestExact.HitCount {
				bestExact = r
			}
			continue
		}

		if (norm != "" && strings.Contains(norm, pat)) || (upperMemo != "" && strings.Contains(upperMemo, pat)) {
			if bestPartial == nil ||
				len(pat) > len(bestPartial.Pattern) ||
				(len(pat) == len(bestPartial.Pattern) && r.HitCount > bestPartial.HitCount) {
				bestPartial = r
			}
		}
	}

	if bestExact != nil {
		return bestExact
	}
	return bestPartial
}

// ParseOFX faz o parsing de arquivos OFX bancários (versões 1.x SGML e 2.x XML)
func ParseOFX(r io.Reader) (*OFXStatement, error) {
	scanner := bufio.NewScanner(r)
	stmt := &OFXStatement{
		Currency:     "BRL",
		Transactions: []OFXTransaction{},
	}

	inTransaction := false
	var currentTrn OFXTransaction

	tagRegex := regexp.MustCompile(`<([A-Za-z0-9_]+)>(.*)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.Contains(line, "<STMTTRN>") {
			inTransaction = true
			currentTrn = OFXTransaction{}
			continue
		}

		if strings.Contains(line, "</STMTTRN>") {
			if inTransaction && currentTrn.FITID != "" {
				stmt.Transactions = append(stmt.Transactions, currentTrn)
			}
			inTransaction = false
			continue
		}

		matches := tagRegex.FindStringSubmatch(line)
		if len(matches) >= 3 {
			tag := strings.ToUpper(matches[1])
			val := matches[2]
			// Remove eventual tag de fechamento (ex: <MEMO>TEXTO</MEMO>)
			if closingIdx := strings.Index(val, "</"); closingIdx != -1 {
				val = val[:closingIdx]
			}
			val = strings.TrimSpace(val)

			if inTransaction {
				switch tag {
				case "TRNTYPE":
					currentTrn.Type = strings.ToUpper(val)
				case "DTPOSTED":
					currentTrn.Date = parseOFXDate(val)
				case "TRNAMT":
					valClean := strings.ReplaceAll(val, ",", ".")
					if d, err := decimal.NewFromString(valClean); err == nil {
						currentTrn.Amount = d
					}
				case "FITID":
					currentTrn.FITID = val
				case "CHECKNUM":
					currentTrn.CheckNum = val
				case "MEMO":
					currentTrn.Memo = val
				}
			} else {
				switch tag {
				case "BANKID":
					stmt.BankID = val
				case "BRANCHID":
					stmt.BranchID = val
				case "ACCTID":
					stmt.AccountID = val
				case "CURDEF":
					stmt.Currency = val
				case "DTSTART":
					stmt.StartDate = parseOFXDate(val)
				case "DTEND":
					stmt.EndDate = parseOFXDate(val)
				}
			}
		}
	}

	// Caso o arquivo termine a última transação sem tag de fechamento explícita
	if inTransaction && currentTrn.FITID != "" {
		stmt.Transactions = append(stmt.Transactions, currentTrn)
	}

	return stmt, scanner.Err()
}

// parseOFXDate converte formatos de data OFX como YYYYMMDD ou YYYYMMDDHHMMSS
func parseOFXDate(val string) time.Time {
	val = strings.TrimSpace(val)
	// Remove fuso horário se existir (ex: [-03:EST])
	if bracketIdx := strings.Index(val, "["); bracketIdx != -1 {
		val = val[:bracketIdx]
	}

	formats := []string{
		"20060102150405",
		"20060102",
		"2006-01-02",
	}

	for _, layout := range formats {
		if len(val) >= len(layout) {
			prefix := val[:len(layout)]
			if t, err := time.Parse(layout, prefix); err == nil {
				return t
			}
		}
	}

	return time.Now()
}

