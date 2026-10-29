package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// ExportService gera arquivos formatados para contabilidade (CSV e HTML para impressão oficial em PDF)
type ExportService struct {
	companyRepo *postgres.CompanyRepository
	reportRepo  *postgres.ReportRepository
}

// NewExportService instancia o serviço de exportação
func NewExportService(companyRepo *postgres.CompanyRepository, reportRepo *postgres.ReportRepository) *ExportService {
	return &ExportService{companyRepo: companyRepo, reportRepo: reportRepo}
}

// ExportTrialBalanceCSV gera o Balancete de Verificação em formato CSV
func (s *ExportService) ExportTrialBalanceCSV(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) ([]byte, string, error) {
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, "", err
	}

	report, err := s.reportRepo.GetTrialBalanceReport(ctx, companyID, startDate, endDate)
	if err != nil {
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	// Adiciona BOM para abertura perfeita com acentuação no Microsoft Excel
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(buf)
	w.Comma = ';' // Padrão Excel em português do Brasil

	_ = w.Write([]string{"NORTABIL - BALANCETE DE VERIFICAÇÃO"})
	_ = w.Write([]string{fmt.Sprintf("Empresa: %s | CNPJ: %s", company.LegalName, company.CNPJ)})
	_ = w.Write([]string{fmt.Sprintf("Período: %s a %s", startDate.Format("02/01/2006"), endDate.Format("02/01/2006"))})
	_ = w.Write([]string{""})

	// Cabeçalho das colunas
	_ = w.Write([]string{
		"Código",
		"Nome da Conta",
		"Classificação",
		"Saldo Inicial",
		"Nat. Inicial",
		"Débito do Período",
		"Crédito do Período",
		"Saldo Final",
		"Nat. Final",
	})

	for _, item := range report.Items {
		isSyn := "Analítica"
		if item.IsSynthetic {
			isSyn = "Sintética"
		}

		_ = w.Write([]string{
			item.AccountCode,
			item.AccountName,
			isSyn,
			strings.ReplaceAll(item.InitialBalance.StringFixed(2), ".", ","),
			string(item.InitialNature),
			strings.ReplaceAll(item.PeriodDebit.StringFixed(2), ".", ","),
			strings.ReplaceAll(item.PeriodCredit.StringFixed(2), ".", ","),
			strings.ReplaceAll(item.FinalBalance.StringFixed(2), ".", ","),
			string(item.FinalNature),
		})
	}

	// Totais gerais
	_ = w.Write([]string{""})
	_ = w.Write([]string{
		"TOTAIS",
		"",
		"",
		"",
		"",
		strings.ReplaceAll(report.TotalPeriodDebit.StringFixed(2), ".", ","),
		strings.ReplaceAll(report.TotalPeriodCredit.StringFixed(2), ".", ","),
		fmt.Sprintf("D: %s | C: %s",
			strings.ReplaceAll(report.TotalDebtorBalance.StringFixed(2), ".", ","),
			strings.ReplaceAll(report.TotalCreditorBalance.StringFixed(2), ".", ",")),
		"",
	})

	w.Flush()
	filename := fmt.Sprintf("Balancete_%s_%s.csv", company.CNPJ, startDate.Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}

// ExportDRECSV gera a Demonstração do Resultado do Exercício em CSV
func (s *ExportService) ExportDRECSV(ctx context.Context, companyID uuid.UUID, startDate, endDate time.Time) ([]byte, string, error) {
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, "", err
	}

	dre, err := s.reportRepo.GetDREReport(ctx, companyID, startDate, endDate)
	if err != nil {
		return nil, "", err
	}

	buf := new(bytes.Buffer)
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(buf)
	w.Comma = ';'

	_ = w.Write([]string{"NORTABIL - DEMONSTRAÇÃO DO RESULTADO DO EXERCÍCIO (DRE)"})
	_ = w.Write([]string{fmt.Sprintf("Empresa: %s | CNPJ: %s", company.LegalName, company.CNPJ)})
	_ = w.Write([]string{fmt.Sprintf("Período: %s a %s", startDate.Format("02/01/2006"), endDate.Format("02/01/2006"))})
	_ = w.Write([]string{""})

	_ = w.Write([]string{"Ordem", "Descrição da Conta / Subtotal", "Valor (R$)"})

	for _, l := range dre.Lines {
		_ = w.Write([]string{
			fmt.Sprintf("%d", l.Order),
			l.Description,
			strings.ReplaceAll(l.Amount.StringFixed(2), ".", ","),
		})
	}

	w.Flush()
	filename := fmt.Sprintf("DRE_%s_%s.csv", company.CNPJ, startDate.Format("2006-01-02"))
	return buf.Bytes(), filename, nil
}

