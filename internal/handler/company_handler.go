package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/matheus-sl2/nortabil/internal/service"
)

// CompanyHandler processa requisições HTTP de empresas/clientes
type CompanyHandler struct {
	companyService *service.CompanyService
}

// NewCompanyHandler cria uma nova instância do handler de empresas
func NewCompanyHandler(companyService *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{companyService: companyService}
}

// Create cadastra uma nova empresa no escritório contábil
// @Summary Criar empresa / cliente
// @Tags Empresas
// @Accept json
// @Produce json
// @Success 201 {object} domain.Company
// @Router /api/v1/companies [post]
func (h *CompanyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input service.CreateCompanyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_JSON", "Corpo da requisição JSON inválido", err.Error())
		return
	}

	company, err := h.companyService.CreateCompany(r.Context(), input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusCreated, company)
}

// List lista as empresas cadastradas no escritório
// @Summary Listar empresas
// @Tags Empresas
// @Produce json
// @Param active_only query bool false "Se true, lista apenas empresas ativas"
// @Success 200 {array} domain.Company
// @Router /api/v1/companies [get]
func (h *CompanyHandler) List(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") == "true"
	companies, err := h.companyService.ListCompanies(r.Context(), activeOnly)
	if err != nil {
		HandleDomainError(w, err)
		return
	}
	JSON(w, http.StatusOK, companies)
}

// GetByID busca empresa pelo UUID
// @Summary Obter detalhes da empresa
// @Tags Empresas
// @Produce json
// @Param id path string true "UUID da empresa"
// @Success 200 {object} domain.Company
// @Router /api/v1/companies/{id} [get]
func (h *CompanyHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		JSONError(w, http.StatusBadRequest, "INVALID_UUID", "Identificador UUID da empresa inválido", nil)
		return
	}

	company, err := h.companyService.GetByID(r.Context(), id)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	JSON(w, http.StatusOK, company)
}
