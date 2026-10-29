package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// AccountHandler processa requisições HTTP do Plano de Contas
type AccountHandler struct {
	service *service.AccountService
}

// NewAccountHandler cria uma nova instância do handler de contas
func NewAccountHandler(service *service.AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

// Create processa a criação de uma nova conta contábil
// @Summary Criar conta contábil
// @Description Cria uma nova conta no plano de contas (sintética ou analítica)
// @Tags Contas
// @Accept json
// @Produce json
// @Success 201 {object} domain.Account
// @Router /api/v1/accounts [post]
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	acc, err := h.service.CreateAccount(r.Context(), input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, acc)
}

// List lista as contas contábeis do plano
// @Summary Listar contas contábeis
// @Description Retorna as contas cadastradas de forma plana ou em árvore hierárquica
// @Tags Contas
// @Produce json
// @Param tree query bool false "Se true, retorna a estrutura hierárquica em árvore"
// @Param active_only query bool false "Se true, filtra apenas contas ativas"
// @Success 200 {array} domain.Account
// @Router /api/v1/accounts [get]
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	isTree := r.URL.Query().Get("tree") == "true"
	activeOnly := r.URL.Query().Get("active_only") == "true"

	if isTree {
		tree, err := h.service.GetAccountTree(r.Context())
		if err != nil {
			HandleDomainError(w, err)
			return
		}
		JSON(w, http.StatusOK, tree)
		return
	}

	accounts, err := h.service.ListAccounts(r.Context(), activeOnly)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, accounts)
}

// GetByID busca conta pelo UUID
// @Summary Obter conta por ID
// @Tags Contas
// @Produce json
// @Param id path string true "UUID da conta"
// @Success 200 {object} domain.Account
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Identificador UUID da conta inválido", nil)
		return
	}

	acc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, acc)
}

// GetByCode busca conta pelo código estruturado
// @Summary Obter conta por Código
// @Tags Contas
// @Produce json
// @Param code path string true "Código da conta (ex: 1.1.01.001)"
// @Success 200 {object} domain.Account
// @Router /api/v1/accounts/code/{code} [get]
func (h *AccountHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	codeParam := chi.URLParam(r, "code")
	acc, err := h.service.GetByCode(r.Context(), codeParam)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, acc)
}

