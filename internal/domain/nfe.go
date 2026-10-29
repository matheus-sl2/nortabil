package domain

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	// ErrInvalidNFeXML ocorre quando o conteúdo XML não corresponde ao leiaute padrão SEFAZ NF-e (Mod. 55) ou NFC-e (Mod. 65).
	ErrInvalidNFeXML = errors.New("arquivo XML fiscal inválido ou fora do padrão SEFAZ (NF-e Modelo 55 / NFC-e Modelo 65)")
)

// NFeDocument representa os dados fiscais e contábeis extraídos de uma NF-e (Modelo 55 - Atacado) ou NFC-e (Modelo 65 - Varejo PDV)
type NFeDocument struct {
	AccessKey       string          `json:"access_key"`
	Model           string          `json:"model"` // "55" (NF-e) ou "65" (NFC-e Varejo)
	Number          string          `json:"number"`
	Series          string          `json:"series"`
	IssueDate       time.Time       `json:"issue_date"`
	OperationNature string          `json:"operation_nature"`
	OperationType   string          `json:"operation_type"` // 0 = Entrada, 1 = Saída
	EmitterCNPJ     string          `json:"emitter_cnpj"`
	EmitterName     string          `json:"emitter_name"`
	RecipientCNPJ   string          `json:"recipient_cnpj"`
	RecipientName   string          `json:"recipient_name"`
	TotalProducts   decimal.Decimal `json:"total_products"`
	TotalInvoice    decimal.Decimal `json:"total_invoice"`
	TotalICMS       decimal.Decimal `json:"total_icms"`
	TotalDiscount   decimal.Decimal `json:"total_discount"`
	ItemsSummary    string          `json:"items_summary"`
}

// NFeImportResult representa o resultado da contabilização automática de uma NF-e / NFC-e no Livro Diário
type NFeImportResult struct {
	Document     *NFeDocument    `json:"document"`
	Direction    string          `json:"direction"` // SAIDA_VENDA, SAIDA_PDV_NFCE ou ENTRADA_COMPRA
	CMVAmount    decimal.Decimal `json:"cmv_amount"`
	JournalEntry *JournalEntry   `json:"journal_entry"`
}

// Estruturas internas para decodificação do XML oficial da SEFAZ (<nfeProc> ou <NFe>)
type xmlNFeProc struct {
	XMLName xml.Name   `xml:"nfeProc"`
	NFe     xmlNFe     `xml:"NFe"`
	ProtNFe xmlProtNFe `xml:"protNFe"`
}

type xmlProtNFe struct {
	InfProt struct {
		ChNFe string `xml:"chNFe"`
	} `xml:"infProt"`
}

type xmlNFe struct {
	XMLName xml.Name  `xml:"NFe"`
	InfNFe  xmlInfNFe `xml:"infNFe"`
}

type xmlInfNFe struct {
	ID  string `xml:"Id,attr"`
	Ide struct {
		Mod   string `xml:"mod"`
		Serie string `xml:"serie"`
		NNF   string `xml:"nNF"`
		DhEmi string `xml:"dhEmi"`
		DEmi  string `xml:"dEmi"`
		TpNF  string `xml:"tpNF"`
		NatOp string `xml:"natOp"`
	} `xml:"ide"`
	Emit struct {
		CNPJ  string `xml:"CNPJ"`
		CPF   string `xml:"CPF"`
		XNome string `xml:"xNome"`
	} `xml:"emit"`
	Dest struct {
		CNPJ  string `xml:"CNPJ"`
		CPF   string `xml:"CPF"`
		XNome string `xml:"xNome"`
	} `xml:"dest"`
	Det []struct {
		Prod struct {
			CProd string `xml:"cProd"`
			XProd string `xml:"xProd"`
			CFOP  string `xml:"CFOP"`
			VProd string `xml:"vProd"`
		} `xml:"prod"`
	} `xml:"det"`
	Total struct {
		ICMSTot struct {
			VProd string `xml:"vProd"`
			VICMS string `xml:"vICMS"`
			VDesc string `xml:"vDesc"`
			VNF   string `xml:"vNF"`
		} `xml:"ICMSTot"`
	} `xml:"total"`
}

