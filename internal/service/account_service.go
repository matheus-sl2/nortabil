package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/domain"
	"github.com/matheus-sl2/nortabil/internal/repository/postgres"
)

// AccountService orquestra as regras de negócio para o Plano de Contas
type AccountService struct {
	repo *postgres.AccountRepository
}

// NewAccountService cria uma nova instância do serviço de contas
func NewAccountService(repo *postgres.AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

// CreateAccountInput dados de entrada para criação de conta contábil
type CreateAccountInput struct {
	Code        string             `json:"code"`
	Name        string             `json:"name"`
	Type        domain.AccountType `json:"type"`
	IsSynthetic bool               `json:"is_synthetic"`
	ParentID    *uuid.UUID         `json:"parent_id,omitempty"`
}

// CreateAccount valida e persiste uma nova conta contábil no plano de contas
func (s *AccountService) CreateAccount(ctx context.Context, input CreateAccountInput) (*domain.Account, error) {
	input.Code = domain.NormalizeAccountCode(input.Code)
	input.Name = strings.TrimSpace(input.Name)

	if input.Code == "" {
		return nil, domain.ErrInvalidAccountCode
	}
	if input.Name == "" {
		return nil, domain.ErrInvalidAccountCode
	}
	if !domain.IsValidAccountType(input.Type) {
		return nil, domain.ErrInvalidAccountType
	}

	// 1. Verifica se já existe conta com este código
	existing, err := s.repo.GetByCode(ctx, input.Code)
	if err == nil && existing != nil {
		return nil, domain.ErrAccountAlreadyExists
	}

	// 2. Validações da conta pai na hierarquia
	if input.ParentID != nil {
		parent, err := s.repo.GetByID(ctx, *input.ParentID)
		if err != nil {
			return nil, domain.ErrParentAccountNotFound
		}

		// Se a conta pai ainda não é sintética, devemos verificar se ela possui lançamentos
		if !parent.IsSynthetic {
			hasEntries, err := s.repo.HasEntries(ctx, parent.ID)
			if err != nil {
				return nil, err
			}
			if hasEntries {
				return nil, domain.ErrAccountHasEntriesCannotBeSynthetic
			}
			// Promove a conta pai para sintética
			if err := s.repo.SetSynthetic(ctx, parent.ID, true); err != nil {
				return nil, err
			}
		}

		// Força o mesmo tipo da conta pai (ex: Ativo filho de Ativo)
		input.Type = parent.Type
	}

	nature := domain.DeriveNature(input.Type)

	acc := &domain.Account{
		ID:          uuid.New(),
		Code:        input.Code,
		Name:        input.Name,
		Type:        input.Type,
		Nature:      nature,
		IsSynthetic: input.IsSynthetic,
		ParentID:    input.ParentID,
		Active:      true,
	}

	if err := s.repo.Create(ctx, acc); err != nil {
		return nil, err
	}

	return acc, nil
}

// GetByID busca conta por ID
func (s *AccountService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByCode busca conta por código estruturado
func (s *AccountService) GetByCode(ctx context.Context, code string) (*domain.Account, error) {
	return s.repo.GetByCode(ctx, domain.NormalizeAccountCode(code))
}

// ListAccounts lista todas as contas
func (s *AccountService) ListAccounts(ctx context.Context, activeOnly bool) ([]domain.Account, error) {
	return s.repo.List(ctx, activeOnly)
}

// GetAccountTree monta a representação hierárquica em árvore de todo o Plano de Contas
func (s *AccountService) GetAccountTree(ctx context.Context) ([]*domain.AccountTreeNode, error) {
	accounts, err := s.repo.List(ctx, false)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[uuid.UUID]*domain.AccountTreeNode)
	var rootNodes []*domain.AccountTreeNode

	for _, acc := range accounts {
		node := &domain.AccountTreeNode{
			Account:  acc,
			Children: []*domain.AccountTreeNode{},
		}
		nodeMap[acc.ID] = node
	}

	for _, acc := range accounts {
		node := nodeMap[acc.ID]
		if acc.ParentID == nil || *acc.ParentID == uuid.Nil {
			rootNodes = append(rootNodes, node)
		} else {
			if parentNode, exists := nodeMap[*acc.ParentID]; exists {
				parentNode.Children = append(parentNode.Children, node)
			} else {
				rootNodes = append(rootNodes, node)
			}
		}
	}

	return rootNodes, nil
}
