package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/config"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
	"github.com/matheus-sl2/nortabil/internal/service"
	"github.com/shopspring/decimal"
)

func main() {
	log.Println("=========================================================================")
	log.Println("   NORTABIL - Carga de Dados Realistas do Norte de Minas Gerais (TCC)   ")
	log.Println("   Cenários: Fruticultura Irrigada (Jaíba) e Atacado (Montes Claros)    ")
	log.Println("=========================================================================")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[ERRO] Falha ao carregar configurações: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := postgres.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("[ERRO] Falha ao conectar ao PostgreSQL: %v", err)
	}
	defer db.Close()

	companyRepo := postgres.NewCompanyRepository(db)
	accountRepo := postgres.NewAccountRepository(db)
	journalRepo := postgres.NewJournalRepository(db)
	periodRepo := postgres.NewPeriodRepository(db)

	companyService := service.NewCompanyService(companyRepo)
	journalService := service.NewJournalService(journalRepo, accountRepo, periodRepo)
	periodService := service.NewPeriodService(periodRepo, journalRepo, accountRepo)

	// =========================================================================
	// 1. CARGA: Fruticultura Vale do Jaíba Ltda (Jaíba - MG)
	// =========================================================================
	cnpjJaiba := "14.882.352/0001-22"
	log.Printf("[INFO] Processando empresa 1: Fruticultura Vale do Jaíba Ltda (%s)...", cnpjJaiba)
	compJaiba, err := seedCompany(ctx, companyService, companyRepo, service.CreateCompanyInput{
		CNPJ:                 cnpjJaiba,
		LegalName:            "Fruticultura Vale do Jaíba Ltda",
		TradeName:            "Frutas do Jaíba",
		TaxRegime:            domain.TaxRegimeLucroPresumido,
		City:                 "Jaíba",
		State:                "MG",
		CloneDefaultAccounts: true,
	})
	if err != nil {
		log.Fatalf("[ERRO] Falha ao cadastrar empresa de Jaíba: %v", err)
	}

	log.Printf("[SUCESSO] Empresa Jaíba carregada (ID: %s)", compJaiba.ID)
	seedJaibaTransactions(ctx, journalService, accountRepo, periodService, compJaiba.ID)

	// =========================================================================
	// 2. CARGA: Distribuidora Comercial Montes Claros Ltda (Montes Claros - MG)
	// =========================================================================
	cnpjMoc := "28.451.983/0001-00"
	log.Printf("[INFO] Processando empresa 2: Distribuidora Comercial Montes Claros Ltda (%s)...", cnpjMoc)
	compMoc, err := seedCompany(ctx, companyService, companyRepo, service.CreateCompanyInput{
		CNPJ:                 cnpjMoc,
		LegalName:            "Distribuidora Comercial Montes Claros Ltda",
		TradeName:            "Norte Atacado Moc",
		TaxRegime:            domain.TaxRegimeLucroReal,
		City:                 "Montes Claros",
		State:                "MG",
		CloneDefaultAccounts: true,
	})
	if err != nil {
		log.Fatalf("[ERRO] Falha ao cadastrar empresa de Montes Claros: %v", err)
	}

	log.Printf("[SUCESSO] Empresa Montes Claros carregada (ID: %s)", compMoc.ID)
	seedMocTransactions(ctx, journalService, accountRepo, periodService, compMoc.ID)

	log.Println("=========================================================================")
	log.Println(" [CONCLUÍDO] Massa de dados regional semeada com sucesso no PostgreSQL!  ")
	log.Println(" Empresas, Contas, Lançamentos em Partidas Dobradas e Períodos Prontos. ")
	log.Println("=========================================================================")
}

func seedCompany(ctx context.Context, svc *service.CompanyService, repo *postgres.CompanyRepository, input service.CreateCompanyInput) (*domain.Company, error) {
	cleanCNPJ := domain.CleanCNPJ(input.CNPJ)
	existing, err := repo.GetByCNPJ(ctx, cleanCNPJ)
	if err == nil && existing != nil {
		return existing, nil
	}
	return svc.CreateCompany(ctx, input)
}

