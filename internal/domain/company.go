package domain

import (
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// TaxRegime representa o regime tributário da empresa
type TaxRegime string

const (
	TaxRegimeSimplesNacional TaxRegime = "SIMPLES_NACIONAL"
	TaxRegimeLucroPresumido  TaxRegime = "LUCRO_PRESUMIDO"
	TaxRegimeLucroReal       TaxRegime = "LUCRO_REAL"
	TaxRegimeMEI             TaxRegime = "MEI"
)

// DefaultCompanyID identificador fixo da empresa matriz padrão
var DefaultCompanyID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Company representa uma entidade empresarial / cliente do escritório de contabilidade
type Company struct {
	ID        uuid.UUID `json:"id"`
	CNPJ      string    `json:"cnpj"`       // Ex: "12.345.678/0001-90"
	LegalName string    `json:"legal_name"` // Razão Social
	TradeName string    `json:"trade_name"` // Nome Fantasia
	TaxRegime TaxRegime `json:"tax_regime"` // SIMPLES_NACIONAL, LUCRO_PRESUMIDO, etc.
	City      string    `json:"city"`       // Ex: Januária, Montes Claros
	State     string    `json:"state"`      // Ex: MG
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsValidTaxRegime verifica se o regime tributário informado é válido
func IsValidTaxRegime(r TaxRegime) bool {
	switch r {
	case TaxRegimeSimplesNacional, TaxRegimeLucroPresumido, TaxRegimeLucroReal, TaxRegimeMEI:
		return true
	default:
		return false
	}
}

// CleanCNPJ remove caracteres não numéricos do CNPJ
func CleanCNPJ(cnpj string) string {
	reg := regexp.MustCompile(`[^0-9]`)
	return reg.ReplaceAllString(cnpj, "")
}

// FormatCNPJ formata uma string numérica de 14 dígitos no padrão XX.XXX.XXX/XXXX-XX
func FormatCNPJ(cnpj string) string {
	c := CleanCNPJ(cnpj)
	if len(c) != 14 {
		return cnpj
	}
	return c[:2] + "." + c[2:5] + "." + c[5:8] + "/" + c[8:12] + "-" + c[12:]
}

// ValidateCNPJ aplica o algoritmo oficial de dígitos verificadores de CNPJ da Receita Federal
func ValidateCNPJ(cnpj string) bool {
	s := CleanCNPJ(cnpj)
	if len(s) != 14 {
		return false
	}

	// Rejeita sequências repetidas conhecidas (00000000000000, 11111111111111, etc.)
	allSame := true
	for i := 1; i < 14; i++ {
		if s[i] != s[0] {
			allSame = false
			break
		}
	}
	if allSame {
		return false
	}

	// 1º Dígito Verificador
	multipliers1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	sum1 := 0
	for i := 0; i < 12; i++ {
		val, _ := strconv.Atoi(string(s[i]))
		sum1 += val * multipliers1[i]
	}
	rem1 := sum1 % 11
	digit1 := 0
	if rem1 >= 2 {
		digit1 = 11 - rem1
	}
	if int(s[12]-'0') != digit1 {
		return false
	}

	// 2º Dígito Verificador
	multipliers2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	sum2 := 0
	for i := 0; i < 13; i++ {
		val, _ := strconv.Atoi(string(s[i]))
		sum2 += val * multipliers2[i]
	}
	rem2 := sum2 % 11
	digit2 := 0
	if rem2 >= 2 {
		digit2 = 11 - rem2
	}
	return int(s[13]-'0') == digit2
}
