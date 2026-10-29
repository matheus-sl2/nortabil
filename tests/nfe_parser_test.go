package tests

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/shopspring/decimal"
)

func TestParseNFeXMLAndAccountingEntry(t *testing.T) {
	rawXML := `<?xml version="1.0" encoding="UTF-8"?>
<nfeProc versao="4.00" xmlns="http://www.portalfiscal.inf.br/nfe">
  <NFe>
    <infNFe Id="NFe31260219131243000197550010001254801001254809" versao="4.00">
      <ide>
        <cUF>31</cUF>
        <natOp>Venda de Producao do Estabelecimento</natOp>
        <mod>55</mod>
        <serie>1</serie>
        <nNF>125480</nNF>
        <dhEmi>2026-02-15T14:15:00-03:00</dhEmi>
        <tpNF>1</tpNF>
      </ide>
      <emit>
        <CNPJ>19131243000197</CNPJ>
        <xNome>FRUTICULTURA VALE DO JAIBA LTDA</xNome>
      </emit>
      <dest>
        <CNPJ>60701190000104</CNPJ>
        <xNome>DISTRIBUIDORA COMERCIAL MONTES CLAROS LTDA</xNome>
      </dest>
      <det nItem="1">
        <prod>
          <cProd>FRU-101</cProd>
          <xProd>Banana Prata Selecionada Exportacao</xProd>
          <CFOP>5101</CFOP>
          <vProd>22500.00</vProd>
        </prod>
      </det>
      <total>
        <ICMSTot>
          <vProd>22500.00</vProd>
          <vICMS>2700.00</vICMS>
          <vDesc>0.00</vDesc>
          <vNF>22500.00</vNF>
        </ICMSTot>
      </total>
    </infNFe>
  </NFe>
  <protNFe versao="4.00">
    <infProt>
      <chNFe>31260219131243000197550010001254801001254809</chNFe>
    </infProt>
  </protNFe>
</nfeProc>`

	doc, err := domain.ParseNFeXML(rawXML)
	if err != nil {
		t.Fatalf("falha ao processar XML NF-e válido: %v", err)
	}

	if doc.AccessKey != "31260219131243000197550010001254801001254809" || doc.Model != "55" {
		t.Errorf("chave/modelo inesperados: %s / %s", doc.AccessKey, doc.Model)
	}
	if doc.Number != "125480" || doc.Series != "1" {
		t.Errorf("número/série inesperados: %s / %s", doc.Number, doc.Series)
	}
	if doc.OperationType != "1" || doc.OperationNature != "Venda de Producao do Estabelecimento" {
		t.Errorf("tipo/natureza inesperados: %s / %s", doc.OperationType, doc.OperationNature)
	}
	if doc.EmitterCNPJ != "19131243000197" || doc.EmitterName != "FRUTICULTURA VALE DO JAIBA LTDA" {
		t.Errorf("emitente inesperado: %s (%s)", doc.EmitterName, doc.EmitterCNPJ)
	}
	if doc.RecipientCNPJ != "60701190000104" || doc.RecipientName != "DISTRIBUIDORA COMERCIAL MONTES CLAROS LTDA" {
		t.Errorf("destinatário inesperado: %s (%s)", doc.RecipientName, doc.RecipientCNPJ)
	}
	if !doc.TotalInvoice.Equal(decimal.NewFromFloat(22500.00)) || !doc.TotalProducts.Equal(decimal.NewFromFloat(22500.00)) {
		t.Errorf("totais da nota inesperados: vNF=%s, vProd=%s", doc.TotalInvoice, doc.TotalProducts)
	}
	if !doc.TotalICMS.Equal(decimal.NewFromFloat(2700.00)) || !doc.TotalDiscount.IsZero() {
		t.Errorf("ICMS/Desconto inesperados: vICMS=%s, vDesc=%s", doc.TotalICMS, doc.TotalDiscount)
	}
	if !strings.Contains(doc.ItemsSummary, "Banana Prata") || doc.IssueDate.IsZero() {
		t.Errorf("resumo de itens ou data de emissão inválidos: %s / %v", doc.ItemsSummary, doc.IssueDate)
	}

	companyID := uuid.New()
	debitAcc := uuid.New()
	creditAcc := uuid.New()
	cmvAcc := uuid.New()
	invAcc := uuid.New()

	entry, err := doc.BuildAccountingEntry(companyID, "SAIDA_VENDA", debitAcc, creditAcc, "contador@nortabil.com.br")
	if err != nil {
		t.Fatalf("falha ao gerar lançamento contábil da NF-e: %v", err)
	}

	// Anexa a baixa automática de Estoque contra CMV (60% de R$ 22.500,00 = R$ 13.500,00)
	cmvAmount := decimal.RequireFromString("13500.00")
	if err := doc.AttachInventoryWriteOff(entry, cmvAmount, cmvAcc, invAcc); err != nil {
		t.Fatalf("falha ao anexar baixa de estoque/CMV: %v", err)
	}

	if err := entry.ValidateDoubleEntry(); err != nil {
		t.Fatalf("lançamento comercial composto (Receita + CMV/Estoque) deve respeitar as partidas dobradas: %v", err)
	}
	// Total D/C composto: 22.500,00 (Venda) + 13.500,00 (CMV/Estoque) = 36.000,00 em 4 partidas
	if len(entry.Lines) != 4 || !entry.TotalDebit.Equal(decimal.RequireFromString("36000.00")) || !entry.TotalCredit.Equal(decimal.RequireFromString("36000.00")) {
		t.Errorf("totais do lançamento composto incorretos: linhas=%d, D=%s, C=%s", len(entry.Lines), entry.TotalDebit, entry.TotalCredit)
	}
}