func getAccID(ctx context.Context, repo *postgres.AccountRepository, compID uuid.UUID, code string) uuid.UUID {
	acc, err := repo.GetByCode(ctx, compID, code)
	if err != nil {
		log.Fatalf("[ERRO CRÍTICO] Conta contábil referencial %s não encontrada para empresa %s: %v", code, compID, err)
	}
	return acc.ID
}

func seedJaibaTransactions(ctx context.Context, js *service.JournalService, ar *postgres.AccountRepository, ps *service.PeriodService, compID uuid.UUID) {
	// Mapeia contas analíticas necessárias
	banco := getAccID(ctx, ar, compID, "1.1.01.002")       // Bancos Conta Movimento
	clientes := getAccID(ctx, ar, compID, "1.1.02.001")    // Clientes Nacionais
	estoques := getAccID(ctx, ar, compID, "1.1.03.001")    // Mercadorias / Insumos
	maquinas := getAccID(ctx, ar, compID, "1.2.01.001")    // Máquinas e Equipamentos (Pivô central / tratores)
	fornecedores := getAccID(ctx, ar, compID, "2.1.01.001")// Fornecedores Nacionais
	salariosPagar := getAccID(ctx, ar, compID, "2.1.03.001")// Salários a Pagar
	capitalSocial := getAccID(ctx, ar, compID, "3.1.01")   // Capital Subscrito e Integralizado
	recVendas := getAccID(ctx, ar, compID, "4.1.01.002")   // Receita de Venda de Mercadorias (Frutas)
	cmv := getAccID(ctx, ar, compID, "5.1.01.001")         // Custo das Mercadorias Vendidas
	despSalarios := getAccID(ctx, ar, compID, "5.2.01.001")// Despesas com Salários (Colhedores de Manga/Banana)
	despEnergia := getAccID(ctx, ar, compID, "5.2.01.003") // Energia Elétrica (Bombas de Irrigação CEMIG)

	lancamentos := []struct {
		desc   string
		date   string
		debito uuid.UUID
		cred   uuid.UUID
		val    string
	}{
		{
			desc:   "Integralização do Capital Social Inicial via Banco do Brasil",
			date:   "2026-01-10",
			debito: banco,
			cred:   capitalSocial,
			val:    "500000.00",
		},
		{
			desc:   "Aquisição de Trator Massey Ferguson e Sistema de Irrigação Pivô Central a Prazo",
			date:   "2026-01-15",
			debito: maquinas,
			cred:   fornecedores,
			val:    "180000.00",
		},
		{
			desc:   "Aquisição de Adubos, Defensivos e Embalagens de Manga (Estoque) via Banco",
			date:   "2026-01-22",
			debito: estoques,
			cred:   banco,
			val:    "45000.00",
		},
		{
			desc:   "Faturamento de Safra de Manga Palmer e Banana Prata para Atacadista da CeasaMinas",
			date:   "2026-02-05",
			debito: clientes,
			cred:   recVendas,
			val:    "145000.00",
		},
		{
			desc:   "Baixa do Custo Agrícola da Colheita Comercializada (CMV)",
			date:   "2026-02-05",
			debito: cmv,
			cred:   estoques,
			val:    "38000.00",
		},
		{
			desc:   "Recebimento de Duplicatas de Venda de Frutas via Transferência Bancária PIX",
			date:   "2026-02-20",
			debito: banco,
			cred:   clientes,
			val:    "110000.00",
		},
		{
			desc:   "Apropriação da Folha de Pagamento dos Trabalhadores Rurais e Colhedores de Jaíba",
			date:   "2026-02-28",
			debito: despSalarios,
			cred:   salariosPagar,
			val:    "28500.00",
		},
		{
			desc:   "Pagamento da Conta de Energia Elétrica das Bombas de Sucção do Rio São Francisco (CEMIG)",
			date:   "2026-02-28",
			debito: despEnergia,
			cred:   banco,
			val:    "8450.00",
		},
	}

	for _, l := range lancamentos {
		t, _ := time.Parse("2006-01-02", l.date)
		val, _ := decimal.NewFromString(l.val)
		_, err := js.CreateEntry(ctx, compID, service.CreateJournalEntryInput{
			CompanyID:   &compID,
			Date:        t,
			Description: l.desc,
			Lines: []service.CreateJournalLineInput{
				{AccountID: l.debito, Type: domain.JournalLineDebit, Amount: val, Description: "Débito"},
				{AccountID: l.cred, Type: domain.JournalLineCredit, Amount: val, Description: "Crédito"},
			},
		})
		if err != nil {
			log.Printf("[AVISO] Lançamento já existente ou com erro: %v", err)
		}
	}
	log.Println("[INFO] Lançamentos agrícolas de Jaíba registrados com sucesso.")
}

