package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// NFeService gerencia a importação e contabilização automática de Notas Fiscais Eletrônicas (XML SEFAZ Modelo 55)
type NFeService struct {
	companyRepo    *postgres.CompanyRepository
	accountRepo    *postgres.AccountRepository
	journalService *JournalService
}

func NewNFeService(
	companyRepo *postgres.CompanyRepository,
	accountRepo *postgres.AccountRepository,
	journalService *JournalService,
) *NFeService {
	return &NFeService{
		companyRepo:    companyRepo,
		accountRepo:    accountRepo,
		journalService: journalService,
	}
}

// ImportNFeXML processa o XML da NF-e, classifica a operação (Entrada ou Saída) e registra o lançamento em partidas dobradas
func (s *NFeService) ImportNFeXML(
	ctx context.Context,
	companyID uuid.UUID,
	rawXML string,
	debitAccountID uuid.UUID,
	creditAccountID uuid.UUID,
	createdBy string,
) (*domain.NFeImportResult, error) {
	doc, err := domain.ParseNFeXML(rawXML)
	if err != nil {
		return nil, err
	}

	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return nil, err
	}

	compDigits := onlyDigits(company.CNPJ)
	emitDigits := onlyDigits(doc.EmitterCNPJ)
	destDigits := onlyDigits(doc.RecipientCNPJ)

	// Identifica se a NF-e é de Entrada (Compra) ou Saída (Venda)
	direction := "SAIDA_VENDA"
	if (destDigits != "" && destDigits == compDigits && emitDigits != compDigits) || doc.OperationType == "0" {
		direction = "ENTRADA_COMPRA"
	}

	// Se o usuário não especificou manualmente as contas analíticas de Débito/Crédito, seleciona automaticamente pelo Plano de Contas
	if debitAccountID == uuid.Nil || creditAccountID == uuid.Nil {
		accounts, err := s.accountRepo.List(ctx, companyID, true)
		if err != nil {
			return nil, err
		}

		autoDebit, autoCredit := resolveDefaultNFeAccounts(accounts, direction)
		if debitAccountID == uuid.Nil {
			debitAccountID = autoDebit
		}
		if creditAccountID == uuid.Nil {
			creditAccountID = autoCredit
		}
	}

	if debitAccountID == uuid.Nil || creditAccountID == uuid.Nil {
		return nil, domain.ErrAccountNotFound
	}

	entry, err := doc.BuildAccountingEntry(companyID, direction, debitAccountID, creditAccountID, createdBy)
	if err != nil {
		return nil, err
	}

	createdLines := make([]CreateJournalLineInput, len(entry.Lines))
	for i, l := range entry.Lines {
		createdLines[i] = CreateJournalLineInput{
			AccountID:   l.AccountID,
			Type:        l.Type,
			Amount:      l.Amount,
			Description: l.Description,
		}
	}

	postedEntry, err := s.journalService.CreateEntry(ctx, companyID, CreateJournalEntryInput{
		CompanyID:   &companyID,
		Date:        entry.Date,
		Description: entry.Description,
		CreatedBy:   createdBy,
		Lines:       createdLines,
	})
	if err != nil {
		return nil, err
	}

	return &domain.NFeImportResult{
		Document:     doc,
		Direction:    direction,
		JournalEntry: postedEntry,
	}, nil
}

func onlyDigits(val string) string {
	var b strings.Builder
	for _, r := range val {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func resolveDefaultNFeAccounts(accounts []domain.Account, direction string) (uuid.UUID, uuid.UUID) {
	byCode := make(map[string]uuid.UUID)
	var firstAsset, firstLiability, firstRevenue, firstExpense uuid.UUID

	for _, a := range accounts {
		if a.IsSynthetic || !a.Active {
			continue
		}
		byCode[a.Code] = a.ID
		switch a.Type {
		case domain.AccountTypeAsset:
			if firstAsset == uuid.Nil {
				firstAsset = a.ID
			}
		case domain.AccountTypeLiability:
			if firstLiability == uuid.Nil {
				firstLiability = a.ID
			}
		case domain.AccountTypeRevenue:
			if firstRevenue == uuid.Nil {
				firstRevenue = a.ID
			}
		case domain.AccountTypeExpense:
			if firstExpense == uuid.Nil {
				firstExpense = a.ID
			}
		}
	}

	if direction == "ENTRADA_COMPRA" {
		// Compra de Mercadorias/Insumos: D - Estoques (1.1.03.001) | C - Fornecedores Nacionais (2.1.01.001)
		deb := byCode["1.1.03.001"]
		if deb == uuid.Nil {
			deb = firstExpense
		}
		if deb == uuid.Nil {
			deb = firstAsset
		}
		cred := byCode["2.1.01.001"]
		if cred == uuid.Nil {
			cred = byCode["1.1.01.002"]
		}
		if cred == uuid.Nil {
			cred = firstLiability
		}
		return deb, cred
	}

	// Saída / Venda de Mercadorias: D - Clientes/Duplicatas a Receber (1.1.02.001) | C - Receita de Vendas (4.1.01.001)
	deb := byCode["1.1.02.001"]
	if deb == uuid.Nil {
		deb = byCode["1.1.01.002"]
	}
	if deb == uuid.Nil {
		deb = firstAsset
	}
	cred := byCode["4.1.01.001"]
	if cred == uuid.Nil {
		cred = firstRevenue
	}
	return deb, cred
}
