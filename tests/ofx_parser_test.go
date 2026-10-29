package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/matheus-sl2/nortabil/internal/domain"
)

// TestOFXParserSGML testa a interpretação de arquivos OFX em formato SGML bancário brasileiro tradicional
func TestOFXParserSGML(t *testing.T) {
	ofxSGML := `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<SIGNONMSGSRSV1>
<SONRS>
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<DTSERVER>20261029120000[-03:EST]
<LANGUAGE>POR
</SONRS>
</SIGNONMSGSRSV1>
<BANKMSGSRSV1>
<STMTTRNRS>
<TRNUID>1001
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<STMTRS>
<CURDEF>BRL
<BANKACCTFROM>
<BANKID>001
<BRANCHID>1234
<ACCTID>98765-4
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20261001000000
<DTEND>20261031235959
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20261005120000[-03:EST]
<TRNAMT>5450.75
<FITID>2026100500198765
<CHECKNUM>100234
<MEMO>PIX RECEBIDO CLIENTE MONTE CARMELO
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20261010143000[-03:EST]
<TRNAMT>-1250.30
<FITID>2026101000198766
<CHECKNUM>554433
<MEMO>PGTO FORNECEDOR CEMIG ENERGIA
</STMTTRN>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20261015090000[-03:EST]
<TRNAMT>-89.90
<FITID>2026101500198767
<MEMO>TARIFA BANCARIA CESTA PJ
</STMTTRN>
</BANKTRANLIST>
<LEDGERBAL>
<BALAMT>15890.55
<DTASOF>20261031235959
</LEDGERBAL>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>
`

	stmt, err := domain.ParseOFX(strings.NewReader(ofxSGML))
	if err != nil {
		t.Fatalf("Erro inesperado ao fazer parsing de OFX SGML: %v", err)
	}

	if stmt.BankID != "001" {
		t.Errorf("Esperava BankID '001', obteve '%s'", stmt.BankID)
	}
	if stmt.BranchID != "1234" {
		t.Errorf("Esperava BranchID '1234', obteve '%s'", stmt.BranchID)
	}
	if stmt.AccountID != "98765-4" {
		t.Errorf("Esperava AccountID '98765-4', obteve '%s'", stmt.AccountID)
	}
	if stmt.Currency != "BRL" {
		t.Errorf("Esperava Moeda BRL, obteve '%s'", stmt.Currency)
	}

	if len(stmt.Transactions) != 3 {
		t.Fatalf("Esperava 3 transações no extrato, obteve %d", len(stmt.Transactions))
	}

	// Transação 1: Crédito
	trn1 := stmt.Transactions[0]
	if trn1.FITID != "2026100500198765" {
		t.Errorf("Trn1 FITID incorreto: %s", trn1.FITID)
	}
	expectedAmount1 := decimal.NewFromFloat(5450.75)
	if !trn1.Amount.Equal(expectedAmount1) {
		t.Errorf("Trn1 Amount incorreto: esperado %s, obteve %s", expectedAmount1, trn1.Amount)
	}
	if trn1.Type != "CREDIT" {
		t.Errorf("Trn1 Type incorreto: esperado CREDIT, obteve %s", trn1.Type)
	}
	if trn1.Memo != "PIX RECEBIDO CLIENTE MONTE CARMELO" {
		t.Errorf("Trn1 Memo incorreto: %s", trn1.Memo)
	}
	if trn1.Date.Year() != 2026 || trn1.Date.Month() != 10 || trn1.Date.Day() != 5 {
		t.Errorf("Trn1 Data incorreta: %v", trn1.Date)
	}

	// Transação 2: Débito
	trn2 := stmt.Transactions[1]
	if trn2.FITID != "2026101000198766" {
		t.Errorf("Trn2 FITID incorreto: %s", trn2.FITID)
	}
	expectedAmount2 := decimal.NewFromFloat(-1250.30)
	if !trn2.Amount.Equal(expectedAmount2) {
		t.Errorf("Trn2 Amount incorreto: esperado %s, obteve %s", expectedAmount2, trn2.Amount)
	}
	if trn2.Type != "DEBIT" {
		t.Errorf("Trn2 Type incorreto: esperado DEBIT, obteve %s", trn2.Type)
	}
	if trn2.CheckNum != "554433" {
		t.Errorf("Trn2 CheckNum incorreto: esperado 554433, obteve %s", trn2.CheckNum)
	}

	// Transação 3: Débito Tarifa
	trn3 := stmt.Transactions[2]
	if trn3.FITID != "2026101500198767" {
		t.Errorf("Trn3 FITID incorreto: %s", trn3.FITID)
	}
	expectedAmount3 := decimal.NewFromFloat(-89.90)
	if !trn3.Amount.Equal(expectedAmount3) {
		t.Errorf("Trn3 Amount incorreto: esperado %s, obteve %s", expectedAmount3, trn3.Amount)
	}
}