func seedMocTransactions(ctx context.Context, js *service.JournalService, ar *postgres.AccountRepository, ps *service.PeriodService, compID uuid.UUID) {
	banco := getAccID(ctx, ar, compID, "1.1.01.002")       // Bancos Conta Movimento
	clientes := getAccID(ctx, ar, compID, "1.1.02.001")    // Clientes Nacionais
	estoques := getAccID(ctx, ar, compID, "1.1.03.001")    // Mercadorias para Revenda
	veiculos := getAccID(ctx, ar, compID, "1.2.01.002")    // Veículos (Caminhões de entrega)
	fornecedores := getAccID(ctx, ar, compID, "2.1.01.001")// Fornecedores Nacionais
	capitalSocial := getAccID(ctx, ar, compID, "3.1.01")   // Capital Social
	recVendas := getAccID(ctx, ar, compID, "4.1.01.002")   // Receita de Venda de Mercadorias
	cmv := getAccID(ctx, ar, compID, "5.1.01.001")         // CMV
	despAluguel := getAccID(ctx, ar, compID, "5.2.01.002") // Despesas com Aluguel de Galpão Logístico
	despInternet := getAccID(ctx, ar, compID, "5.2.01.004")// Despesas de Internet e ERP

	lancamentos := []struct {
		desc   string
		date   string
		debito uuid.UUID
		cred   uuid.UUID
		val    string
	}{
		{
			desc:   "Subscrição e Integralização de Capital Social em Conta Santander",
			date:   "2026-01-05",
			debito: banco,
			cred:   capitalSocial,
			val:    "350000.00",
		},
		{
			desc:   "Compra de 2 Caminhões Iveco para Logística de Distribuição no Norte de MG a Prazo",
			date:   "2026-01-12",
			debito: veiculos,
			cred:   fornecedores,
			val:    "220000.00",
		},
		{
			desc:   "Aquisição de Lote de Mercadorias Atacadistas (Secos e Molhados) a Prazo",
			date:   "2026-01-20",
			debito: estoques,
			cred:   fornecedores,
			val:    "95000.00",
		},
		{
			desc:   "Venda Atacadista Faturada para Supermercados de Januária, Bocaiuva e Janaúba",
			date:   "2026-02-10",
			debito: clientes,
			cred:   recVendas,
			val:    "185000.00",
		},
		{
			desc:   "Baixa de Estoque por Venda Mercantil Atacadista (CMV)",
			date:   "2026-02-10",
			debito: cmv,
			cred:   estoques,
			val:    "74000.00",
		},
		{
			desc:   "Recebimento Bancário de Clientes da Região Norte",
			date:   "2026-02-25",
			debito: banco,
			cred:   clientes,
			val:    "135000.00",
		},
		{
			desc:   "Pagamento do Aluguel de Galpão no Distrito Industrial de Montes Claros",
			date:   "2026-02-28",
			debito: despAluguel,
			cred:   banco,
			val:    "12000.00",
		},
		{
			desc:   "Pagamento de Link de Internet Fibra e Licenças de TI",
			date:   "2026-02-28",
			debito: despInternet,
			cred:   banco,
			val:    "1850.00",
		},
	}

	for _, l := range lancamentos {
		t, _ := time.Parse("2006-01-02", l.date)
		val, _ := decimal.NewFromString(l.val)
		_, err := js.CreateEntry(ctx, compID, service.CreateJournalEntryInput{
			CompanyID:   &compID,
			Date:        t,
			Description: l.desc,
			Lines: []service.CreateJournalLineInput{
				{AccountID: l.debito, Type: domain.JournalLineDebit, Amount: val, Description: "Débito"},
				{AccountID: l.cred, Type: domain.JournalLineCredit, Amount: val, Description: "Crédito"},
			},
		})
		if err != nil {
			log.Printf("[AVISO] Lançamento já existente ou com erro: %v", err)
		}
	}
	log.Println("[INFO] Lançamentos comerciais de Montes Claros registrados com sucesso.")
}
