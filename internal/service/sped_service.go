package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
	"github.com/shopspring/decimal"
)

// SPEDService gera a escrituração contábil digital oficial da Receita Federal (SPED ECD)
type SPEDService struct {
	companyRepo *postgres.CompanyRepository
	accountRepo *postgres.AccountRepository
	journalRepo *postgres.JournalRepository
}

// NewSPEDService cria uma nova instância do serviço de exportação SPED
func NewSPEDService(
	companyRepo *postgres.CompanyRepository,
	accountRepo *postgres.AccountRepository,
	journalRepo *postgres.JournalRepository,
) *SPEDService {
	return &SPEDService{
		companyRepo: companyRepo,
		accountRepo: accountRepo,
		journalRepo: journalRepo,
	}
}

// GenerateECD gera o arquivo completo do SPED Contábil (Livro Diário Geral - Forma G)
func (s *SPEDService) GenerateECD(ctx context.Context, cfg domain.SPEDExportConfig) (string, error) {
	// 1. Obtém os dados cadastrais da empresa
	company, err := s.companyRepo.GetByID(ctx, cfg.CompanyID)
	if err != nil {
		return "", fmt.Errorf("empresa não encontrada para exportação SPED: %w", err)
	}

	// 2. Busca o plano de contas da empresa
	accounts, err := s.accountRepo.List(ctx, cfg.CompanyID, false)
	if err != nil {
		return "", fmt.Errorf("falha ao carregar plano de contas da empresa: %w", err)
	}

	// 3. Busca os lançamentos e partidas contábeis do período
	entries, err := s.journalRepo.GetEntriesForPeriod(ctx, cfg.CompanyID, cfg.StartDate, cfg.EndDate)
	if err != nil {
		return "", fmt.Errorf("falha ao carregar lançamentos do período: %w", err)
	}

	return BuildSPEDFile(company, accounts, entries, cfg), nil
}