// TestOFXParserXML testa a interpretação de extratos OFX em formato XML com tags fechadas
func TestOFXParserXML(t *testing.T) {
	ofxXML := `<?xml version="1.0" encoding="utf-8"?>
<OFX>
  <BANKMSGSRSV1>
    <STMTTRNRS>
      <STMTRS>
        <CURDEF>BRL</CURDEF>
        <BANKACCTFROM>
          <BANKID>341</BANKID>
          <BRANCHID>0432</BRANCHID>
          <ACCTID>12345-6</ACCTID>
        </BANKACCTFROM>
        <BANKTRANLIST>
          <STMTTRN>
            <TRNTYPE>DEBIT</TRNTYPE>
            <DTPOSTED>20261020</DTPOSTED>
            <TRNAMT>-3200,50</TRNAMT>
            <FITID>ITAU-20261020-0099</FITID>
            <MEMO>FOLHA DE PAGAMENTO OUTUBRO</MEMO>
          </STMTTRN>
        </BANKTRANLIST>
      </STMTRS>
    </STMTTRNRS>
  </BANKMSGSRSV1>
</OFX>`

	stmt, err := domain.ParseOFX(strings.NewReader(ofxXML))
	if err != nil {
		t.Fatalf("Erro inesperado ao fazer parsing de OFX XML: %v", err)
	}

	if stmt.BankID != "341" {
		t.Errorf("Esperava BankID '341', obteve '%s'", stmt.BankID)
	}
	if len(stmt.Transactions) != 1 {
		t.Fatalf("Esperava 1 transação, obteve %d", len(stmt.Transactions))
	}

	trn := stmt.Transactions[0]
	if trn.FITID != "ITAU-20261020-0099" {
		t.Errorf("FITID incorreto: %s", trn.FITID)
	}
	expectedAmount := decimal.NewFromFloat(-3200.50)
	if !trn.Amount.Equal(expectedAmount) {
		t.Errorf("Valor com vírgula decimal não interpretado corretamente: esperado %s, obteve %s", expectedAmount, trn.Amount)
	}
	if trn.Memo != "FOLHA DE PAGAMENTO OUTUBRO" {
		t.Errorf("Memo incorreto: %s", trn.Memo)
	}
	if trn.Date.Format("2006-01-02") != "2026-10-20" {
		t.Errorf("Data incorreta: %s", trn.Date.Format("2006-01-02"))
	}
}

// TestOFXParseDateFormats testa as variações de datas encontradas em bancos do Brasil
func TestOFXParseDateFormats(t *testing.T) {
	// Cria mock simples de transação para cada formato
	checkDate := func(rawDate string, expectedYear int, expectedMonth time.Month, expectedDay int) {
		ofxMock := "<OFX>\n<BANKTRANLIST>\n<STMTTRN>\n<FITID>1</FITID>\n<DTPOSTED>" + rawDate + "</DTPOSTED>\n</STMTTRN>\n</BANKTRANLIST>\n</OFX>\n"
		stmt, err := domain.ParseOFX(strings.NewReader(ofxMock))
		if err != nil || len(stmt.Transactions) == 0 {
			t.Fatalf("Falha ao processar data '%s': %v (transações: %d)", rawDate, err, len(stmt.Transactions))
		}
		d := stmt.Transactions[0].Date
		if d.Year() != expectedYear || d.Month() != expectedMonth || d.Day() != expectedDay {
			t.Errorf("Data '%s' interpretada erroneamente: ano %d, mes %v, dia %d", rawDate, d.Year(), d.Month(), d.Day())
		}
	}

	checkDate("20261029143000[-03:EST]", 2026, time.October, 29)
	checkDate("20261029", 2026, time.October, 29)
	checkDate("2026-10-29", 2026, time.October, 29)
}
