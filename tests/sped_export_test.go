package tests

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// TestSPEDECDGeneration valida a conformidade dos Blocos 0, I, J e 9 do SPED ECD
func TestSPEDECDGeneration(t *testing.T) {
	companyID := uuid.New()
	company := &domain.Company{
		ID:        companyID,
		CNPJ:      "03.743.085/0001-20",
		LegalName: "AGROPECUARIA VALE DO SAO FRANCISCO LTDA",
		City:      "Januária",
		State:     "MG",
		Active:    true,
	}

	accBancoID := uuid.New()
	accClientesID := uuid.New()
	accounts := []domain.Account{
		{
			ID:          uuid.New(),
			CompanyID:   companyID,
			Code:        "1",
			Name:        "ATIVO",
			Type:        domain.AccountTypeAsset,
			IsSynthetic: true,
		},
		{
			ID:          uuid.New(),
			CompanyID:   companyID,
			Code:        "1.1",
			Name:        "ATIVO CIRCULANTE",
			Type:        domain.AccountTypeAsset,
			IsSynthetic: true,
		},
		{
			ID:          accBancoID,
			CompanyID:   companyID,
			Code:        "1.1.01.002",
			Name:        "Banco do Brasil S/A",
			Type:        domain.AccountTypeAsset,
			IsSynthetic: false,
		},
		{
			ID:          accClientesID,
			CompanyID:   companyID,
			Code:        "1.1.02.001",
			Name:        "Clientes a Receber - Mercado Local Januária",
			Type:        domain.AccountTypeAsset,
			IsSynthetic: false,
		},
	}

	entryDate := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	entryAmount := decimal.NewFromFloat(15000.50)

	entries := []domain.JournalEntry{
		{
			ID:          uuid.New(),
			CompanyID:   companyID,
			EntryNumber: 1,
			Date:        entryDate,
			Description: "Recebimento de vendas safra de mangas via Banco do Brasil",
			Status:      domain.JournalStatusPosted,
			TotalDebit:  entryAmount,
			TotalCredit: entryAmount,
			Lines: []domain.JournalLine{
				{
					ID:          uuid.New(),
					AccountID:   accBancoID,
					AccountCode: "1.1.01.002",
					AccountName: "Banco do Brasil S/A",
					Type:        domain.JournalLineDebit,
					Amount:      entryAmount,
					Description: "Recebimento via PIX",
				},
				{
					ID:          uuid.New(),
					AccountID:   accClientesID,
					AccountCode: "1.1.02.001",
					AccountName: "Clientes a Receber - Mercado Local Januária",
					Type:        domain.JournalLineCredit,
					Amount:      entryAmount,
					Description: "Baixa de duplicata cliente",
				},
			},
		},
	}

	cfg := domain.SPEDExportConfig{
		CompanyID:     companyID,
		StartDate:     time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		EndDate:       time.Date(2026, time.December, 31, 23, 59, 59, 0, time.UTC),
		BookType:      "G",
		AuditorCRC:    "MG-085421/O",
		AuditorName:   "Sebastião Ferreira da Silva",
		AuditorCPF:    "12345678909",
		SignatoryName: "Manoel de Oliveira",
		SignatoryCPF:  "98765432100",
		SignatoryQual: "207",
	}

	content := service.BuildSPEDFile(company, accounts, entries, cfg)

	if len(content) == 0 {
		t.Fatalf("Arquivo SPED gerado vazio")
	}

	lines := strings.Split(strings.TrimRight(content, "\r\n"), "\r\n")

	// 1. Validação de estrutura delimitada por pipes
	for idx, line := range lines {
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
			t.Errorf("Linha %d não está delimitada por pipes: %s", idx+1, line)
		}
	}

	// 2. Validação do Registro 0000 (Abertura e Identificação)
	reg0000 := lines[0]
	if !strings.Contains(reg0000, "|0000|LECD|01012026|31122026|AGROPECUARIA VALE DO SAO FRANCISCO LTDA|03743085000120|MG||3135209|") {
		t.Errorf("Registro 0000 incorreto: %s", reg0000)
	}

	// 3. Validação do Registro I010 (Livro Diário Geral Forma G)
	hasI010 := false
	for _, l := range lines {
		if l == "|I010|G|1.00|" {
			hasI010 = true
			break
		}
	}
	if !hasI010 {
		t.Errorf("Registro I010 (Livro Diário Geral) não encontrado")
	}

	// 4. Validação do Plano de Contas (I050)
	hasI050Banco := false
	for _, l := range lines {
		if strings.Contains(l, "|I050|01012026|01|A|4|1.1.01.002|1.1.01|BANCO DO BRASIL S/A|") {
			hasI050Banco = true
			break
		}
	}
	if !hasI050Banco {
		t.Errorf("Registro I050 para conta analítica de banco não encontrado")
	}

	// 5. Validação dos Lançamentos Contábeis (I200 e I250 com partidas dobradas)
	hasI200 := false
	hasI250Debito := false
	hasI250Credito := false

	for _, l := range lines {
		if strings.Contains(l, "|I200|1|15102026|15000,50|N|") {
			hasI200 = true
		}
		if strings.Contains(l, "|I250|1.1.01.002||15000,50|D|1|RECEBIMENTO VIA PIX|") {
			hasI250Debito = true
		}
		if strings.Contains(l, "|I250|1.1.02.001||15000,50|C|1|BAIXA DE DUPLICATA CLIENTE|") {
			hasI250Credito = true
		}
	}

	if !hasI200 {
		t.Errorf("Registro I200 de cabeçalho do lançamento com R$ 15000,50 não encontrado")
	}
	if !hasI250Debito {
		t.Errorf("Partida a Débito I250 no Banco não encontrada")
	}
	if !hasI250Credito {
		t.Errorf("Partida a Crédito I250 em Clientes não encontrada")
	}

	// 6. Validação dos Signatários (J930)
	hasContador := false
	hasAdmin := false
	for _, l := range lines {
		if strings.Contains(l, "|J930|SEBASTIÃO FERREIRA DA SILVA|12345678909|CONTABILISTA|900|MG-085421/O|") {
			hasContador = true
		}
		if strings.Contains(l, "|J930|MANOEL DE OLIVEIRA|98765432100|ADMINISTRADOR|207|") {
			hasAdmin = true
		}
	}
	if !hasContador {
		t.Errorf("Signatário Contador J930 não encontrado no arquivo")
	}
	if !hasAdmin {
		t.Errorf("Signatário Administrador J930 não encontrado no arquivo")
	}

	// 7. Validação do Totalizador Geral 9999
	lastLine := lines[len(lines)-1]
	if !strings.HasPrefix(lastLine, "|9999|") {
		t.Fatalf("Última linha deve ser o registro 9999, obteve: %s", lastLine)
	}

	parts := strings.Split(lastLine, "|")
	if len(parts) < 4 {
		t.Fatalf("Registro 9999 mal formatado: %s", lastLine)
	}
	totalDeclarado, err := strconv.Atoi(parts[2])
	if err != nil {
		t.Fatalf("Valor do registro 9999 não é numérico: %s", parts[2])
	}

	if totalDeclarado != len(lines) {
		t.Errorf("Totalizador 9999 diverge do número real de linhas do arquivo! Declarado: %d, Real: %d", totalDeclarado, len(lines))
	}
}