// ParseNFeXML analisa um documento XML de NF-e (Modelo 55) ou NFC-e (Modelo 65) e retorna a estrutura fiscal validada
func ParseNFeXML(rawXML string) (*NFeDocument, error) {
	cleaned := strings.TrimSpace(rawXML)
	if cleaned == "" {
		return nil, ErrInvalidNFeXML
	}

	var inf xmlInfNFe
	var accessKey string

	if strings.Contains(cleaned, "<nfeProc") {
		var proc xmlNFeProc
		if err := xml.Unmarshal([]byte(cleaned), &proc); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidNFeXML, err)
		}
		inf = proc.NFe.InfNFe
		accessKey = strings.TrimSpace(proc.ProtNFe.InfProt.ChNFe)
	} else {
		var nfe xmlNFe
		if err := xml.Unmarshal([]byte(cleaned), &nfe); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidNFeXML, err)
		}
		inf = nfe.InfNFe
	}

	if accessKey == "" && inf.ID != "" {
		accessKey = strings.TrimPrefix(strings.TrimSpace(inf.ID), "NFe")
	}

	if strings.TrimSpace(inf.Ide.NNF) == "" {
		return nil, fmt.Errorf("%w: número da nota fiscal (nNF) ausente", ErrInvalidNFeXML)
	}

	vNF, err := parseXMLDecimal(inf.Total.ICMSTot.VNF)
	if err != nil || !vNF.IsPositive() {
		return nil, fmt.Errorf("%w: valor total da nota fiscal (vNF) inválido ou zerado", ErrInvalidNFeXML)
	}

	vProd, _ := parseXMLDecimal(inf.Total.ICMSTot.VProd)
	if vProd.IsZero() {
		vProd = vNF
	}
	vICMS, _ := parseXMLDecimal(inf.Total.ICMSTot.VICMS)
	vDesc, _ := parseXMLDecimal(inf.Total.ICMSTot.VDesc)

	issueDate := parseNFeDate(inf.Ide.DhEmi, inf.Ide.DEmi)

	model := strings.TrimSpace(inf.Ide.Mod)
	if model == "" {
		model = "55"
	}

	emitDoc := strings.TrimSpace(inf.Emit.CNPJ)
	if emitDoc == "" {
		emitDoc = strings.TrimSpace(inf.Emit.CPF)
	}
	destDoc := strings.TrimSpace(inf.Dest.CNPJ)
	if destDoc == "" {
		destDoc = strings.TrimSpace(inf.Dest.CPF)
	}
	destName := strings.TrimSpace(inf.Dest.XNome)
	if destName == "" && model == "65" {
		destName = "CONSUMIDOR FINAL (PDV VAREJO)"
	}
	if destDoc == "" && model == "65" {
		destDoc = "CONSUMIDOR-PDV"
	}

	var items []string
	for _, d := range inf.Det {
		name := strings.TrimSpace(d.Prod.XProd)
		if name != "" {
			items = append(items, name)
		}
	}
	itemsSummary := strings.Join(items, "; ")
	if itemsSummary == "" {
		itemsSummary = "Mercadorias / Serviços conforme Documento Fiscal"
	}

	tpNF := strings.TrimSpace(inf.Ide.TpNF)
	if tpNF == "" {
		tpNF = "1"
	}

	serie := strings.TrimSpace(inf.Ide.Serie)
	if serie == "" {
		serie = "1"
	}

	return &NFeDocument{
		AccessKey:       accessKey,
		Model:           model,
		Number:          strings.TrimSpace(inf.Ide.NNF),
		Series:          serie,
		IssueDate:       issueDate,
		OperationNature: strings.TrimSpace(inf.Ide.NatOp),
		OperationType:   tpNF,
		EmitterCNPJ:     emitDoc,
		EmitterName:     strings.TrimSpace(inf.Emit.XNome),
		RecipientCNPJ:   destDoc,
		RecipientName:   destName,
		TotalProducts:   vProd,
		TotalInvoice:    vNF,
		TotalICMS:       vICMS,
		TotalDiscount:   vDesc,
		ItemsSummary:    itemsSummary,
	}, nil
}

