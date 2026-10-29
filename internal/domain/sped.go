package domain

import (
	"time"

	"github.com/google/uuid"
)

// SPEDExportConfig define as configurações e signatários para geração da ECD
type SPEDExportConfig struct {
	CompanyID     uuid.UUID `json:"company_id"`
	StartDate     time.Time `json:"start_date"`
	EndDate       time.Time `json:"end_date"`
	BookType      string    `json:"book_type"`      // "G" - Diário Geral Completo
	AuditorCRC    string    `json:"auditor_crc"`    // Registro CRC do Contador (ex: MG-085421/O)
	AuditorName   string    `json:"auditor_name"`   // Nome do Profissional da Contabilidade
	AuditorCPF    string    `json:"auditor_cpf"`    // CPF do Contador
	SignatoryName string    `json:"signatory_name"` // Nome do Representante Legal da Empresa
	SignatoryCPF  string    `json:"signatory_cpf"`  // CPF do Representante Legal
	SignatoryQual string    `json:"signatory_qual"` // Qualificação do Assinante (ex: "207" - Administrador)
}

// SPEDRegisterCounter rastreia a contagem de linhas por tipo de registro para o Bloco 9 (9900)
type SPEDRegisterCounter struct {
	Counts map[string]int
	Order  []string
}

// NewSPEDRegisterCounter instancia um novo acumulador de registros SPED
func NewSPEDRegisterCounter() *SPEDRegisterCounter {
	return &SPEDRegisterCounter{
		Counts: make(map[string]int),
		Order:  make([]string, 0),
	}
}

// Increment adiciona a contagem de um registro
func (c *SPEDRegisterCounter) Increment(reg string) {
	if _, exists := c.Counts[reg]; !exists {
		c.Order = append(c.Order, reg)
	}
	c.Counts[reg]++
}