// BuildSPEDFile monta a estrutura canônica e os registros pipe-delimited da ECD
func BuildSPEDFile(
	company *domain.Company,
	accounts []domain.Account,
	entries []domain.JournalEntry,
	cfg domain.SPEDExportConfig,
) string {
	// Normaliza dados dos signatários caso não informados
	if cfg.AuditorCRC == "" {
		cfg.AuditorCRC = "MG-085421/O"
	}
	if cfg.AuditorName == "" {
		cfg.AuditorName = "CONTADOR RESPONSAVEL TECNICO CRC/MG"
	}
	if cfg.AuditorCPF == "" {
		cfg.AuditorCPF = "00000000000"
	}
	if cfg.SignatoryName == "" {
		cfg.SignatoryName = company.LegalName
	}
	if cfg.SignatoryCPF == "" {
		cfg.SignatoryCPF = "00000000000"
	}
	if cfg.SignatoryQual == "" {
		cfg.SignatoryQual = "207" // Administrador
	}

	digitsOnly := regexp.MustCompile(`\D`)
	cnpjClean := digitsOnly.ReplaceAllString(company.CNPJ, "")
	auditorCpfClean := digitsOnly.ReplaceAllString(cfg.AuditorCPF, "")
	signatoryCpfClean := digitsOnly.ReplaceAllString(cfg.SignatoryCPF, "")

	dtIni := cfg.StartDate.Format("02012006")
	dtFin := cfg.EndDate.Format("02012006")

	// Código IBGE do município (ex: Januária: 3135209, Montes Claros: 3143302)
	codMun := "3135209" // Januária - MG (Padrão do projeto)
	if strings.EqualFold(company.City, "Montes Claros") {
		codMun = "3143302"
	}

	var sb strings.Builder
	counter := domain.NewSPEDRegisterCounter()

	writeLine := func(reg string, fields ...string) {
		sb.WriteString("|")
		sb.WriteString(reg)
		for _, f := range fields {
			sb.WriteString("|")
			sb.WriteString(f)
		}
		sb.WriteString("|\r\n")
		counter.Increment(reg)
	}

	// ==========================================
	// BLOCO 0: ABERTURA E IDENTIFICAÇÃO DA EMPRESA
	// ==========================================
	writeLine("0000", "LECD", dtIni, dtFin, strings.ToUpper(company.LegalName), cnpjClean, company.State, "", codMun, "", "0", "", "1", "0", "0", "", "")
	writeLine("0001", "0")
	writeLine("0007", "", "")

	// 0990: Encerramento do Bloco 0 (calculado incluindo o próprio 0990)
	bloco0Count := counter.Counts["0000"] + counter.Counts["0001"] + counter.Counts["0007"] + 1
	writeLine("0990", fmt.Sprintf("%d", bloco0Count))

	// ==========================================
	// BLOCO I: LIVRO DIÁRIO GERAL (FORMA G)
	// ==========================================
	writeLine("I001", "0")
	writeLine("I010", "G", "1.00")
	writeLine("I030", "TERMO DE ABERTURA", "1", "LIVRO DIARIO GERAL", fmt.Sprintf("%d", len(entries)), strings.ToUpper(company.LegalName), "", cnpjClean, dtIni, "", company.City)

	// I050: Plano de Contas
	for _, acc := range accounts {
		codNat := getSPEDAccountNature(acc.Code)
		indCta := "A"
		if acc.IsSynthetic {
			indCta = "S"
		}
		nivel := fmt.Sprintf("%d", strings.Count(acc.Code, ".")+1)

		// Conta superior imediata
		codSup := ""
		if lastDot := strings.LastIndex(acc.Code, "."); lastDot != -1 {
			codSup = acc.Code[:lastDot]
		}

		dtAlt := "01012026"
		writeLine("I050", dtAlt, codNat, indCta, nivel, acc.Code, codSup, strings.ToUpper(acc.Name))
	}

	writeLine("I150", dtIni, dtFin)

	// I200 / I250: Lançamentos e Partidas Contábeis
	for _, entry := range entries {
		numLanc := fmt.Sprintf("%d", entry.EntryNumber)
		dtLanc := entry.Date.Format("02012006")
		vlLanc := formatSPEDDecimal(entry.TotalDebit)
		tipoLanc := "N" // Normal

		writeLine("I200", numLanc, dtLanc, vlLanc, tipoLanc)

		for _, line := range entry.Lines {
			indDC := "D"
			if line.Type == domain.JournalLineCredit {
				indDC = "C"
			}
			vlPartida := formatSPEDDecimal(line.Amount)
			hist := line.Description
			if hist == "" {
				hist = entry.Description
			}

			writeLine("I250", line.AccountCode, "", vlPartida, indDC, numLanc, strings.ToUpper(hist))
		}
	}

	// I990: Encerramento do Bloco I
	blocoICount := 1 // incluirá o próprio I990
	for reg, v := range counter.Counts {
		if strings.HasPrefix(reg, "I") {
			blocoICount += v
		}
	}
	writeLine("I990", fmt.Sprintf("%d", blocoICount))

	// ==========================================
	// BLOCO J: DEMONSTRAÇÕES E SIGNATÁRIOS
	// ==========================================
	writeLine("J001", "0")
	writeLine("J005", dtIni, dtFin, "1", "BALANCO PATRIMONIAL E RESULTADO DO EXERCICIO")
	writeLine("J900", "TERMO DE ENCERRAMENTO", "1", "LIVRO DIARIO GERAL", strings.ToUpper(company.LegalName), cnpjClean, dtFin)
	// J930: Signatário 1: Contador
	writeLine("J930", strings.ToUpper(cfg.AuditorName), auditorCpfClean, "CONTABILISTA", "900", cfg.AuditorCRC, "", "", "", "", "", "")
	// J930: Signatário 2: Representante Legal
	writeLine("J930", strings.ToUpper(cfg.SignatoryName), signatoryCpfClean, "ADMINISTRADOR", cfg.SignatoryQual, "", "", "", "", "", "", "")

	// J990: Encerramento do Bloco J
	blocoJCount := 1
	for reg, v := range counter.Counts {
		if strings.HasPrefix(reg, "J") {
			blocoJCount += v
		}
	}
	writeLine("J990", fmt.Sprintf("%d", blocoJCount))

	// ==========================================
	// BLOCO 9: TOTALIZADORES E ENCERRAMENTO
	// ==========================================
	writeLine("9001", "0")

	// Tipos de registros já presentes no arquivo até o 9001
	regsInFile := make([]string, 0, len(counter.Order)+3)
	for _, reg := range counter.Order {
		regsInFile = append(regsInFile, reg)
	}
	// Adiciona os registros finais do Bloco 9
	for _, r := range []string{"9900", "9990", "9999"} {
		regsInFile = append(regsInFile, r)
	}

	totalRegs9900 := len(regsInFile)

	// Registros 9900
	for _, reg := range regsInFile {
		qtd := 0
		switch reg {
		case "9900":
			qtd = totalRegs9900
		case "9990", "9999":
			qtd = 1
		default:
			qtd = counter.Counts[reg]
		}
		writeLine("9900", reg, fmt.Sprintf("%d", qtd))
	}

	// 9990: Total de linhas do Bloco 9 (9001 + total de 9900 + próprio 9990)
	bloco9Linhas := 1 + totalRegs9900 + 1
	writeLine("9990", fmt.Sprintf("%d", bloco9Linhas))

	// 9999: Total geral de linhas do arquivo inteiro (incluindo o próprio 9999)
	totalLinhasAteAgora := strings.Count(sb.String(), "\r\n") + 1
	writeLine("9999", fmt.Sprintf("%d", totalLinhasAteAgora))

	return sb.String()
}

// formatSPEDDecimal converte valor decimal para o formato exigido pelo SPED (ex: 1500,50 sem separador de milhar)
func formatSPEDDecimal(d decimal.Decimal) string {
	fixed := d.StringFixed(2)
	return strings.ReplaceAll(fixed, ".", ",")
}

// getSPEDAccountNature retorna o código da natureza da conta para o registro I050 do SPED
func getSPEDAccountNature(code string) string {
	switch {
	case strings.HasPrefix(code, "1"):
		return "01" // Ativo
	case strings.HasPrefix(code, "2.3") || strings.HasPrefix(code, "2.4"):
		return "03" // Patrimônio Líquido
	case strings.HasPrefix(code, "2"):
		return "02" // Passivo
	case strings.HasPrefix(code, "3") || strings.HasPrefix(code, "4") || strings.HasPrefix(code, "5"):
		return "04" // Contas de Resultado (Receitas, Custos e Despesas)
	default:
		return "09" // Outras
	}
}
