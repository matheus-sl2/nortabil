package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/middleware"
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
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	companyID := middleware.GetCompanyID(r)
	acc, err := h.service.CreateAccount(r.Context(), companyID, input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, acc)
}

// List lista as contas contábeis do plano
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	isTree := r.URL.Query().Get("tree") == "true"
	activeOnly := r.URL.Query().Get("active_only") == "true"

	if isTree {
		tree, err := h.service.GetAccountTree(r.Context(), companyID)
		if err != nil {
			HandleDomainError(w, err)
			return
		}
		JSON(w, http.StatusOK, tree)
		return
	}

	accounts, err := h.service.ListAccounts(r.Context(), companyID, activeOnly)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, accounts)
}

// GetByID busca conta pelo UUID
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
func (h *AccountHandler) GetByCode(w http.ResponseWriter, r *http.Request) {
	companyID := middleware.GetCompanyID(r)
	codeParam := chi.URLParam(r, "code")
	acc, err := h.service.GetByCode(r.Context(), companyID, codeParam)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, acc)
}