// ExportPrintableHTML gera um documento HTML estruturado com CSS para impressão ou salvamento em PDF
func (s *ExportService) ExportPrintableHTML(ctx context.Context, companyID uuid.UUID, reportType string, startDate, endDate time.Time) (string, error) {
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return "", err
	}

	dtIni := startDate.Format("02/01/2006")
	dtFin := endDate.Format("02/01/2006")
	emitidoEm := time.Now().Format("02/01/2006 15:04:05")

	var sb strings.Builder
	fmt.Fprintf(&sb, `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>Nortabil - Relatório Contábil Oficial</title>
  <style>
    @page { size: A4; margin: 15mm; }
    body { font-family: 'Helvetica Neue', Arial, sans-serif; font-size: 11px; color: #222; margin: 0; padding: 20px; }
    .header { border-bottom: 2px solid #1e3a8a; padding-bottom: 8px; margin-bottom: 15px; }
    .header h1 { margin: 0; font-size: 16px; color: #1e3a8a; }
    .header p { margin: 2px 0; color: #555; }
    .title-box { background: #f1f5f9; padding: 8px 12px; font-weight: bold; font-size: 13px; text-transform: uppercase; margin-bottom: 12px; border-left: 4px solid #1e3a8a; }
    table { width: 100%%; border-collapse: collapse; margin-bottom: 20px; font-size: 10px; }
    th { background: #f8fafc; border: 1px solid #cbd5e1; padding: 6px; text-align: left; font-weight: bold; }
    td { border: 1px solid #e2e8f0; padding: 5px 6px; }
    .text-right { text-align: right; }
    .font-bold { font-weight: bold; }
    .bg-total { background: #f8fafc; font-weight: bold; }
    .footer { margin-top: 40px; border-top: 1px solid #cbd5e1; padding-top: 20px; }
    .signatures { display: flex; justify-content: space-around; margin-top: 50px; }
    .sig-line { width: 40%%; text-align: center; border-top: 1px solid #333; padding-top: 5px; font-size: 10px; }
    @media print {
      .no-print { display: none; }
      body { padding: 0; }
    }
  </style>
</head>
<body>
  <div class="no-print" style="margin-bottom: 15px;">
    <button onclick="window.print()" style="background: #1e3a8a; color: white; border: none; padding: 8px 16px; border-radius: 4px; font-weight: bold; cursor: pointer;">
      🖨️ Imprimir / Salvar em PDF
    </button>
  </div>
  <div class="header">
    <h1>%s</h1>
    <p>CNPJ: %s | Município: %s - %s</p>
    <p>Regime Tributário: %s | Escrituração Contábil Oficial</p>
  </div>
`, company.LegalName, company.CNPJ, company.City, company.State, string(company.TaxRegime))

	switch strings.ToLower(reportType) {
	case "dre":
		dre, err := s.reportRepo.GetDREReport(ctx, companyID, startDate, endDate)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, `
  <div class="title-box">DEMONSTRAÇÃO DO RESULTADO DO EXERCÍCIO (DRE) - Período: %s a %s</div>
  <table>
    <thead>
      <tr>
        <th style="width: 10%%;">Item</th>
        <th style="width: 65%%;">Descrição das Contas de Resultado</th>
        <th style="width: 25%%; text-align: right;">Valor (R$)</th>
      </tr>
    </thead>
    <tbody>
`, dtIni, dtFin)
		for _, l := range dre.Lines {
			rowClass := ""
			if l.IsTotal {
				rowClass = " class=\"bg-total\""
			}
			valStr := strings.ReplaceAll(l.Amount.StringFixed(2), ".", ",")
			fmt.Fprintf(&sb, "<tr%s><td>%d</td><td>%s</td><td class=\"text-right\">R$ %s</td></tr>\n",
				rowClass, l.Order, l.Description, valStr)
		}
		sb.WriteString(`</tbody></table>`)

	case "balance-sheet", "balanco":
		bs, err := s.reportRepo.GetBalanceSheetReport(ctx, companyID, endDate)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, `
  <div class="title-box">BALANÇO PATRIMONIAL ESTRUTURADO - Posição em %s</div>
  <div style="display: flex; gap: 20px;">
    <div style="flex: 1;">
      <h3 style="color: #1e3a8a; margin-top: 0;">ATIVO</h3>
      <table>
        <thead><tr><th>Conta</th><th class="text-right">Saldo</th></tr></thead>
        <tbody>
`, dtFin)
		for _, l := range bs.AssetLines {
			rowClass := ""
			if l.IsSynthetic {
				rowClass = " class=\"font-bold\""
			}
			valStr := strings.ReplaceAll(l.Amount.StringFixed(2), ".", ",")
			fmt.Fprintf(&sb, "<tr%s><td>%s %s</td><td class=\"text-right\">R$ %s</td></tr>\n",
				rowClass, l.AccountCode, l.AccountName, valStr)
		}
		fmt.Fprintf(&sb, "<tr class=\"bg-total\"><td>TOTAL DO ATIVO</td><td class=\"text-right\">R$ %s</td></tr>",
			strings.ReplaceAll(bs.TotalAsset.StringFixed(2), ".", ","))
		sb.WriteString(`</tbody></table></div>
    <div style="flex: 1;">
      <h3 style="color: #1e3a8a; margin-top: 0;">PASSIVO E PATRIMÔNIO LÍQUIDO</h3>
      <table>
        <thead><tr><th>Conta</th><th class="text-right">Saldo</th></tr></thead>
        <tbody>
`)
		for _, l := range bs.LiabilityLines {
			rowClass := ""
			if l.IsSynthetic {
				rowClass = " class=\"font-bold\""
			}
			valStr := strings.ReplaceAll(l.Amount.StringFixed(2), ".", ",")
			fmt.Fprintf(&sb, "<tr%s><td>%s %s</td><td class=\"text-right\">R$ %s</td></tr>\n",
				rowClass, l.AccountCode, l.AccountName, valStr)
		}
		fmt.Fprintf(&sb, "<tr class=\"bg-total\"><td>TOTAL DO PASSIVO + PL</td><td class=\"text-right\">R$ %s</td></tr>",
			strings.ReplaceAll(bs.TotalLiabilityAndEquity.StringFixed(2), ".", ","))
		sb.WriteString(`</tbody></table></div></div>`)

	case "ledger", "razao":
		ledger, err := s.reportRepo.GetLedgerReport(ctx, companyID, startDate, endDate, nil)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, `
  <div class="title-box">LIVRO RAZÃO ANALÍTICO (NBC ITG 2000) - Período: %s a %s</div>
`, dtIni, dtFin)
		for _, acc := range ledger.Accounts {
			if len(acc.Movements) == 0 && acc.InitialBalance.IsZero() && acc.FinalBalance.IsZero() {
				continue
			}
			initStr := strings.ReplaceAll(acc.InitialBalance.StringFixed(2), ".", ",")
			finStr := strings.ReplaceAll(acc.FinalBalance.StringFixed(2), ".", ",")
			totDeb := strings.ReplaceAll(acc.TotalDebit.StringFixed(2), ".", ",")
			totCred := strings.ReplaceAll(acc.TotalCredit.StringFixed(2), ".", ",")
			fmt.Fprintf(&sb, `
  <div style="margin-bottom: 18px; page-break-inside: avoid;">
    <div style="background: #e2e8f0; padding: 6px 10px; font-weight: bold; display: flex; justify-content: space-between; border: 1px solid #cbd5e1; border-bottom: none;">
      <span>Conta: %s - %s (%s)</span>
      <span>Saldo Inicial: R$ %s (%s) | Saldo Final: R$ %s (%s)</span>
    </div>
    <table>
      <thead>
        <tr>
          <th style="width: 12%%;">Data</th>
          <th style="width: 10%%;">Lanç. #</th>
          <th style="width: 38%%;">Histórico da Partida</th>
          <th style="width: 13%%;" class="text-right">Débito (R$)</th>
          <th style="width: 13%%;" class="text-right">Crédito (R$)</th>
          <th style="width: 14%%;" class="text-right">Saldo Acumulado</th>
        </tr>
      </thead>
      <tbody>
`, acc.AccountCode, acc.AccountName, string(acc.AccountType), initStr, string(acc.InitialNature), finStr, string(acc.FinalNature))
			for _, m := range acc.Movements {
				dtMov := m.Date.Format("02/01/2006")
				debStr := "-"
				if m.Debit.IsPositive() {
					debStr = strings.ReplaceAll(m.Debit.StringFixed(2), ".", ",")
				}
				credStr := "-"
				if m.Credit.IsPositive() {
					credStr = strings.ReplaceAll(m.Credit.StringFixed(2), ".", ",")
				}
				runStr := strings.ReplaceAll(m.RunningBalance.StringFixed(2), ".", ",")
				natSuffix := "D"
				if m.BalanceNature == "CREDIT" {
					natSuffix = "C"
				}
				fmt.Fprintf(&sb, "<tr><td>%s</td><td>#%d</td><td>%s</td><td class=\"text-right\">%s</td><td class=\"text-right\">%s</td><td class=\"text-right font-bold\">%s %s</td></tr>\n",
					dtMov, m.EntryNumber, m.Description, debStr, credStr, runStr, natSuffix)
			}
			fmt.Fprintf(&sb, "<tr class=\"bg-total\"><td colspan=\"3\">TOTAIS DA CONTA NO PERÍODO</td><td class=\"text-right\">R$ %s</td><td class=\"text-right\">R$ %s</td><td class=\"text-right\">R$ %s (%s)</td></tr></tbody></table></div>\n",
				totDeb, totCred, finStr, string(acc.FinalNature))
		}

	default: // Balancete
		report, err := s.reportRepo.GetTrialBalanceReport(ctx, companyID, startDate, endDate)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&sb, `
  <div class="title-box">BALANCETE DE VERIFICAÇÃO - Período: %s a %s</div>
  <table>
    <thead>
      <tr>
        <th>Código</th>
        <th>Nome da Conta</th>
        <th class="text-right">Saldo Inicial</th>
        <th class="text-right">Débitos</th>
        <th class="text-right">Créditos</th>
        <th class="text-right">Saldo Final</th>
      </tr>
    </thead>
    <tbody>
`, dtIni, dtFin)
		for _, it := range report.Items {
			rowClass := ""
			if it.IsSynthetic {
				rowClass = " class=\"font-bold\""
			}
			initStr := strings.ReplaceAll(it.InitialBalance.StringFixed(2), ".", ",")
			debStr := strings.ReplaceAll(it.PeriodDebit.StringFixed(2), ".", ",")
			credStr := strings.ReplaceAll(it.PeriodCredit.StringFixed(2), ".", ",")
			finStr := strings.ReplaceAll(it.FinalBalance.StringFixed(2), ".", ",")
			fmt.Fprintf(&sb, "<tr%s><td>%s</td><td>%s</td><td class=\"text-right\">%s %s</td><td class=\"text-right\">%s</td><td class=\"text-right\">%s</td><td class=\"text-right\">%s %s</td></tr>\n",
				rowClass, it.AccountCode, it.AccountName, initStr, it.InitialNature, debStr, credStr, finStr, it.FinalNature)
		}
		totDeb := strings.ReplaceAll(report.TotalPeriodDebit.StringFixed(2), ".", ",")
		totCred := strings.ReplaceAll(report.TotalPeriodCredit.StringFixed(2), ".", ",")
		fmt.Fprintf(&sb, "<tr class=\"bg-total\"><td colspan=\"3\">TOTAIS DO PERÍODO</td><td class=\"text-right\">R$ %s</td><td class=\"text-right\">R$ %s</td><td></td></tr>",
			totDeb, totCred)
		sb.WriteString(`</tbody></table>`)
	}

	// Termos e Assinaturas
	fmt.Fprintf(&sb, `
  <div class="footer">
    <p style="text-align: right; color: #777;">Documento gerado eletronicamente pelo Sistema NORTABIL em %s</p>
    <div class="signatures">
      <div class="sig-line">
        <strong>%s</strong><br>
        Representante Legal - CNPJ %s
      </div>
      <div class="sig-line">
        <strong>CONTADOR RESPONSÁVEL TÉCNICO</strong><br>
        CRC/MG - Região Norte de Minas Gerais
      </div>
    </div>
  </div>
</body>
</html>`, emitidoEm, company.LegalName, company.CNPJ)

	return sb.String(), nil
}
