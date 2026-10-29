package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Script de demonstração contábil com casos reais de escritórios contábeis
func main() {
	baseURL := "http://localhost:8080/api/v1"
	log.Println("=================================================================")
	log.Println("   NORTABIL - Script de Demonstração e Carga de Fatos Contábeis  ")
	log.Println("=================================================================")

	// 1. Testa conectividade
	resp, err := http.Get("http://localhost:8080/healthz")
	if err != nil {
		log.Fatalf("API offline em %s: %v. Inicie o servidor antes de rodar a demonstração.", baseURL, err)
	}
	defer resp.Body.Close()
	log.Println("[1/4] API Nortabil ativa e saudável.")

	// 2. Busca contas para mapear IDs das contas analíticas
	accResp, err := http.Get(baseURL + "/accounts")
	if err != nil {
		log.Fatalf("Erro ao consultar contas: %v", err)
	}
	defer accResp.Body.Close()

	var accListWrapper struct {
		Success bool `json:"success"`
		Data    []struct {
			ID   string `json:"id"`
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"data"`
	}
	body, _ := io.ReadAll(accResp.Body)
	_ = json.Unmarshal(body, &accListWrapper)

	codeToID := make(map[string]string)
	for _, a := range accListWrapper.Data {
		codeToID[a.Code] = a.ID
	}
	log.Printf("[2/4] Identificadas %d contas contábeis no plano.", len(codeToID))

	// IDs das contas analíticas padrão
	caixaID := codeToID["1.1.01.001"]
	bancoID := codeToID["1.1.01.002"]
	clientesID := codeToID["1.1.02.001"]
	estoqueID := codeToID["1.1.03.001"]
	fornecedoresID := codeToID["2.1.01.001"]
	salariosPagarID := codeToID["2.1.03.001"]
	capitalSocialID := codeToID["3.1.01"]
	receitaServicosID := codeToID["4.1.01.001"]
	receitaVendasID := codeToID["4.1.01.002"]
	cmvID := codeToID["5.1.01.001"]
	despSalariosID := codeToID["5.2.01.001"]
	despAluguelID := codeToID["5.2.01.002"]

	today := time.Now().Format("2006-01-02T15:04:05Z")

	// Lançamentos contábeis reais
	demonstracoes := []map[string]interface{}{
		{
			"date":        today,
			"description": "Constituição da empresa com integralização de Capital Social em Banco e Caixa",
			"lines": []map[string]interface{}{
				{"account_id": bancoID, "type": "DEBIT", "amount": "80000.00", "description": "Depósito bancário inicial"},
				{"account_id": caixaID, "type": "DEBIT", "amount": "20000.00", "description": "Aporte em moeda corrente"},
				{"account_id": capitalSocialID, "type": "CREDIT", "amount": "100000.00", "description": "Subscrição e integralização de quotas"},
			},
		},
		{
			"date":        today,
			"description": "Aquisição de mercadorias para revenda a prazo conforme NF-e 1024",
			"lines": []map[string]interface{}{
				{"account_id": estoqueID, "type": "DEBIT", "amount": "35000.00", "description": "Entrada de estoque de mercadorias"},
				{"account_id": fornecedoresID, "type": "CREDIT", "amount": "35000.00", "description": "Fatura a pagar em 30 dias"},
			},
		},
		{
			"date":        today,
			"description": "Venda de mercadorias com 50% à vista e 50% a prazo conforme NF-e 501",
			"lines": []map[string]interface{}{
				{"account_id": bancoID, "type": "DEBIT", "amount": "25000.00", "description": "Recebimento PIX"},
				{"account_id": clientesID, "type": "DEBIT", "amount": "25000.00", "description": "Duplicatas a receber em 30/60 dias"},
				{"account_id": receitaVendasID, "type": "CREDIT", "amount": "50000.00", "description": "Faturamento bruto de vendas"},
			},
		},
		{
			"date":        today,
			"description": "Reconhecimento do Custo das Mercadorias Vendidas (CMV da NF-e 501)",
			"lines": []map[string]interface{}{
				{"account_id": cmvID, "type": "DEBIT", "amount": "20000.00", "description": "Apropriação do custo dos produtos vendidos"},
				{"account_id": estoqueID, "type": "CREDIT", "amount": "20000.00", "description": "Baixa física e contábil do estoque"},
			},
		},
		{
			"date":        today,
			"description": "Apropriação da folha de pagamento do mês de competência",
			"lines": []map[string]interface{}{
				{"account_id": despSalariosID, "type": "DEBIT", "amount": "12000.00", "description": "Despesas com salários da equipe"},
				{"account_id": salariosPagarID, "type": "CREDIT", "amount": "12000.00", "description": "Obrigação salarial a liquidar no 5º dia útil"},
			},
		},
		{
			"date":        today,
			"description": "Prestação de serviços contábeis e de consultoria recebidos via Banco",
			"lines": []map[string]interface{}{
				{"account_id": bancoID, "type": "DEBIT", "amount": "15000.00", "description": "Depósito bancário recebido"},
				{"account_id": receitaServicosID, "type": "CREDIT", "amount": "15000.00", "description": "Serviços prestados a terceiros"},
			},
		},
		{
			"date":        today,
			"description": "Pagamento de aluguel do escritório comercial via transferência bancária",
			"lines": []map[string]interface{}{
				{"account_id": despAluguelID, "type": "DEBIT", "amount": "3000.00", "description": "Aluguel referente ao mês corrente"},
				{"account_id": bancoID, "type": "CREDIT", "amount": "3000.00", "description": "Saída via TED bancária"},
			},
		},
	}

	log.Printf("[3/4] Enviando lote com %d lançamentos em partidas dobradas...", len(demonstracoes))
	payloadBytes, _ := json.Marshal(demonstracoes)
	batchResp, err := http.Post(baseURL+"/journal-entries/batch", "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		log.Fatalf("Erro ao enviar lote: %v", err)
	}
	defer batchResp.Body.Close()

	var batchReport struct {
		Success bool `json:"success"`
		Data    struct {
			TotalProcessed int   `json:"total_processed"`
			TotalSuccess   int   `json:"total_success"`
			TotalFailed    int   `json:"total_failed"`
			DurationMs     int64 `json:"duration_ms"`
		} `json:"data"`
	}
	bReportBytes, _ := io.ReadAll(batchResp.Body)
	_ = json.Unmarshal(bReportBytes, &batchReport)

	log.Printf("[4/4] Lote processado em %d ms! Total: %d, Sucesso: %d, Falhas: %d",
		batchReport.Data.DurationMs,
		batchReport.Data.TotalProcessed,
		batchReport.Data.TotalSuccess,
		batchReport.Data.TotalFailed,
	)

	fmt.Println("\n>> Demonstração concluída com sucesso! Acesse http://localhost:8080/swagger para inspecionar os relatórios (Razão, Diário e Balancete).")
}