func parseXMLDecimal(val string) (decimal.Decimal, error) {
	s := strings.TrimSpace(strings.ReplaceAll(val, ",", "."))
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

func parseNFeDate(dhEmi, dEmi string) time.Time {
	raw := strings.TrimSpace(dhEmi)
	if raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t
		}
		if len(raw) >= 10 {
			if t, err := time.Parse("2006-01-02", raw[:10]); err == nil {
				return t
			}
		}
	}
	rawDate := strings.TrimSpace(dEmi)
	if rawDate != "" {
		if t, err := time.Parse("2006-01-02", rawDate); err == nil {
			return t
		}
	}
	return time.Now().UTC()
}

// BuildAccountingEntry monta o lançamento de partidas dobradas correspondente à NF-e (Mod. 55) ou NFC-e (Mod. 65)
func (doc *NFeDocument) BuildAccountingEntry(companyID uuid.UUID, direction string, debitAccountID, creditAccountID uuid.UUID, createdBy string) (*JournalEntry, error) {
	docLabel := "NF-e"
	if doc.Model == "65" {
		docLabel = "NFC-e (PDV Varejo)"
	}

	var desc string
	if direction == "ENTRADA_COMPRA" {
		desc = fmt.Sprintf("%s Entrada nº %s (Série %s) - %s - Emitente: %s (%s) - Itens: %s",
			docLabel, doc.Number, doc.Series, doc.OperationNature, doc.EmitterName, doc.EmitterCNPJ, doc.ItemsSummary)
	} else {
		desc = fmt.Sprintf("%s Saída nº %s (Série %s) - %s - Destinatário: %s (%s) - Itens: %s",
			docLabel, doc.Number, doc.Series, doc.OperationNature, doc.RecipientName, doc.RecipientCNPJ, doc.ItemsSummary)
	}

	if doc.AccessKey != "" {
		desc = fmt.Sprintf("%s [Chave: %s]", desc, doc.AccessKey)
	}

	if createdBy == "" {
		createdBy = "Importador Fiscal NF-e/NFC-e"
	}

	entry := &JournalEntry{
		ID:          uuid.New(),
		CompanyID:   companyID,
		Date:        doc.IssueDate,
		Description: desc,
		Status:      JournalStatusPosted,
		CreatedBy:   createdBy,
		Lines: []JournalLine{
			{
				ID:          uuid.New(),
				AccountID:   debitAccountID,
				Type:        JournalLineDebit,
				Amount:      doc.TotalInvoice,
				Description: fmt.Sprintf("Débito ref. %s nº %s (%s)", docLabel, doc.Number, doc.OperationNature),
			},
			{
				ID:          uuid.New(),
				AccountID:   creditAccountID,
				Type:        JournalLineCredit,
				Amount:      doc.TotalInvoice,
				Description: fmt.Sprintf("Crédito ref. %s nº %s (%s)", docLabel, doc.Number, doc.OperationNature),
			},
		},
	}

	entry.Clean()
	if err := entry.ValidateDoubleEntry(); err != nil {
		return nil, err
	}
	return entry, nil
}

// AttachInventoryWriteOff adiciona as partidas de baixa automática de Estoque contra CMV (NBC TG 16) nas vendas do comércio
func (doc *NFeDocument) AttachInventoryWriteOff(entry *JournalEntry, cmvAmount decimal.Decimal, cmvAccountID, inventoryAccountID uuid.UUID) error {
	if !cmvAmount.IsPositive() || cmvAccountID == uuid.Nil || inventoryAccountID == uuid.Nil {
		return nil
	}

	docLabel := "NF-e"
	if doc.Model == "65" {
		docLabel = "NFC-e"
	}

	entry.Lines = append(entry.Lines,
		JournalLine{
			ID:          uuid.New(),
			AccountID:   cmvAccountID,
			Type:        JournalLineDebit,
			Amount:      cmvAmount,
			Description: fmt.Sprintf("Reconhecimento de CMV (NBC TG 16) ref. %s nº %s", docLabel, doc.Number),
		},
		JournalLine{
			ID:          uuid.New(),
			AccountID:   inventoryAccountID,
			Type:        JournalLineCredit,
			Amount:      cmvAmount,
			Description: fmt.Sprintf("Baixa automática de Estoque de Mercadorias ref. %s nº %s", docLabel, doc.Number),
		},
	)

	entry.Clean()
	return entry.ValidateDoubleEntry()
}
